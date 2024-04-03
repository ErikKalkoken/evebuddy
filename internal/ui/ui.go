// Package ui contains the code for rendering the UI.
package ui

import (
	"database/sql"
	"example/esiapp/internal/model"
	"log/slog"
	"net/http"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
)

const (
	myDateTime = "2006.01.02 15:04"
)

// The ui is the root element of the UI, which contains all UI areas.
//
// Each UI area holds a pointer of the ui instance,
// which allow it to access the other UI areas and shared variables
type ui struct {
	app              fyne.App
	characterArea    *characterArea
	currentCharacter *model.Character
	folderArea       *folderArea
	headerArea       *headerArea
	mailArea         *mailArea
	statusArea       *statusArea
	window           fyne.Window
}

var httpClient = &http.Client{
	Timeout: time.Second * 30, // Timeout after 30 seconds
}

// NewUI build the UI and returns it.
func NewUI() *ui {
	a := app.New()
	w := a.NewWindow("Eve Online App")
	u := &ui{app: a, window: w}

	c, err := model.FetchFirstCharacter()
	if err != nil {
		if err != sql.ErrNoRows {
			slog.Error("Failed to load any character", "error", err)
		}
	} else {
		u.currentCharacter = c
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

// ShowAndRun shows the UI and runs it (blocking).
func (u *ui) ShowAndRun() {
	u.window.ShowAndRun()
}

func (u *ui) CurrentCharID() int32 {
	if u.currentCharacter == nil {
		return 0
	}
	return u.currentCharacter.ID
}

func (u *ui) CurrentChar() *model.Character {
	return u.currentCharacter
}

func (u *ui) SetCurrentCharacter(c *model.Character) {
	u.currentCharacter = c
}

func (u *ui) ResetCurrentCharacter() {
	u.currentCharacter = nil
}
