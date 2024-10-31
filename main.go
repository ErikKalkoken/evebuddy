// Evebuddy is a companion app for Eve Online players.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"fyne.io/fyne/v2/app"
	"github.com/antihax/goesi"
	"github.com/juju/mutex/v2"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/ErikKalkoken/evebuddy/internal/app/character"
	"github.com/ErikKalkoken/evebuddy/internal/app/esistatus"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverse"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/appdirs"
	"github.com/ErikKalkoken/evebuddy/internal/cache"
	"github.com/ErikKalkoken/evebuddy/internal/deleteapp"
	"github.com/ErikKalkoken/evebuddy/internal/eveimage"
	"github.com/ErikKalkoken/evebuddy/internal/httptransport"
	"github.com/ErikKalkoken/evebuddy/internal/sso"
)

const (
	appID         = "io.github.erikkalkoken.evebuddy"
	dbFileName    = "evebuddy.sqlite"
	logFileName   = "evebuddy.log"
	logMaxBackups = 3
	logMaxSizeMB  = 50
	mutexDelay    = 100 * time.Millisecond
	mutexTimeout  = 250 * time.Millisecond
	ssoClientID   = "11ae857fe4d149b2be60d875649c05f1"
	userAgent     = "EveBuddy kalkoken87@gmail.com"
)

type logLevelFlag struct {
	value slog.Level
}

func (l logLevelFlag) String() string {
	return l.value.String()
}

func (l *logLevelFlag) Set(value string) error {
	m := map[string]slog.Level{
		"DEBUG": slog.LevelDebug,
		"INFO":  slog.LevelInfo,
		"WARN":  slog.LevelWarn,
		"ERROR": slog.LevelError,
	}
	v, ok := m[strings.ToUpper(value)]
	if !ok {
		return fmt.Errorf("unknown log level")
	}
	l.value = v
	return nil
}

// defined flags
var (
	levelFlag                  logLevelFlag
	deleteAppFlag              = flag.Bool("delete-data", false, "Delete user data")
	isUpdateTickerDisabledFlag = flag.Bool("disable-updates", false, "Disable all periodic updates")
	isOfflineFlag              = flag.Bool("offline", false, "Start app in offline mode")
	pprofFlag                  = flag.Bool("pprof", false, "Enable pprof web server")
)

func init() {
	levelFlag.value = slog.LevelInfo
	flag.Var(&levelFlag, "loglevel", "set log level")
}

type realtime struct{}

func (r realtime) After(d time.Duration) <-chan time.Time {
	c := make(chan time.Time)
	go func() {
		time.Sleep(d)
		c <- time.Now()
	}()
	return c
}

func (r realtime) Now() time.Time {
	return time.Now()
}

func main() {
	// init dirs & flags
	ad, err := appdirs.New()
	if err != nil {
		log.Fatal(err)
	}
	flag.Parse()

	// setup logging
	slog.SetLogLoggerLevel(levelFlag.value)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	logger := &lumberjack.Logger{
		Filename:   fmt.Sprintf("%s/%s", ad.Log, logFileName),
		MaxSize:    logMaxSizeMB,
		MaxBackups: logMaxBackups,
	}
	log.SetOutput(logger)

	// ensure only one instance is running
	slog.Info("Checking for other instances")
	r, err := mutex.Acquire(mutex.Spec{
		Name:    strings.ReplaceAll(appID, ".", "-"),
		Clock:   realtime{},
		Delay:   mutexDelay,
		Timeout: mutexTimeout,
	})
	if errors.Is(err, mutex.ErrTimeout) {
		slog.Warn("Attempted to run an additional instance. Shutting down that process.")
		fmt.Println("There is already an instance running")
		os.Exit(1)
	} else if err != nil {
		log.Fatal(err)
	}
	defer r.Release()
	slog.Info("No other instances running")

	// start fyne app
	fyneApp := app.NewWithID(appID)
	ad.SetSettings(fyneApp.Storage().RootURI().Path())

	// start uninstall app if requested
	if *deleteAppFlag {
		log.SetOutput(os.Stderr)
		if err := logger.Close(); err != nil {
			slog.Error("Failed to close log file", "error", err)
		}
		u := deleteapp.NewUI(fyneApp, ad)
		u.ShowAndRun()
		return
	}

	// setup crash reporting
	f, err := os.Create(filepath.Join(ad.Log, "crash.txt"))
	if err != nil {
		log.Fatal(err)
	}
	if err := debug.SetCrashOutput(f, debug.CrashOptions{}); err != nil {
		log.Fatal(err)
	}

	// init database
	dsn := fmt.Sprintf("file:%s/%s", ad.Data, dbFileName)
	db, err := storage.InitDB(dsn)
	if err != nil {
		log.Fatalf("Failed to initialize database %s: %s", dsn, err)
	}
	defer db.Close()
	st := storage.New(db)

	// init HTTP client, ESI client and cache
	httpClient := &http.Client{
		Transport: httptransport.LoggedTransport{
			BlockedResponseURLs: []string{"login.eveonline.com/v2/oauth/token"},
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
	cache := cache.New()

	// Init StatusCache service
	sc := statuscache.New(cache)
	if err := sc.InitCache(context.TODO(), st); err != nil {
		log.Fatal(err)
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
	cs.SSOService = sso.New(ssoClientID, httpClient, cache)

	// Init UI
	u := ui.NewUI(fyneApp, ad)
	slog.Debug("ui instance created")
	u.CacheService = cache
	u.CharacterService = cs
	u.ESIStatusService = esistatus.New(esiClient)
	u.EveImageService = eveimage.New(ad.Cache, httpClient, *isOfflineFlag)
	u.EveUniverseService = eu
	u.StatusCacheService = sc
	u.IsOffline = *isOfflineFlag
	u.IsUpdateTickerDisabled = *isUpdateTickerDisabledFlag
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
