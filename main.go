package main

import (
	"flag"
	"log"
	"log/slog"

	"fyne.io/fyne/v2/app"

	"example/esiapp/internal/gui"
	"example/esiapp/internal/models"
)

func main() {
	flag.Parse()
	slog.SetLogLoggerLevel(levelFlag.value)
	log.SetFlags(log.LstdFlags | log.Llongfile)
	if err := models.Initialize("storage2.sqlite"); err != nil {
		panic(err)
	}

	// storage.Test()

	a := app.New()
	w := gui.NewEsiApp(a)
	w.ShowAndRun()
}
