package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/ErikKalkoken/evebuddy/internal/service"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/ErikKalkoken/evebuddy/internal/ui"
	"github.com/chasinglogic/appdirs"
)

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
