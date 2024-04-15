package main

import (
	"flag"
	"log"
	"log/slog"

	"example/evebuddy/internal/repository"
	"example/evebuddy/internal/service"
	"example/evebuddy/internal/ui"
)

func main() {
	flag.Parse()
	slog.SetLogLoggerLevel(levelFlag.value)
	log.SetFlags(log.LstdFlags | log.Llongfile)
	slog.Info("current flags", "createDB", *createDBFlag)
	db, err := repository.ConnectDB("storage.sqlite", *createDBFlag)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	queries := repository.New(db)
	s := service.NewService(queries)
	e := ui.NewUI(s)
	e.ShowAndRun()
}
