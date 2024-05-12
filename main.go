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
	logFileFlag = flag.Bool("logfile", false, "Write a log file")
	debugFlag   = flag.Bool("debug", false, "Run in debug mode")
)

func init() {
	levelFlag.value = slog.LevelInfo
	flag.Var(&levelFlag, "loglevel", "set log level")
}

func main() {
	flag.Parse()
	slog.SetLogLoggerLevel(levelFlag.value)
	log.SetFlags(log.LstdFlags | log.Llongfile)
	ad := appdirs.New("evebuddy")
	if *logFileFlag {
		fn := makeLogFileName(ad, *debugFlag)
		f, err := os.OpenFile(fn, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("error opening file %s: %v", fn, err)
		}
		defer f.Close()
		log.SetOutput(f)
	}
	dsn := makeDSN(ad, *debugFlag)
	db, err := storage.InitDB(dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database %s: %s", dsn, err)
	}
	defer db.Close()
	repository := storage.New(db)
	s := service.NewService(repository)
	p := makeImageCachePath(ad, *debugFlag)
	e := ui.NewUI(s, p)
	e.ShowAndRun()
}

func makeLogFileName(ad *appdirs.App, isDebug bool) string {
	fn := "evebuddy.log"
	if isDebug {
		return fn
	}
	if err := os.MkdirAll(ad.UserLog(), os.ModePerm); err != nil {
		panic(err)
	}
	path := fmt.Sprintf("%s/%s", ad.UserLog(), fn)
	return path
}

func makeDSN(ad *appdirs.App, isDebug bool) string {
	fn := "evebuddy.sqlite"
	if isDebug {
		return fmt.Sprintf("file:%s", fn)
	}
	path := ad.UserData()
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		panic(err)
	}
	dsn := fmt.Sprintf("file:%s/%s", path, fn)
	return dsn
}

func makeImageCachePath(ad *appdirs.App, isDebug bool) string {
	var p string
	if isDebug {
		p = filepath.Join(".temp", "images")
	} else {
		p = filepath.Join(ad.UserCache(), "images")
	}
	if err := os.MkdirAll(p, os.ModePerm); err != nil {
		panic(err)
	}
	return p
}
