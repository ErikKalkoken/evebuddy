package main

import (
	"flag"
	"log"
	"log/slog"

	"example/esiapp/internal/cache"
	"example/esiapp/internal/gui"
	"example/esiapp/internal/model"
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
	cacheDb, err := cache.InitDB("cache.sqlite")
	if err != nil {
		panic(err)
	}
	defer cacheDb.Close()
	// storage.Test()

	e := gui.NewEveApp()
	e.ShowAndRun()
}
