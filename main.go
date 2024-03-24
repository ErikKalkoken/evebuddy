package main

import (
	"flag"
	"log"
	"log/slog"

	"fyne.io/fyne/v2/app"

	"example/esiapp/internal/gui"
	"example/esiapp/internal/storage"
)

func main() {
	flag.Parse()
	slog.SetLogLoggerLevel(levelFlag.value)
	log.SetFlags(log.LstdFlags | log.Llongfile)
	if err := storage.Initialize(); err != nil {
		panic(err)
	}

	storage.Test()

	a := app.New()
	w := gui.NewEsiApp(a)
	w.ShowAndRun()
}
