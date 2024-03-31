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

// The UI is the root type that holds all UI areas together
type ui struct {
	window        fyne.Window
	statusArea    *statusArea
	characterArea *characterArea
}

func NewUI() *ui {
	a := app.New()
	w := a.NewWindow("Eve Online App")
	u := &ui{window: w}

	var charID int32
	c, err := model.FetchFirstCharacter()
	if err != nil {
		if err != sql.ErrNoRows {
			slog.Error("Failed to load any character", "error", err)
		}
	} else {
		charID = c.ID
	}

	bar := u.newStatusArea()
	u.statusArea = bar

	mail := u.newMailArea()
	headers := u.newHeaderArea(mail)
	folders := u.newFolderArea(headers)
	characters := u.newCharacterArea(folders)
	characters.update(charID)
	u.characterArea = characters

	headersMail := container.NewHSplit(headers.content, mail.content)
	headersMail.SetOffset(0.35)

	main := container.NewHSplit(folders.content, headersMail)
	main.SetOffset(0.15)

	content := container.NewBorder(characters.content, bar.content, nil, nil, main)
	w.SetContent(content)
	w.Resize(fyne.NewSize(800, 600))

	folders.updateMails()

	w.SetMainMenu(MakeMenu(a, u))
	w.SetMaster()
	return u
}

func (u *ui) ShowAndRun() {
	u.window.ShowAndRun()
}
