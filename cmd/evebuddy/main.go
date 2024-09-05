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
	"github.com/chasinglogic/appdirs"
)

const (
	ssoClientId = "11ae857fe4d149b2be60d875649c05f1"
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
	ad := appdirs.New("evebuddy")
	if *showDirsFlag {
		fmt.Printf("Database: %s\n", ad.UserData())
		fmt.Printf("Cache: %s\n", ad.UserCache())
		fmt.Printf("Logs: %s\n", ad.UserLog())
		return
	}
	if *uninstallFlag {
		fmt.Print("Are you sure you want to uninstall this app and delete all user files (y/N)?")
		var input string
		fmt.Scanln(&input)
		if strings.ToLower(input) == "y" {
			uninstall(ad.UserData(), ad.UserLog(), ad.UserCache())
		} else {
			fmt.Println("Aborted")
		}
		return
	}
	if *logFileFlag {
		fn, err := makeLogFileName(ad)
		if err != nil {
			log.Fatal(err)
		}
		log.SetOutput(&lumberjack.Logger{
			Filename:   fn,
			MaxSize:    50, // megabytes
			MaxBackups: 3,
		})
	}
	dsn, err := makeDSN(ad)
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
	cs.SSOService = sso.New(ssoClientId, httpClient, cache)

	imageCacheDir, err := initImageCachePath(ad)
	if err != nil {
		log.Fatal(err)
	}
	u := ui.NewUI(*debugFlag)
	u.CacheService = cache
	u.CharacterService = cs
	u.ESIStatusService = esistatus.New(esiClient)
	u.EveImageService = eveimage.New(imageCacheDir, httpClient)
	u.EveUniverseService = eu
	u.StatusCacheService = sc
	u.Init()
	u.ShowAndRun()
}

func makeLogFileName(ad *appdirs.App) (string, error) {
	fn := "evebuddy.log"
	if err := os.MkdirAll(ad.UserLog(), os.ModePerm); err != nil {
		return "", err
	}
	path := fmt.Sprintf("%s/%s", ad.UserLog(), fn)
	return path, nil
}

func makeDSN(ad *appdirs.App) (string, error) {
	fn := "evebuddy.sqlite"
	path := ad.UserData()
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return "", err
	}
	dsn := fmt.Sprintf("file:%s/%s", path, fn)
	return dsn, nil
}

func initImageCachePath(ad *appdirs.App) (string, error) {
	p := filepath.Join(ad.UserCache(), "images")
	if err := os.MkdirAll(p, os.ModePerm); err != nil {
		return "", err
	}
	return p, nil
}

func uninstall(data string, logs string, cache string) {
	pp := []struct {
		name string
		path string
	}{
		{"user data", data},
		{"user log", logs},
		{"user cache", cache},
	}
	for _, p := range pp {
		if err := os.RemoveAll(p.path); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Deleted %s: %s\n", p.name, p.path)
	}
}
