// Evebuddy is a companion app for Eve Online players.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"
	"github.com/antihax/goesi"
	"github.com/chasinglogic/appdirs"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/ErikKalkoken/evebuddy/internal/app/character"
	"github.com/ErikKalkoken/evebuddy/internal/app/esistatus"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverse"
	"github.com/ErikKalkoken/evebuddy/internal/app/pcache"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/cache"
	"github.com/ErikKalkoken/evebuddy/internal/deleteapp"
	"github.com/ErikKalkoken/evebuddy/internal/eveimage"
	"github.com/ErikKalkoken/evebuddy/internal/httptransport"
	"github.com/ErikKalkoken/evebuddy/internal/sso"
)

const (
	appID               = "io.github.erikkalkoken.evebuddy"
	appName             = "evebuddy"
	cacheCleanUpTimeout = time.Minute * 30
	dbFileName          = appName + ".sqlite"
	logFileName         = appName + ".log"
	logFolderName       = "log"
	logLevelDefault     = slog.LevelWarn // for startup only
	logMaxBackups       = 3
	logMaxSizeMB        = 50
	ssoClientID         = "11ae857fe4d149b2be60d875649c05f1"
	userAgent           = "EveBuddy kalkoken87@gmail.com"
)

// define flags
var (
	deleteAppFlag              = flag.Bool("delete-data", false, "Delete user data")
	dirsFlag                   = flag.Bool("dirs", false, "Show directories for user data")
	isUpdateTickerDisabledFlag = flag.Bool("disable-updates", false, "Disable all periodic updates")
	isOfflineFlag              = flag.Bool("offline", false, "Start app in offline mode")
	pprofFlag                  = flag.Bool("pprof", false, "Enable pprof web server")
)

func main() {
	// init log & flags
	slog.SetLogLoggerLevel(logLevelDefault)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	flag.Parse()

	// start fyne app
	fyneApp := app.NewWithID(appID)
	_, isDesktop := fyneApp.(desktop.App)

	// set log level
	ln := fyneApp.Preferences().StringWithFallback(ui.SettingLogLevel, ui.SettingLogLevelDefault)
	l := ui.LogLevelName2Level(ln)
	if l != logLevelDefault {
		slog.Info("Setting log level", "level", ln)
		slog.SetLogLoggerLevel(l)
	}

	var dataDir, logDir string
	// desktop related init
	if isDesktop {
		// data dir
		ad := appdirs.New(appName)
		dataDir = ad.UserData()
		if err := os.MkdirAll(dataDir, os.ModePerm); err != nil {
			log.Fatal(err)
		}

		// start uninstall app if requested
		if *deleteAppFlag {
			u := deleteapp.NewUI(fyneApp)
			u.DataDir = dataDir
			u.ShowAndRun()
			return
		}

		// setup logfile for desktop
		logDir = filepath.Join(dataDir, logFolderName)
		if err := os.MkdirAll(logDir, os.ModePerm); err != nil {
			log.Fatal(err)
		}
		logger := &lumberjack.Logger{
			Filename:   fmt.Sprintf("%s/%s", logDir, logFileName),
			MaxSize:    logMaxSizeMB,
			MaxBackups: logMaxBackups,
		}
		multi := io.MultiWriter(os.Stderr, logger)
		log.SetOutput(multi)

		// ensure single instance
		mu, err := ensureSingleInstance()
		if err != nil {
			log.Fatal(err)
		}
		defer mu.Release()

		// setup crash reporting
		crashFile, err := os.Create(filepath.Join(logDir, "crash.txt"))
		if err != nil {
			slog.Error("Failed to create crash report file", "error", err)
		}
		defer crashFile.Close()
		if err := debug.SetCrashOutput(crashFile, debug.CrashOptions{}); err != nil {
			slog.Error("Failed to setup crash report", "error", err)
		}
	}

	if *dirsFlag {
		fmt.Println(dataDir)
		fmt.Println(fyneApp.Storage().RootURI().Path())
		return
	}

	// init database
	var dbPath string
	if isDesktop {
		dbPath = fmt.Sprintf("%s/%s", dataDir, dbFileName)
	} else {
		// EXPERIMENTAL
		dbPath = ensureFileExists(fyneApp.Storage(), dbFileName)
	}
	db, err := storage.InitDB("file://" + dbPath)
	if err != nil {
		slog.Error("Failed to initialize database", "dsn", dbPath, "error", err)
		os.Exit(1)
	}
	defer db.Close()
	st := storage.New(db)

	// init HTTP client, ESI client and cache
	httpClient := &http.Client{
		Transport: httptransport.LoggedTransport{
			// tokens must not be logged
			BlacklistedResponseURLs: []string{"login.eveonline.com/v2/oauth/token"},
		},
	}
	esiHttpClient := &http.Client{
		Transport: httptransport.LoggedTransportWithRetries{
			MaxRetries: 3,
			StatusCodesToRetry: []int{
				http.StatusBadGateway,
				http.StatusGatewayTimeout,
				http.StatusServiceUnavailable,
			},
		},
	}
	esiClient := goesi.NewAPIClient(esiHttpClient, userAgent)
	memCache := cache.New()

	// Init StatusCache service
	sc := statuscache.New(memCache)
	if err := sc.InitCache(context.TODO(), st); err != nil {
		slog.Error("Failed to init cache", "error", err)
		os.Exit(1)
	}
	// Init EveUniverse service
	eu := eveuniverse.New(st, esiClient)
	eu.StatusCacheService = sc

	// Init EveNotification service
	en := evenotification.New()
	en.EveUniverseService = eu

	// Init Character service
	cs := character.New(st, httpClient, esiClient)
	cs.EveNotificationService = en
	cs.EveUniverseService = eu
	cs.StatusCacheService = sc
	ssoService := sso.New(ssoClientID, httpClient)
	ssoService.OpenURL = fyneApp.OpenURL
	cs.SSOService = ssoService

	// PCache init
	pc := pcache.New(st, cacheCleanUpTimeout)
	go pc.CleanUp()

	// Init UI
	u := ui.NewUI(fyneApp)
	slog.Debug("ui instance created")
	u.CacheService = memCache
	u.CharacterService = cs
	u.ESIStatusService = esistatus.New(esiClient)
	u.EveImageService = eveimage.New(pc, httpClient, *isOfflineFlag)
	u.EveUniverseService = eu
	u.StatusCacheService = sc
	u.IsOffline = *isOfflineFlag
	u.IsUpdateTickerDisabled = *isUpdateTickerDisabledFlag
	u.DataPaths = map[string]string{
		"db":  dbPath,
		"log": logDir,
	}
	u.Init()
	slog.Debug("ui initialized")

	// start pprof web server
	if *pprofFlag {
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	// Start app
	u.ShowAndRun()
}

func ensureFileExists(st fyne.Storage, name string) string {
	var p string
	u, err := st.Open(name)
	if err != nil {
		u, err := st.Create(name)
		if err != nil {
			log.Fatal(err)
		}
		p = u.URI().Path()
		u.Close()
		log.Println("created new file: ", p)
	} else {
		p = u.URI().Path()
		u.Close()
		log.Println("found existing file: ", p)
	}
	return p
}
