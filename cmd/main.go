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
	db, err := repository.NewDB("storage.sqlite")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	queries := repository.New(db)
	s := service.NewService(queries)
	e := ui.NewUI(s)
	e.ShowAndRun()
}
