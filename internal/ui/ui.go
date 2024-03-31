// Package ui contains the code for rendering the UI.
package ui

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

// The ui is the root of the UI tree that holds all UI areas together
type ui struct {
	window        fyne.Window
	characterArea *characterArea
	folderArea    *folderArea
	headerArea    *headerArea
	mailArea      *mailArea
	statusArea    *statusArea
	currentCharID int32
}

// NewUI returns a new ui instance.
func NewUI() *ui {
	a := app.New()
	w := a.NewWindow("Eve Online App")
	u := &ui{window: w}

	c, err := model.FetchFirstCharacter()
	if err != nil {
		if err != sql.ErrNoRows {
			slog.Error("Failed to load any character", "error", err)
		}
	} else {
		u.currentCharID = c.ID
	}

	mail := u.NewMailArea()
	u.mailArea = mail

	headers := u.NewHeaderArea()
	u.headerArea = headers

	folders := u.NewFolderArea()
	u.folderArea = folders

	characters := u.NewCharacterArea()
	u.characterArea = characters

	bar := u.newStatusArea()
	u.statusArea = bar

	characters.Redraw()

	headersMail := container.NewHSplit(headers.content, mail.content)
	headersMail.SetOffset(0.35)

	main := container.NewHSplit(folders.content, headersMail)
	main.SetOffset(0.15)

	content := container.NewBorder(characters.content, bar.content, nil, nil, main)
	w.SetContent(content)
	w.Resize(fyne.NewSize(800, 600))

	w.SetMainMenu(MakeMenu(a, u))
	w.SetMaster()
	return u
}

func (u *ui) ShowAndRun() {
	u.window.ShowAndRun()
}
