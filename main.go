package main

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"example/esiapp/internal/storage"
)

const (
	myDateTime = "2006.01.02 15:04"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Llongfile)

	if err := storage.Initialize(); err != nil {
		log.Fatal(err)
	}

	// storage.Test()

	a := app.New()
	w := newEsiApp(a)
	w.Resize(fyne.NewSize(800, 600))
	w.ShowAndRun()
}
