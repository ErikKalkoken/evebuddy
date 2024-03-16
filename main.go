package main

import (
	"log"

	"fyne.io/fyne/v2/app"

	"example/esiapp/internal/gui"
	"example/esiapp/internal/storage"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Llongfile)

	if err := storage.Initialize(); err != nil {
		log.Fatal(err)
	}

	// storage.Test()

	a := app.New()
	w := gui.NewEsiApp(a)
	w.ShowAndRun()
}
