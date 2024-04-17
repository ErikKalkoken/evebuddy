package main

import (
	"flag"
	"log"
	"log/slog"

	"example/evebuddy/internal/service"
	"example/evebuddy/internal/storage"
	"example/evebuddy/internal/ui"
)

func main() {
	flag.Parse()
	slog.SetLogLoggerLevel(levelFlag.value)
	log.SetFlags(log.LstdFlags | log.Llongfile)
	slog.Info("current flags", "createDB", *createDBFlag)
	db, err := storage.ConnectDB("storage.sqlite", *createDBFlag)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	repository := storage.New(db)
	s := service.NewService(repository)
	e := ui.NewUI(s)
	e.ShowAndRun()
}
