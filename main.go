package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
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
	logFileFlag = flag.Bool("logfile", true, "Wether to write a log file")
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
		fn := makeLogFileName(ad)
		f, err := os.OpenFile(fn, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("error opening file %s: %v", fn, err)
		}
		defer f.Close()
		log.SetOutput(f)
	}
	dsn := makeDSN(ad)
	db, err := storage.InitDB(dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database %s: %s", dsn, err)
	}
	defer db.Close()
	repository := storage.New(db)
	s := service.NewService(repository)
	e := ui.NewUI(s)
	e.ShowAndRun()
}

func makeLogFileName(ad *appdirs.App) string {
	path := ad.UserLog()
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		panic(err)
	}
	fn := fmt.Sprintf("%s/evebuddy.log", ad.UserLog())
	return fn
}

func makeDSN(ad *appdirs.App) string {
	path := ad.UserData()
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		panic(err)
	}
	dsn := fmt.Sprintf("file:%s/evebuddy.sqlite", path)
	return dsn
}
