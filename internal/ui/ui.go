// Package ui contains the code for rendering the UI.
package ui

import (
	"errors"
	"log/slog"
	"runtime"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"

	"github.com/ErikKalkoken/evebuddy/internal/eveonline/images"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/service"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

// UI constants
const (
	myDateTime               = "2006.01.02 15:04"
	defaultIconSize          = 32
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
	toolbarArea           *toolbarArea
	tabs                  *container.AppTabs
	window                fyne.Window
	imageManager          *images.Manager
}

// NewUI build the UI and returns it.
func NewUI(service *service.Service, imageCachePath string) *ui {
	a := app.New()
	w := a.NewWindow(appName(a))
	u := &ui{app: a, window: w, service: service, imageManager: images.New(imageCachePath)}

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
	u.toolbarArea = u.newToolbarArea()

	u.tabs = container.NewAppTabs(characterTab, mailTab, skillqueueTab, walletTab, overviewTab)
	u.tabs.SetTabLocation(container.TabLocationLeading)

	mainContent := container.NewBorder(u.toolbarArea.content, u.statusArea.content, nil, nil, u.tabs)
	w.SetContent(mainContent)
	w.SetMaster()

	var c *model.MyCharacter
	cID, ok, err := service.DictionaryInt(model.SettingLastCharacterID)
	if err != nil {
		panic(err)
	}
	if ok {
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
	keyW := "window-width"
	width, ok, err := u.service.DictionaryFloat32(keyW)
	if err != nil || !ok {
		width = 1000
	}
	keyH := "window-height"
	height, ok, err := u.service.DictionaryFloat32(keyH)
	if err != nil || !ok {
		width = 600
	}
	w.Resize(fyne.NewSize(width, height))

	keyTabID := "tab-ID"
	index, ok, err := u.service.DictionaryInt(keyTabID)
	if err == nil && ok {
		u.tabs.SelectIndex(index)
	}
	w.SetOnClosed(func() {
		s := w.Canvas().Size()
		u.service.DictionarySetFloat32(keyW, s.Width)
		u.service.DictionarySetFloat32(keyH, s.Height)
		index := u.tabs.SelectedIndex()
		u.service.DictionarySetInt(keyTabID, index)
	})
	return u
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
	u.RefreshOverview()
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
	err := u.service.DictionarySetInt(model.SettingLastCharacterID, int(c.ID))
	if err != nil {
		slog.Error("Failed to update last character setting", "characterID", c.ID)
	}
	u.RefreshCurrentCharacter()
}

func (u *ui) RefreshCurrentCharacter() {
	u.toolbarArea.Refresh()
	u.characterArea.Redraw()
	u.folderArea.Refresh()
	u.skillqueueArea.Refresh()
	u.walletTransactionArea.Refresh()
	u.window.Content().Refresh()
}

func (u *ui) RefreshOverview() {
	u.overviewArea.Refresh()
}

func (u *ui) ResetCurrentCharacter() {
	u.currentCharacter = nil
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
				lastUpdated, ok, err := u.service.DictionaryTime(key)
				if err != nil || !ok {
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
