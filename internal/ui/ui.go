// Package ui contains the code for rendering the UI.
package ui

import (
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"example/evebuddy/internal/model"
	"example/evebuddy/internal/service"
	"example/evebuddy/internal/storage"
)

// UI constants
const (
	myDateTime               = "2006.01.02 15:04"
	defaultIconSize          = 64
	mailUpdateTimeoutSeconds = 60
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
	mailArea         *mailDetailArea
	statusArea       *statusArea
	service          *service.Service
	toolbarBadge     *fyne.Container
	window           fyne.Window
}

// NewUI build the UI and returns it.
func NewUI(s *service.Service) *ui {
	a := app.New()
	w := a.NewWindow("Eve Buddy")
	u := &ui{app: a, window: w, service: s}

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
	mailTab := container.NewTabItemWithIcon("Mail", theme.MailComposeIcon(), mailContent)

	characterArea := u.NewCharacterArea()
	u.characterArea = characterArea
	characterContent := container.NewBorder(nil, nil, nil, nil, characterArea.content)
	characterTab := container.NewTabItemWithIcon("Character", theme.AccountIcon(), characterContent)

	statusArea := u.newStatusArea()
	u.statusArea = statusArea

	tabs := container.NewAppTabs(characterTab, mailTab)
	tabs.SetTabLocation(container.TabLocationLeading)

	toolbar := makeToolbar(u)
	mainContent := container.NewBorder(toolbar, statusArea.content, nil, nil, tabs)
	w.SetContent(mainContent)
	w.SetMaster()
	w.Resize(fyne.NewSize(800, 600))
	// w.SetMainMenu(MakeMenu(a, u))

	var c model.Character
	cID, err := s.DictionaryInt(model.SettingLastCharacterID)
	if err != nil {
		panic(err)
	}
	if cID != 0 {
		c, err = s.GetCharacter(int32(cID))
		if err != nil {
			if !errors.Is(err, storage.ErrNotFound) {
				slog.Error("Failed to load character", "error", err)
			}
		}
	}
	if c.ID != 0 {
		u.SetCurrentCharacter(&c)
	} else {
		u.ResetCurrentCharacter()
	}
	return u
}

func makeToolbar(u *ui) *fyne.Container {
	badge := container.NewHBox()
	u.toolbarBadge = badge
	toolbar := container.NewHBox(
		badge,
		layout.NewSpacer(),
		widget.NewButtonWithIcon("", theme.InfoIcon(), func() {
			u.ShowAboutDialog()
		}),
		widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
			u.ShowSettingsDialog()
		}),
		widget.NewButtonWithIcon("", theme.AccountIcon(), func() {
			u.ShowAccountDialog()
		}),
	)
	return container.NewVBox(toolbar, widget.NewSeparator())
}

// ShowAndRun shows the UI and runs it (blocking).
func (u *ui) ShowAndRun() {
	go func() {
		//TODO: Workaround to mitigate a bug that causes the window to sometimes render
		// only in parts and freeze. The issue is known to happen on Linux desktops.
		if runtime.GOOS == "linux" {
			time.Sleep(400 * time.Millisecond)
			u.window.Resize(fyne.NewSize(800, 601))
			u.window.Resize(fyne.NewSize(800, 600))
		}
		u.statusArea.StartUpdateTicker()
		u.characterArea.StartUpdateTicker()
		u.folderArea.StartUpdateTicker()
	}()
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
	u.updateToolbarBadge(c)
	err := u.service.DictionarySetInt(model.SettingLastCharacterID, int(c.ID))
	if err != nil {
		slog.Error("Failed to update last character setting", "characterID", c.ID)
	}
	u.characterArea.Redraw()
	u.folderArea.Refresh()
	updatedAt := u.service.MailUpdatedAt(c.ID)
	if updatedAt.IsZero() {
		go u.folderArea.UpdateMails(true)
	}
}

func (u *ui) updateToolbarBadge(c *model.Character) {
	if c == nil {
		u.toolbarBadge.RemoveAll()
		l := widget.NewLabel("No character")
		l.TextStyle = fyne.TextStyle{Italic: true}
		u.toolbarBadge.Add(l)
		return
	}
	uri, _ := c.PortraitURL(32)
	image := canvas.NewImageFromURI(uri)
	image.FillMode = canvas.ImageFillOriginal
	name := widget.NewLabel(fmt.Sprintf("%s (%s)", c.Name, c.Corporation.Name))
	name.TextStyle = fyne.TextStyle{Bold: true}
	u.toolbarBadge.RemoveAll()
	u.toolbarBadge.Add(container.NewPadded(image))
	u.toolbarBadge.Add(name)
}

func (u *ui) ResetCurrentCharacter() {
	u.currentCharacter = nil
	u.updateToolbarBadge(nil)
	err := u.service.DictionaryDelete(model.SettingLastCharacterID)
	if err != nil {
		slog.Error("Failed to delete last character setting")
	}
	u.characterArea.Redraw()
	u.folderArea.Refresh()
}
