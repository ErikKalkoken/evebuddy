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

func (l *logLevelFlag) String() string {
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
	levelFlag   logLevelFlag
	loadMapFlag = flag.Bool("loadmap", false, "loads map")
)

func init() {
	levelFlag.value = slog.LevelWarn
	flag.Var(&levelFlag, "level", "log level name")
}

func main() {
	flag.Parse()
	slog.SetLogLoggerLevel(levelFlag.value)
	log.SetFlags(log.LstdFlags | log.Llongfile)

	db, err := storage.InitDB(makeDSN())
	if err != nil {
		panic(err)
	}
	defer db.Close()
	repository := storage.New(db)
	s := service.NewService(repository)

	if *loadMapFlag {
		err := s.LoadMap()
		if err != nil {
			slog.Error("Failed to load map", "err", err)
		}
		return
	}
	e := ui.NewUI(s)
	e.ShowAndRun()
}

func makeDSN() string {
	ad := appdirs.New("evebuddy")
	dataPath := ad.UserData()
	if err := os.MkdirAll(dataPath, os.ModePerm); err != nil {
		panic(err)
	}
	dsn := fmt.Sprintf("file:%s/evebuddy.sqlite", dataPath)
	return dsn
}
