package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"fyne.io/fyne/v2/app"
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
	"github.com/antihax/goesi"
)

const (
	ssoClientID = "11ae857fe4d149b2be60d875649c05f1"
	appID       = "io.github.erikkalkoken.evebuddy"
	userAgent   = "EveBuddy kalkoken87@gmail.com"
	dbFileName  = "evebuddy.sqlite"
	logFileName = "evebuddy.log"
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
	logFileFlag   = flag.Bool("logfile", true, "Write logs to a file instead of the console")
	uninstallFlag = flag.Bool("uninstall", false, "Uninstalls the app by deleting all user files")
	// showDirsFlag  = flag.Bool("show-dirs", false, "Show directories where user data is stored")
)

func init() {
	levelFlag.value = slog.LevelInfo
	flag.Var(&levelFlag, "loglevel", "set log level")
}

func main() {
	flag.Parse()
	fyneApp := app.NewWithID(appID)
	ad, err := appdirs.New(fyneApp)
	if err != nil {
		log.Fatal(err)
	}
	// if *showDirsFlag {
	// 	fmt.Printf("Database: %s\n", ad.data)
	// 	fmt.Printf("Cache: %s\n", ad.cache)
	// 	fmt.Printf("Logs: %s\n", ad.log)
	// 	fmt.Printf("Settings: %s\n", ad.settings)
	// 	return
	// }
	if *uninstallFlag {
		u := uninstall.NewUI(fyneApp, ad)
		u.ShowAndRun()
		return
	}
	f, err := os.Create(filepath.Join(ad.Log, "crash.txt"))
	if err != nil {
		log.Fatal(err)
	}
	if err := debug.SetCrashOutput(f, debug.CrashOptions{}); err != nil {
		log.Fatal(err)
	}
	slog.SetLogLoggerLevel(levelFlag.value)
	if *logFileFlag {
		fn := fmt.Sprintf("%s/%s", ad.Log, logFileName)
		log.SetOutput(&lumberjack.Logger{
			Filename:   fn,
			MaxSize:    50, // megabytes
			MaxBackups: 3,
		})
	}
	dsn := fmt.Sprintf("file:%s/%s", ad.Data, dbFileName)
	db, err := storage.InitDB(dsn)
	if err != nil {
		log.Fatalf("Failed to initialize database %s: %s", dsn, err)
	}
	defer db.Close()
	st := storage.New(db)
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
	sc := statuscache.New(cache)
	if err := sc.InitCache(context.TODO(), st); err != nil {
		panic(err)
	}

	eu := eveuniverse.New(st, esiClient)
	eu.StatusCacheService = sc

	en := evenotification.New()
	en.EveUniverseService = eu

	cs := character.New(st, httpClient, esiClient)
	cs.EveNotificationService = en
	cs.EveUniverseService = eu
	cs.StatusCacheService = sc
	cs.SSOService = sso.New(ssoClientID, httpClient, cache)

	u := ui.NewUI(fyneApp, *debugFlag)
	u.CacheService = cache
	u.CharacterService = cs
	u.ESIStatusService = esistatus.New(esiClient)
	u.EveImageService = eveimage.New(ad.Cache, httpClient)
	u.EveUniverseService = eu
	u.StatusCacheService = sc
	u.Init()
	u.ShowAndRun()
}
