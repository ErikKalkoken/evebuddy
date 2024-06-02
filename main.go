package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/ErikKalkoken/evebuddy/internal/service"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/ErikKalkoken/evebuddy/internal/ui"
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
		f, err := os.OpenFile(fn, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("error opening file %s: %v", fn, err)
		}
		defer f.Close()
		log.SetOutput(f)
	}
	dsn, err := makeDSN(ad, *localFlag)
	if err != nil {
		log.Fatal(err)
	}
	db, err := storage.InitDB(dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database %s: %s", dsn, err)
	}
	defer db.Close()
	repository := storage.New(db)
	s := service.New(repository)
	cache, err := makeImageCachePath(ad, *localFlag)
	if err != nil {
		log.Fatal(err)
	}
	e := ui.NewUI(s, cache)
	e.ShowAndRun()
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
