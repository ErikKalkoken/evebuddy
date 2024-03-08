package main

import (
	"example/esiapp/internal/storage"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

type esiApp struct {
	main        fyne.Window
	characterID int32
}

func newEsiApp(a fyne.App) fyne.Window {
	w := a.NewWindow("Eve Online App")
	e := &esiApp{main: w}

	c, err := storage.FetchFirstCharacter()
	if err != nil {
		log.Printf("Failed to load any character: %v", err)
	} else {
		e.characterID = c.ID
	}

	characters := e.newCharacters()
	mails := e.newMails()

	folders := e.newFolders()
	folders.update(e.characterID)

	main := container.NewHSplit(folders.container, mails.container)
	main.SetOffset(0.15)

	content := container.NewBorder(characters, nil, nil, nil, main)
	w.SetContent(content)

	return w
}
