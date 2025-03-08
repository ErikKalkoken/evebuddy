// Evebuddy is a companion app for Eve Online players.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"maps"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"
	"github.com/antihax/goesi"
	"github.com/chasinglogic/appdirs"
	"github.com/gregjones/httpcache"
	"github.com/hashicorp/go-retryablehttp"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/ErikKalkoken/evebuddy/internal/app/character"
	"github.com/ErikKalkoken/evebuddy/internal/app/esistatus"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverse"
	"github.com/ErikKalkoken/evebuddy/internal/app/pcache"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	uidesktop "github.com/ErikKalkoken/evebuddy/internal/app/ui/desktop"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/mobile"
	"github.com/ErikKalkoken/evebuddy/internal/deleteapp"
	"github.com/ErikKalkoken/evebuddy/internal/eveimage"
	"github.com/ErikKalkoken/evebuddy/internal/memcache"
	"github.com/ErikKalkoken/evebuddy/internal/sso"
)

const (
	appID               = "io.github.erikkalkoken.evebuddy"
	appName             = "evebuddy"
	cacheCleanUpTimeout = time.Minute * 30
	crashFileName       = "crash.txt"
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
	deleteDataFlag     = flag.Bool("delete-data", false, "Delete user data")
	developFlag        = flag.Bool("dev", false, "Enable developer features")
	dirsFlag           = flag.Bool("dirs", false, "Show directories for user data")
	disableUpdatesFlag = flag.Bool("disable-updates", false, "Disable all periodic updates")
	offlineFlag        = flag.Bool("offline", false, "Start app in offline mode")
	pprofFlag          = flag.Bool("pprof", false, "Enable pprof web server")
	versionFlag        = flag.Bool("v", false, "Show version")
	logLevelFlag       = flag.String("log-level", "", "Set log level for this session")
	resetSettingsFlag  = flag.Bool("reset-settings", false, "Resets desktop settings")
)

