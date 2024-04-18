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
	db, err := storage.InitDB("evebuddy.sqlite")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	repository := storage.New(db)
	s := service.NewService(repository)
	e := ui.NewUI(s)
	e.ShowAndRun()
}
