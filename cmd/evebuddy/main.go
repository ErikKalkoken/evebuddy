package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
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
	"github.com/ErikKalkoken/evebuddy/internal/cache"
	"github.com/ErikKalkoken/evebuddy/internal/eveimage"
	"github.com/ErikKalkoken/evebuddy/internal/httptransport"
	"github.com/ErikKalkoken/evebuddy/internal/sso"
	"github.com/antihax/goesi"
)

const (
	ssoClientID = "11ae857fe4d149b2be60d875649c05f1"
	appID       = "io.github.erikkalkoken.evebuddy"
)

// defined flags
var (
	levelFlag     logLevelFlag
	debugFlag     = flag.Bool("debug", false, "Show additional debug information")
	logFileFlag   = flag.Bool("logfile", true, "Write logs to a file instead of the console")
	uninstallFlag = flag.Bool("uninstall", false, "Uninstalls the app by deleting all user files")
	showDirsFlag  = flag.Bool("show-dirs", false, "Show directories where user data is stored")
)

func init() {
	levelFlag.value = slog.LevelInfo
	flag.Var(&levelFlag, "loglevel", "set log level")
}

func main() {
	flag.Parse()
	slog.SetLogLoggerLevel(levelFlag.value)
	fyneApp := app.NewWithID(appID)
	ad := newAppDirs(fyneApp)
	if *showDirsFlag {
		fmt.Printf("Database: %s\n", ad.data)
		fmt.Printf("Cache: %s\n", ad.cache)
		fmt.Printf("Logs: %s\n", ad.log)
		fmt.Printf("Settings: %s\n", ad.settings)
		return
	}
	if *uninstallFlag {
		fmt.Print("Are you sure you want to uninstall this app and delete all user files (y/N)?")
		var input string
		fmt.Scanln(&input)
		if strings.ToLower(input) == "y" {
			if err := ad.deleteAll(); err != nil {
				log.Fatal(err)
			}
			fmt.Printf("App uninstalled")
		} else {
			fmt.Println("Aborted")
		}
		return
	}
	if *logFileFlag {
		fn, err := ad.initLogFile()
		if err != nil {
			log.Fatal(err)
		}
		log.SetOutput(&lumberjack.Logger{
			Filename:   fn,
			MaxSize:    50, // megabytes
			MaxBackups: 3,
		})
	}
	dsn, err := ad.initDSN()
	if err != nil {
		log.Fatal(err)
	}
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
	userAgent := "EveBuddy kalkoken87@gmail.com"
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

	imageCacheDir, err := ad.initImageCachePath()
	if err != nil {
		log.Fatal(err)
	}

	u := ui.NewUI(fyneApp, *debugFlag)
	u.CacheService = cache
	u.CharacterService = cs
	u.ESIStatusService = esistatus.New(esiClient)
	u.EveImageService = eveimage.New(imageCacheDir, httpClient)
	u.EveUniverseService = eu
	u.StatusCacheService = sc
	u.Init()
	u.ShowAndRun()
}