func main() {
	// init log & flags
	slog.SetLogLoggerLevel(logLevelDefault)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	flag.Parse()

	// set manual log level for this session if requested
	if v := *logLevelFlag; v != "" {
		m := map[string]slog.Level{
			"debug": slog.LevelDebug,
			"info":  slog.LevelInfo,
			"warn":  slog.LevelWarn,
			"error": slog.LevelError,
		}
		l, ok := m[strings.ToLower(v)]
		if !ok {
			fmt.Println("valid log levels are: ", strings.Join(slices.Collect(maps.Keys(m)), ", "))
			os.Exit(1)
		}
		slog.SetLogLoggerLevel(l)
	}

	// start fyne app
	fyneApp := app.NewWithID(appID)
	_, isDesktop := fyneApp.(desktop.App)

	if *versionFlag {
		fmt.Println(fyneApp.Metadata().Version)
		return
	}

	// set log level from settings
	if *logLevelFlag == "" {
		ln := fyneApp.Preferences().StringWithFallback(ui.SettingLogLevel, ui.SettingLogLevelDefault)
		l := ui.LogLevelName2Level(ln)
		if l != logLevelDefault {
			slog.Info("Setting log level", "level", ln)
			slog.SetLogLoggerLevel(l)
		}
	}

	var dataDir string

	// data dir
	if isDesktop || *developFlag {
		ad := appdirs.New(appName)
		dataDir = ad.UserData()
		if err := os.MkdirAll(dataDir, os.ModePerm); err != nil {
			log.Fatal(err)
		}
	} else {
		dataDir = fyneApp.Storage().RootURI().Path()
	}

	if *dirsFlag {
		fmt.Println(dataDir)
		fmt.Println(fyneApp.Storage().RootURI().Path())
		return
	}

	// desktop related init
	if isDesktop {
		// start uninstall app if requested
		if *deleteDataFlag {
			u := deleteapp.NewUI(fyneApp)
			u.DataDir = dataDir
			u.ShowAndRun()
			return
		}
	}

	// setup logfile for desktop
	logDir := filepath.Join(dataDir, logFolderName)
	if err := os.MkdirAll(logDir, os.ModePerm); err != nil {
		log.Fatal(err)
	}
	logFilePath := filepath.Join(logDir, logFileName)
	logger := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    logMaxSizeMB,
		MaxBackups: logMaxBackups,
	}
	multi := io.MultiWriter(os.Stderr, logger)
	log.SetOutput(multi)

	if isDesktop {
		// ensure single instance
		mu, err := ensureSingleInstance()
		if err != nil {
			log.Fatal(err)
		}
		defer mu.Release()
	}

	crashFilePath := setupCrashFile(logDir)

	// start pprof web server
	if *pprofFlag {
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	// init database
	dbPath := filepath.Join(dataDir, dbFileName)
	dsn := "file:///" + filepath.ToSlash(dbPath)
	db, err := storage.InitDB(dsn)
	if err != nil {
		slog.Error("Failed to initialize database", "dsn", dsn, "error", err)
		os.Exit(1)
	}
	defer db.Close()
	st := storage.New(db)

	// Initialize caches
	memCache := memcache.New()
	defer memCache.Close()
	pc := pcache.New(st, cacheCleanUpTimeout)
	defer pc.Close()

	// TODO: Add response logging for DEBUG log level
	// Initialize shared HTTP client
	// Automatically retries on connection and most server errors
	// Logs requests on debug level and all HTTP error responses as warnings
	rhc := retryablehttp.NewClient()
	rhc.HTTPClient.Transport = httpcache.NewTransport(newCacheAdapter(pc, "httpcache-", 24*time.Hour))
	rhc.Logger = slog.Default()
	rhc.ResponseLogHook = func(l retryablehttp.Logger, r *http.Response) {
		if r.StatusCode >= 400 {
			slog.Warn("HTTP error response", "method", r.Request.Method, "url", r.Request.URL, "status", r.Status)
		}
	}

	// Initialize shared ESI client
	esiClient := goesi.NewAPIClient(rhc.StandardClient(), userAgent)

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
	cs := character.New(st, rhc.StandardClient(), esiClient)
	cs.EveNotificationService = en
	cs.EveUniverseService = eu
	cs.StatusCacheService = sc
	ssoService := sso.New(ssoClientID, rhc.StandardClient())
	ssoService.OpenURL = fyneApp.OpenURL
	cs.SSOService = ssoService

	// Init UI
	ess := esistatus.New(esiClient)
	eis := eveimage.New(pc, rhc.StandardClient(), *offlineFlag)
	bu := ui.NewBaseUI(fyneApp)
	bu.CacheService = memCache
	bu.CharacterService = cs
	bu.ESIStatusService = ess
	bu.EveImageService = eis
	bu.EveUniverseService = eu
	bu.StatusCacheService = sc
	bu.IsOffline = *offlineFlag
	bu.IsUpdateTickerDisabled = *disableUpdatesFlag
	bu.DataPaths = map[string]string{
		"db":        dbPath,
		"log":       logFilePath,
		"crashfile": crashFilePath,
	}
	if isDesktop {
		u := uidesktop.NewDesktopUI(bu)
		u.Init()
		if *resetSettingsFlag {
			u.ResetDesktopSettings()
		}
		u.ShowAndRun()
	} else {
		u := mobile.NewMobileUI(bu)
		u.Init()
		u.ShowAndRun()
	}
}

func setupCrashFile(logDir string) (path string) {
	path = filepath.Join(logDir, crashFileName)
	crashFile, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		slog.Error("Failed to open crash report file", "error", err)
		return
	}
	if err := debug.SetCrashOutput(crashFile, debug.CrashOptions{}); err != nil {
		slog.Error("Failed to setup crash report", "error", err)
	}
	crashFile.Close()
	return
}
