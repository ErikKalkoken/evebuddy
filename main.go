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
	"github.com/ErikKalkoken/evebuddy/internal/eveimage"
	"github.com/ErikKalkoken/evebuddy/internal/httptransport"
	"github.com/ErikKalkoken/evebuddy/internal/sso"
	"github.com/ErikKalkoken/evebuddy/internal/uninstall"
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
	m := map[string]slog.Level{"DEBUG": slog.LevelDebug, "INFO": slog.LevelInfo, "WARN": slog.LevelWarn, "ERROR": slog.LevelError}
	v, ok := m[strings.ToUpper(value)]
	if !ok {
		return fmt.Errorf("unknown log level")
	}
	l.value = v
	return nil
}

// defined flags
var (
	levelFlag     logLevelFlag
	debugFlag     = flag.Bool("debug", false, "Show additional debug information")
	uninstallFlag = flag.Bool("uninstall", false, "Uninstalls the app by deleting all user files")
	pprofFlag     = flag.Bool("pprof", false, "Enable pprof web server")
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
	fyneApp := app.NewWithID(appID)
	ad, err := appdirs.New(fyneApp)
	if err != nil {
		log.Fatal(err)
	}
	flag.Parse()

	// setup crash reporting
	f, err := os.Create(filepath.Join(ad.Log, "crash.txt"))
	if err != nil {
		log.Fatal(err)
	}
	if err := debug.SetCrashOutput(f, debug.CrashOptions{}); err != nil {
		log.Fatal(err)
	}

	// setup logging
	slog.SetLogLoggerLevel(levelFlag.value)
	fn := fmt.Sprintf("%s/%s", ad.Log, logFileName)
	log.SetOutput(&lumberjack.Logger{
		Filename:   fn,
		MaxSize:    logMaxSizeMB, // megabytes
		MaxBackups: logMaxBackups,
	})

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

	// start uninstall app if requested
	if *uninstallFlag {
		u := uninstall.NewUI(fyneApp, ad)
		u.ShowAndRun()
		return
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
		Transport: httptransport.LoggedTransport{},
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
	u := ui.NewUI(fyneApp, ad, *debugFlag)
	u.CacheService = cache
	u.CharacterService = cs
	u.ESIStatusService = esistatus.New(esiClient)
	u.EveImageService = eveimage.New(ad.Cache, httpClient)
	u.EveUniverseService = eu
	u.StatusCacheService = sc
	u.Init()

	// start pprof web server
	if *pprofFlag {
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	// Start app
	u.ShowAndRun()
}
