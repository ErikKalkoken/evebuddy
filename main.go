package main

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"

	"example/esiapp/internal/storage"
)

const (
	myDateTime = "2006.01.02 15:04"
)

type esiapp struct {
	main        fyne.Window
	characterID int32
}

func main() {
	log.SetFlags(log.LstdFlags | log.Llongfile)

	if err := storage.Initialize(); err != nil {
		log.Fatal(err)
	}

	// storage.Test()

	a := app.New()
	w := a.NewWindow("Eve Online App")
	ui := &esiapp{main: w}

	c, err := storage.FetchFirstCharacter()
	if err != nil {
		log.Printf("Failed to load any character: %v", err)
	} else {
		ui.characterID = c.ID
	}

	characters := ui.newCharacters(w)
	mails := ui.newMails()

	folders := ui.newFolders()
	folders.update(ui.characterID)

	main := container.NewHSplit(folders.container, mails.container)
	main.SetOffset(0.15)

	content := container.NewBorder(characters, nil, nil, nil, main)
	w.SetContent(content)
	w.Resize(fyne.NewSize(800, 600))
	w.ShowAndRun()
}
