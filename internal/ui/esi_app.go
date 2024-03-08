// Package UI contains the code for rendering the UI.
package ui

import (
	"example/esiapp/internal/storage"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

const (
	myDateTime = "2006.01.02 15:04"
)

type esiApp struct {
	main        fyne.Window
	characterID int32
}

func NewEsiApp(a fyne.App) fyne.Window {
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
	w.Resize(fyne.NewSize(800, 600))

	return w
}
