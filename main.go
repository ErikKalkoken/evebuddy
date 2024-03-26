package main

import (
	"flag"
	"log"
	"log/slog"

	"fyne.io/fyne/v2/app"

	"example/esiapp/internal/gui"
	"example/esiapp/internal/model"
)

func main() {
	flag.Parse()
	slog.SetLogLoggerLevel(levelFlag.value)
	log.SetFlags(log.LstdFlags | log.Llongfile)
	db, err := model.Initialize("storage.sqlite", true)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	// storage.Test()

	a := app.New()
	e := gui.NewEsiApp(a)
	e.Main.ShowAndRun()
}
