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
	myDateTime                = "2006.01.02 15:04"
	defaultIconSize           = 32
	myFloatFormat             = "#,###.##"
	eveCharacterUpdateTicker  = 60 * time.Second
	eveCharacterUpdateTimeout = 3600 * time.Second
)

// The ui is the root object of the UI and contains all UI areas.
//
// Each UI area holds a pointer of the ui instance, so that areas can
// call methods on other UI areas and access shared variables in the UI.
type ui struct {
	app                   fyne.App
	currentCharacter      *model.MyCharacter
	folderArea            *folderArea
	headerArea            *headerArea
	imageManager          *images.Manager
	mailArea              *mailDetailArea
	mailTab               *container.TabItem
	overviewArea          *overviewArea
	statusArea            *statusArea
	service               *service.Service
	skillqueueArea        *skillqueueArea
	skillqueueTab         *container.TabItem
	toolbarArea           *toolbarArea
	tabs                  *container.AppTabs
	walletJournalArea     *walletJournalArea
	walletTransactionArea *walletTransactionArea
	window                fyne.Window
}

// NewUI build the UI and returns it.
func NewUI(service *service.Service, imageCachePath string) *ui {
	app := app.New()
	w := app.NewWindow(appName(app))
	u := &ui{app: app, window: w, service: service, imageManager: images.New(imageCachePath)}

	u.mailArea = u.NewMailArea()
	u.headerArea = u.NewHeaderArea()
	u.folderArea = u.NewFolderArea()
	split1 := container.NewHSplit(u.headerArea.content, u.mailArea.content)
	split1.SetOffset(0.35)
	split2 := container.NewHSplit(u.folderArea.content, split1)
	split2.SetOffset(0.15)
	u.mailTab = container.NewTabItemWithIcon("Mail", theme.MailComposeIcon(), split2)

	u.overviewArea = u.NewOverviewArea()
	overviewTab := container.NewTabItemWithIcon("Characters",
		theme.NewThemedResource(resourceGroupSvg), u.overviewArea.content)

	u.skillqueueArea = u.NewSkillqueueArea()
	u.skillqueueTab = container.NewTabItemWithIcon("Training",
		theme.NewThemedResource(resourceChecklistrtlSvg), u.skillqueueArea.content)

	u.walletJournalArea = u.NewWalletJournalArea()
	u.walletTransactionArea = u.NewWalletTransactionArea()
	tabs := container.NewAppTabs(
		container.NewTabItem("Transactions", u.walletJournalArea.content),
		container.NewTabItem("Market Transactions", u.walletTransactionArea.content),
	)
	walletTab := container.NewTabItemWithIcon("Wallet",
		theme.NewThemedResource(resourceAttachmoneySvg), tabs)

	u.statusArea = u.newStatusArea()
	u.toolbarArea = u.newToolbarArea()

	u.tabs = container.NewAppTabs(u.mailTab, u.skillqueueTab, walletTab, overviewTab)
	u.tabs.SetTabLocation(container.TabLocationLeading)

	mainContent := container.NewBorder(u.toolbarArea.content, u.statusArea.content, nil, nil, u.tabs)
	w.SetContent(mainContent)
	w.SetMaster()

	var c *model.MyCharacter
	cID, ok, err := service.DictionaryInt(model.SettingLastCharacterID)
	if err == nil && ok {
		c, err = service.GetMyCharacter(int32(cID))
		if err != nil {
			if !errors.Is(err, storage.ErrNotFound) {
				slog.Error("Failed to load character", "error", err)
			}
		}
	}
	if c != nil {
		u.setCurrentCharacter(c)
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

	name, ok, err := u.service.DictionaryString(model.SettingTheme)
	if err != nil || !ok {
		name = model.ThemeAuto
	}
	u.SetTheme(name)
	return u
}

func (u *ui) SetTheme(name string) {
	switch name {
	case model.ThemeAuto:
		switch u.app.Settings().ThemeVariant() {
		case 0:
			u.app.Settings().SetTheme(theme.DarkTheme())
		default:
			u.app.Settings().SetTheme(theme.LightTheme())
		}
	case model.ThemeLight:
		u.app.Settings().SetTheme(theme.LightTheme())
	case model.ThemeDark:
		u.app.Settings().SetTheme(theme.DarkTheme())
	}
}

// ShowAndRun shows the UI and runs it (blocking).
func (u *ui) ShowAndRun() {
	go func() {
		// Workaround to mitigate a bug that causes the window to sometimes render
		// only in parts and freeze. The issue is known to happen on Linux desktops.
		if runtime.GOOS == "linux" {
			time.Sleep(1000 * time.Millisecond)
			s := u.window.Canvas().Size()
			u.window.Resize(fyne.NewSize(s.Width-0.2, s.Height-0.2))
			u.window.Resize(fyne.NewSize(s.Width, s.Height))
		}
		u.statusArea.StartUpdateTicker()
		u.overviewArea.StartUpdateTicker()
		u.folderArea.StartUpdateTicker()
		u.skillqueueArea.StartUpdateTicker()
		u.walletJournalArea.StartUpdateTicker()
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

func (u *ui) LoadCurrentCharacter(characterID int32) error {
	c, err := u.service.GetMyCharacter(characterID)
	if err != nil {
		return err
	}
	u.setCurrentCharacter(c)
	return nil
}

func (u *ui) setCurrentCharacter(c *model.MyCharacter) {
	u.currentCharacter = c
	err := u.service.DictionarySetInt(model.SettingLastCharacterID, int(c.ID))
	if err != nil {
		slog.Error("Failed to update last character setting", "characterID", c.ID)
	}
	u.refreshCurrentCharacter()
}

func (u *ui) refreshCurrentCharacter() {
	u.toolbarArea.Refresh()
	u.folderArea.Refresh()
	u.skillqueueArea.Refresh()
	u.walletJournalArea.Refresh()
	u.walletTransactionArea.Refresh()
	c := u.CurrentChar()
	if c != nil {
		u.tabs.EnableIndex(0)
		u.tabs.EnableIndex(1)
		u.tabs.EnableIndex(2)
		go u.folderArea.MaybeUpdateAndRefresh(c.ID)
		go u.overviewArea.MaybeUpdateAndRefresh(c.ID)
		go u.skillqueueArea.MaybeUpdateAndRefresh(c.ID)
		go u.walletJournalArea.MaybeUpdateAndRefresh(c.ID)
		go u.walletTransactionArea.MaybeUpdateAndRefresh(c.ID)
	} else {
		u.tabs.DisableIndex(0)
		u.tabs.DisableIndex(1)
		u.tabs.DisableIndex(2)
		u.tabs.SelectIndex(3)
	}
	u.window.Content().Refresh()
}

func (u *ui) SetAnyCharacter() error {
	c, err := u.service.GetAnyMyCharacter()
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			u.ResetCurrentCharacter()
		} else {
			return err
		}
	} else {
		u.setCurrentCharacter(c)
	}
	return nil
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
	u.refreshCurrentCharacter()
}

func (u *ui) StartUpdateTickerEveCharacters() {
	ticker := time.NewTicker(eveCharacterUpdateTicker)
	key := "eve-characters-last-updated"
	go func() {
		for {
			func() {
				lastUpdated, ok, err := u.service.DictionaryTime(key)
				if err != nil || !ok {
					return
				}
				if time.Now().Before(lastUpdated.Add(eveCharacterUpdateTimeout)) {
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
