// Package UI contains the code for rendering the UI.
package gui

import (
	"database/sql"
	"example/esiapp/internal/model"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
)

const (
	myDateTime = "2006.01.02 15:04"
)

// Main type for the core app structure
type eveApp struct {
	winMain    fyne.Window
	statusBar  *statusBar
	characters *characters
}

func (e *eveApp) ShowAndRun() {
	e.winMain.ShowAndRun()
}

func NewEveApp() *eveApp {
	a := app.New()
	w := a.NewWindow("Eve Online App")
	e := &eveApp{winMain: w}

	var charID int32
	c, err := model.FetchFirstCharacter()
	if err != nil {
		if err != sql.ErrNoRows {
			slog.Error("Failed to load any character", "error", err)
		}
	} else {
		charID = c.ID
	}

	bar := e.newStatusBar()
	e.statusBar = bar

	mail := e.newMail()
	headers := e.newHeaders(mail)
	folders := e.newFolders(headers)
	characters := e.newCharacters(folders)
	characters.update(charID)
	e.characters = characters

	headersMail := container.NewHSplit(headers.content, mail.content)
	headersMail.SetOffset(0.35)

	main := container.NewHSplit(folders.content, headersMail)
	main.SetOffset(0.15)

	content := container.NewBorder(characters.content, bar.content, nil, nil, main)
	w.SetContent(content)
	w.Resize(fyne.NewSize(800, 600))

	folders.updateMails()

	w.SetMainMenu(MakeMenu(a, e))
	w.SetMaster()
	return e
}
