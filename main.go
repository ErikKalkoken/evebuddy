package main

import (
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
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverse"
	"github.com/ErikKalkoken/evebuddy/internal/app/sqlite"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/cache"
	"github.com/ErikKalkoken/evebuddy/internal/dictionary"
	"github.com/ErikKalkoken/evebuddy/internal/eveimage"
	"github.com/ErikKalkoken/evebuddy/internal/httptransport"
	"github.com/antihax/goesi"
	"github.com/chasinglogic/appdirs"
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

// TODO: Reset flag defaults for production

// defined flags
var (
	levelFlag   logLevelFlag
	debugFlag   = flag.Bool("debug", false, "Show additional debug information")
	logFileFlag = flag.Bool("logfile", false, "Write logs to a file instead of the console")
	localFlag   = flag.Bool("local", false, "Store all files in the current directory instead of the user's home")
	removeFlag  = flag.Bool("remove-user-files", false, "Remove all user files of this app")
)

func init() {
	levelFlag.value = slog.LevelWarn
	flag.Var(&levelFlag, "loglevel", "set log level")
}

func main() {
	flag.Parse()
	slog.SetLogLoggerLevel(levelFlag.value)
	log.SetFlags(log.LstdFlags | log.Llongfile)
	ad := appdirs.New("evebuddy")
	if *removeFlag {
		fmt.Print("Are you sure you want to remove all local data of this app (y/N)?")
		var input string
		fmt.Scanln(&input)
		if strings.ToLower(input) == "y" {
			pp := []struct {
				name string
				path string
			}{
				{"user data", ad.UserData()},
				{"user log", ad.UserLog()},
				{"user cache", ad.UserCache()},
			}
			for _, p := range pp {
				if err := os.RemoveAll(p.path); err != nil {
					log.Fatal(err)
				}
				fmt.Printf("Deleted %s: %s\n", p.name, p.path)
			}
		} else {
			fmt.Println("Aborted")
		}
		return
	}
	fn, err := makeLogFileName(ad, *localFlag)
	if err != nil {
		log.Fatal(err)
	}
	if *logFileFlag {
		log.SetOutput(&lumberjack.Logger{
			Filename:   fn,
			MaxSize:    50, // megabytes
			MaxBackups: 3,
		})
	}
	dsn, err := makeDSN(ad, *localFlag)
	if err != nil {
		log.Fatal(err)
	}
	db, err := sqlite.InitDB(dsn)
	if err != nil {
		log.Fatalf("Failed to initialize database %s: %s", dsn, err)
	}
	defer db.Close()
	st := sqlite.New(db)
	imageCacheDir, err := makeImageCachePath(ad, *localFlag)
	if err != nil {
		log.Fatal(err)
	}
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
	dt := dictionary.New(st)
	cache := cache.New()
	sc := statuscache.New(cache)
	if err := sc.InitCache(st); err != nil {
		panic(err)
	}

	eu := eveuniverse.New(st, esiClient)
	eu.StatusCacheService = sc

	cs := character.New(st, httpClient, esiClient)
	cs.DictionaryService = dt
	cs.EveUniverseService = eu
	cs.StatusCacheService = sc

	u := ui.NewUI(*debugFlag)
	u.CacheService = cache
	u.CharacterService = cs
	u.DictionaryService = dt
	u.ESIStatusService = esistatus.New(esiClient)
	u.EveImageService = eveimage.New(imageCacheDir, httpClient)
	u.EveUniverseService = eu
	u.StatusCacheService = sc
	u.Init()
	u.ShowAndRun()
}

func makeLogFileName(ad *appdirs.App, isDebug bool) (string, error) {
	fn := "evebuddy.log"
	if isDebug {
		return fn, nil
	}
	if err := os.MkdirAll(ad.UserLog(), os.ModePerm); err != nil {
		return "", err
	}
	path := fmt.Sprintf("%s/%s", ad.UserLog(), fn)
	return path, nil
}

func makeDSN(ad *appdirs.App, isDebug bool) (string, error) {
	fn := "evebuddy.sqlite"
	if isDebug {
		return fmt.Sprintf("file:%s", fn), nil
	}
	path := ad.UserData()
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return "", err
	}
	dsn := fmt.Sprintf("file:%s/%s", path, fn)
	return dsn, nil
}

func makeImageCachePath(ad *appdirs.App, isDebug bool) (string, error) {
	var p string
	if isDebug {
		p = filepath.Join(".temp", "images")
	} else {
		p = filepath.Join(ad.UserCache(), "images")
	}
	if err := os.MkdirAll(p, os.ModePerm); err != nil {
		return "", err
	}
	return p, nil
}
