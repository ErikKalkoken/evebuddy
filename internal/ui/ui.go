// Package ui contains the code for rendering the UI.
package ui

import (
	"errors"
	"example/evebuddy/internal/repository"
	"example/evebuddy/internal/service"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
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
	characterArea    *characterArea
	currentCharacter *repository.Character
	folderArea       *folderArea
	headerArea       *headerArea
	mailArea         *mailArea
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
	mailTab := container.NewTabItemWithIcon("Mail", theme.MailComposeIcon(), addTitle(mailContent, "Mail"))

	characterArea := u.NewCharacterArea()
	u.characterArea = characterArea
	characterContent := container.NewBorder(nil, nil, nil, nil, characterArea.content)
	characterTab := container.NewTabItemWithIcon("Character", theme.AccountIcon(), addTitle(characterContent, "Character Sheet"))

	status := u.newStatusArea()
	u.statusArea = status

	tabs := container.NewAppTabs(characterTab, mailTab)
	tabs.SetTabLocation(container.TabLocationLeading)

	toolbar := makeToolbar(u)
	mainContent := container.NewBorder(toolbar, status.content, nil, nil, tabs)
	w.SetContent(mainContent)
	w.SetMaster()
	w.Resize(fyne.NewSize(800, 600))
	// w.SetMainMenu(MakeMenu(a, u))

	characterID, err := s.GetSettingInt32(settingLastCharacterID)
	if err != nil {
		panic(err)
	}
	if characterID != 0 {
		c, err := s.GetCharacter(characterID)
		if err != nil {
			if !errors.Is(err, repository.ErrNotFound) {
				slog.Error("Failed to load character", "error", err)
			}
		} else {
			u.SetCurrentCharacter(&c)
		}
	}
	go func() {
		//TODO: Find better workaround
		time.Sleep(250 * time.Millisecond)
		s.StartEsiStatusTicker(status.status)
		w.Resize(fyne.NewSize(800, 601))
		w.Resize(fyne.NewSize(800, 600))
	}()
	return u
}

func makeToolbar(u *ui) *fyne.Container {
	badge := container.NewHBox()
	u.toolbarBadge = badge
	toolbar := container.NewHBox(
		badge,
		layout.NewSpacer(),
		widget.NewButtonWithIcon("", theme.InfoIcon(), func() {
			text := "Eve Buddy v0.1.0\n\n(c) 2024 Erik Kalkoken"
			d := dialog.NewInformation("About", text, u.window)
			d.Show()
		}),
		widget.NewButtonWithIcon("", theme.AccountIcon(), func() {
			u.ShowManageDialog()
		}),
	)
	return container.NewVBox(toolbar, widget.NewSeparator())
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

func (u *ui) CurrentChar() *repository.Character {
	return u.currentCharacter
}

func (u *ui) SetCurrentCharacter(c *repository.Character) {
	u.currentCharacter = c
	u.updateToolbarBadge(c)
	err := u.service.SetSettingInt32(settingLastCharacterID, c.ID)
	if err != nil {
		slog.Error("Failed to update last character setting", "characterID", c.ID)
	}
	u.characterArea.Redraw()
	u.folderArea.Redraw()
}

func (u *ui) updateToolbarBadge(c *repository.Character) {
	if c == nil {
		u.toolbarBadge.RemoveAll()
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
	err := u.service.DeleteSetting(settingLastCharacterID)
	if err != nil {
		slog.Error("Failed to delete last character setting")
	}
	u.characterArea.Redraw()
	u.folderArea.Redraw()
}

func addTitle(c fyne.CanvasObject, title string) *fyne.Container {
	label := widget.NewLabel(strings.ToUpper(title))
	x := container.NewBorder(container.NewVBox(label, widget.NewSeparator()), nil, nil, nil, c)
	return x
}
