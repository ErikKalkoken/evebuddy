package main

import (
	"log"

	"fyne.io/fyne/v2/app"

	"example/esiapp/internal/storage"
	"example/esiapp/internal/ui"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Llongfile)

	if err := storage.Initialize(); err != nil {
		log.Fatal(err)
	}

	// storage.Test()

	a := app.New()
	w := ui.NewEsiApp(a)
	w.ShowAndRun()
}
