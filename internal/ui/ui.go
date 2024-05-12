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

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/service"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/ErikKalkoken/evebuddy/internal/widgets"
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
	app                   fyne.App
	characterArea         *characterArea
	currentCharacter      *model.MyCharacter
	folderArea            *folderArea
	headerArea            *headerArea
	mailArea              *mailDetailArea
	overviewArea          *overviewArea
	statusArea            *statusArea
	service               *service.Service
	skillqueueArea        *skillqueueArea
	walletTransactionArea *walletTransactionArea
	toolbarBadge          *fyne.Container
	window                fyne.Window
}

// NewUI build the UI and returns it.
func NewUI(service *service.Service) *ui {
	a := app.New()
	w := a.NewWindow(appName(a))
	u := &ui{app: a, window: w, service: service}

	u.mailArea = u.NewMailArea()
	u.headerArea = u.NewHeaderArea()
	u.folderArea = u.NewFolderArea()
	split1 := container.NewHSplit(u.headerArea.content, u.mailArea.content)
	split1.SetOffset(0.35)
	split2 := container.NewHSplit(u.folderArea.content, split1)
	split2.SetOffset(0.15)
	mailTab := container.NewTabItemWithIcon("Mail", theme.MailComposeIcon(), split2)

	u.characterArea = u.NewCharacterArea()
	characterContent := container.NewBorder(nil, nil, nil, nil, u.characterArea.content)
	characterTab := container.NewTabItemWithIcon("Character Sheet",
		theme.NewThemedResource(resourcePortraitSvg), characterContent)

	u.overviewArea = u.NewOverviewArea()
	overviewTab := container.NewTabItemWithIcon("Characters",
		theme.NewThemedResource(resourceGroupSvg), u.overviewArea.content)

	u.skillqueueArea = u.NewSkillqueueArea()
	skillqueueTab := container.NewTabItemWithIcon("Skill Queue",
		theme.NewThemedResource(resourceChecklistrtlSvg), u.skillqueueArea.content)

	u.walletTransactionArea = u.NewWalletTransactionArea()
	walletTab := container.NewTabItemWithIcon("Wallet",
		theme.NewThemedResource(resourceAttachmoneySvg), u.walletTransactionArea.content)

	u.statusArea = u.newStatusArea()

	tabs := container.NewAppTabs(characterTab, mailTab, skillqueueTab, walletTab, overviewTab)
	tabs.SetTabLocation(container.TabLocationLeading)

	mainContent := container.NewBorder(makeToolbar(u), u.statusArea.content, nil, nil, tabs)
	w.SetContent(mainContent)
	w.SetMaster()
	w.Resize(fyne.NewSize(1000, 600))
	// w.SetFullScreen(true)

	var c *model.MyCharacter
	cID, err := service.DictionaryInt(model.SettingLastCharacterID)
	if err != nil {
		panic(err)
	}
	if cID != 0 {
		c, err = service.GetMyCharacter(int32(cID))
		if err != nil {
			if !errors.Is(err, storage.ErrNotFound) {
				slog.Error("Failed to load character", "error", err)
			}
		}
	}
	if c != nil {
		u.SetCurrentCharacter(c)
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
		widget.NewButtonWithIcon("", theme.NewThemedResource(resourceManageaccountsSvg), func() {
			u.ShowAccountDialog()
		}),
	)
	return container.NewVBox(toolbar, widget.NewSeparator())
}

// ShowAndRun shows the UI and runs it (blocking).
func (u *ui) ShowAndRun() {
	go func() {
		// Workaround to mitigate a bug that causes the window to sometimes render
		// only in parts and freeze. The issue is known to happen on Linux desktops.
		if runtime.GOOS == "linux" {
			time.Sleep(400 * time.Millisecond)
			s := u.window.Canvas().Size()
			u.window.Resize(fyne.NewSize(s.Width, s.Height+0.1))
			u.window.Resize(fyne.NewSize(s.Width, s.Height))
		}
		u.statusArea.StartUpdateTicker()
		u.characterArea.StartUpdateTicker()
		u.folderArea.StartUpdateTicker()
		u.skillqueueArea.StartUpdateTicker()
		u.walletTransactionArea.StartUpdateTicker()
		u.StartUpdateTickerEveCharacters()
	}()
	u.window.ShowAndRun()
}

func (u *ui) CurrentCharID() int32 {
	if u.currentCharacter == nil {
		return 0
	}
	return u.currentCharacter.ID
}

func (u *ui) CurrentChar() *model.MyCharacter {
	return u.currentCharacter
}

func (u *ui) SetCurrentCharacter(c *model.MyCharacter) {
	u.currentCharacter = c
	u.updateToolbarBadge(c)
	err := u.service.DictionarySetInt(model.SettingLastCharacterID, int(c.ID))
	if err != nil {
		slog.Error("Failed to update last character setting", "characterID", c.ID)
	}
	u.RefreshCurrentCharacter()
}

func (u *ui) RefreshCurrentCharacter() {
	u.characterArea.Redraw()
	u.folderArea.Refresh()
	u.skillqueueArea.Refresh()
	u.walletTransactionArea.Refresh()
	u.window.Content().Refresh()
}

func (u *ui) updateToolbarBadge(c *model.MyCharacter) {
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
	s := fmt.Sprintf("%s (%s)", c.Character.Name, c.Character.Corporation.Name)
	name := widget.NewLabel(s)
	name.TextStyle = fyne.TextStyle{Bold: true}
	u.toolbarBadge.RemoveAll()
	u.toolbarBadge.Add(container.NewPadded(image))
	u.toolbarBadge.Add(name)
	cc, err := u.service.ListMyCharactersShort()
	if err != nil {
		panic(err)
	}
	menuItems := make([]*fyne.MenuItem, 0)
	for _, myC := range cc {
		if myC.ID == c.ID {
			continue
		}
		item := fyne.NewMenuItem(myC.Name, func() {
			newChar, err := u.service.GetMyCharacter(myC.ID)
			if err != nil {
				panic(err)
			}
			u.SetCurrentCharacter(newChar)
		})
		menuItems = append(menuItems, item)
	}
	menu := fyne.NewMenu("", menuItems...)
	b := widgets.NewContextMenuButtonWithIcon(
		theme.NewThemedResource(resourceSwitchaccountSvg), "", menu)
	u.toolbarBadge.Add(b)
}

func (u *ui) ResetCurrentCharacter() {
	u.currentCharacter = nil
	u.updateToolbarBadge(nil)
	err := u.service.DictionaryDelete(model.SettingLastCharacterID)
	if err != nil {
		slog.Error("Failed to delete last character setting")
	}
	u.RefreshCurrentCharacter()
}

func (u *ui) StartUpdateTickerEveCharacters() {
	ticker := time.NewTicker(30 * time.Second)
	key := "eve-characters-last-updated"
	go func() {
		for {
			func() {
				lastUpdated, err := u.service.DictionaryTime(key)
				if err != nil {
					return
				}
				if time.Now().Before(lastUpdated.Add(3600 * time.Second)) {
					return
				}
				u.service.UpdateAllEveCharactersESI()
			}()
			<-ticker.C
		}
	}()
}

func appName(a fyne.App) string {
	info := a.Metadata()
	name := info.Name
	if name == "" {
		return "EVE Buddy"
	}
	return name
}
