// Package UI contains the code for rendering the UI.
package ui

import (
	"example/esiapp/internal/storage"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

const (
	myDateTime      = "2006.01.02 15:04"
	allMailsLabelID = 0
)

type esiApp struct {
	main      fyne.Window
	statusBar *statusBar
}

func NewEsiApp(a fyne.App) fyne.Window {
	w := a.NewWindow("Eve Online App")
	e := &esiApp{main: w}

	var charID int32
	c, err := storage.FetchFirstCharacter()
	if err != nil {
		log.Printf("Failed to load any character: %v", err)
	} else {
		charID = c.ID
	}

	mail := e.newMail()
	headers := e.newHeaders(mail)
	folders := e.newFolders(headers)
	characters := e.newCharacters(folders)
	characters.update(charID)

	headersMail := container.NewHSplit(headers.container, mail.container)
	headersMail.SetOffset(0.35)

	main := container.NewHSplit(folders.container, headersMail)
	main.SetOffset(0.15)

	bar := e.newStatusBar()
	bar.update("PLACEHOLDER")
	e.statusBar = bar

	content := container.NewBorder(characters.container, bar.content, nil, nil, main)
	w.SetContent(content)
	w.Resize(fyne.NewSize(800, 600))

	return w
}
