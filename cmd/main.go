package main

import (
	"flag"
	"log"
	"log/slog"

	"example/evebuddy/internal/model"
	"example/evebuddy/internal/service"
	"example/evebuddy/internal/ui"
)

func main() {
	flag.Parse()
	slog.SetLogLoggerLevel(levelFlag.value)
	log.SetFlags(log.LstdFlags | log.Llongfile)
	db, err := model.InitDB("storage.sqlite")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	// storage.Test()

	s := service.NewService()
	e := ui.NewUI(s)
	e.ShowAndRun()
}
