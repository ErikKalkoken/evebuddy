// Package ui contains the code for rendering the UI.
package ui

import (
	"database/sql"
	"example/evebuddy/internal/logic"
	"example/evebuddy/internal/model"
	"fmt"
	"log/slog"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// UI constants
const (
	myDateTime               = "2006.01.02 15:04"
	defaultIconSize          = 64
	mailUpdateTimeoutSeconds = 60
)

// Setting keys
const (
	settingLastCharacterID = "lastCharacterID"
)

// The ui is the root element of the UI, which contains all UI areas.
//
// Each UI area holds a pointer of the ui instance,
// which allow it to access the other UI areas and shared variables
type ui struct {
	app              fyne.App
	accountArea      *accountArea
	characterArea    *characterArea
	currentCharacter *model.Character
	folderArea       *folderArea
	headerArea       *headerArea
	mailArea         *mailArea
	statusArea       *statusArea
	window           fyne.Window
}

// NewUI build the UI and returns it.
func NewUI() *ui {
	a := app.New()
	w := a.NewWindow("Eve Buddy")
	u := &ui{app: a, window: w}

	mail := u.NewMailArea()
	u.mailArea = mail

	headers := u.NewHeaderArea()
	u.headerArea = headers

	folders := u.NewFolderArea()
	u.folderArea = folders

	headersMail := container.NewHSplit(headers.content, mail.content)
	headersMail.SetOffset(0.35)

	mailContent := container.NewHSplit(folders.content, headersMail)
	mailContent.SetOffset(0.15)
	mailTab := container.NewTabItemWithIcon("Mail", theme.MailComposeIcon(), addTitle(mailContent, "Mail"))

	characterArea := u.NewCharacterArea()
	u.characterArea = characterArea
	characterContent := container.NewBorder(nil, nil, nil, nil, characterArea.content)
	characterTab := container.NewTabItemWithIcon("Character", theme.AccountIcon(), addTitle(characterContent, "Character Sheet"))

	accountArea := u.NewAccountArea()
	u.accountArea = accountArea
	accountTab := container.NewTabItemWithIcon("Manage", theme.SettingsIcon(), addTitle(accountArea.content, "Manage Characters"))

	status := u.newStatusArea()
	u.statusArea = status

	tabs := container.NewAppTabs(characterTab, mailTab, accountTab)
	tabs.SetTabLocation(container.TabLocationLeading)

	c := container.NewBorder(nil, status.content, nil, nil, tabs)
	w.SetContent(c)
	w.Resize(fyne.NewSize(800, 600))
	w.SetMaster()

	characterID, err := model.GetSetting[int32](settingLastCharacterID)
	if err != nil {
		panic(err)
	}
	if characterID != 0 {
		c, err := model.FetchCharacter(characterID)
		if err != nil {
			if err != sql.ErrNoRows {
				slog.Error("Failed to load character", "error", err)
			}
		} else {
			u.SetCurrentCharacter(c)
		}
	}
	logic.StartEsiStatusTicker(status.status)
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
	u.window.SetTitle(fmt.Sprintf("Eve Buddy [%s]", c.Name))
	err := model.SetSetting(settingLastCharacterID, c.ID)
	if err != nil {
		slog.Error("Failed to update last character setting", "characterID", c.ID)
	}
	u.accountArea.Redraw()
	u.folderArea.Redraw()
	u.characterArea.Redraw()
}

func (u *ui) ResetCurrentCharacter() {
	u.currentCharacter = nil
	err := model.DeleteSetting(settingLastCharacterID)
	if err != nil {
		slog.Error("Failed to delete last character setting")
	}
	u.accountArea.Redraw()
	u.folderArea.Redraw()
	u.characterArea.Redraw()
}

func addTitle(c fyne.CanvasObject, title string) *fyne.Container {
	label := widget.NewLabel(strings.ToUpper(title))
	x := container.NewBorder(container.NewVBox(label, widget.NewSeparator()), nil, nil, nil, c)
	return x
}
