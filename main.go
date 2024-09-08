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
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	xappdirs "github.com/chasinglogic/appdirs"
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
	ssoClientID     = "11ae857fe4d149b2be60d875649c05f1"
	appID           = "io.github.erikkalkoken.evebuddy"
	appName         = "evebuddy"
	logFileName     = "evebuddy.log"
	dbFileName      = "evebuddy.sqlite"
	cacheFolderName = "images"
	userAgent       = "EveBuddy kalkoken87@gmail.com"
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
	showDirsFlag  = flag.Bool("show-dirs", false, "Show directories where user data is stored")
)

// appDirs represents the app's local directories for storing logs etc.
type appDirs struct {
	cache    string
	data     string
	log      string
	settings string
}

func newAppDirs(fyneApp fyne.App) appDirs {
	ad := xappdirs.New(appName)
	x := appDirs{
		data:     ad.UserData(),
		cache:    ad.UserCache(),
		log:      ad.UserLog(),
		settings: fyneApp.Storage().RootURI().Path(),
	}
	return x
}

func (ad appDirs) deleteAll() error {
	for _, p := range []string{ad.log, ad.cache, ad.data, ad.settings} {
		if err := os.RemoveAll(p); err != nil {
			return err
		}
		fmt.Printf("Deleted %s\n", p)
	}
	return nil
}

func (ad appDirs) initLogFile() (string, error) {
	if err := os.MkdirAll(ad.log, os.ModePerm); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s", ad.log, logFileName), nil
}

func (ad appDirs) initDSN() (string, error) {
	if err := os.MkdirAll(ad.data, os.ModePerm); err != nil {
		return "", err
	}
	dsn := fmt.Sprintf("file:%s/%s", ad.data, dbFileName)
	return dsn, nil
}

func (ad appDirs) initImageCachePath() (string, error) {
	p := filepath.Join(ad.cache, cacheFolderName)
	if err := os.MkdirAll(p, os.ModePerm); err != nil {
		return "", err
	}
	return p, nil
}

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
