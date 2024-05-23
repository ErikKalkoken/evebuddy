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
	myDateTime                    = "2006.01.02 15:04"
	defaultIconSize               = 32
	myFloatFormat                 = "#,###.##"
	eveDataUpdateTicker           = 60 * time.Second
	eveCharacterUpdateTimeout     = 3600 * time.Second
	eveCategorySkillUpdateTimeout = 24 * time.Hour
)

// The ui is the root object of the UI and contains all UI areas.
//
// Each UI area holds a pointer of the ui instance, so that areas can
// call methods on other UI areas and access shared variables in the UI.
type ui struct {
	app                   fyne.App
	attributesArea        *attributesArea
	biographyArea         *biographyArea
	currentCharacter      *model.Character
	imageManager          *images.Manager
	implantsArea          *implantsArea
	jumpClonesArea        *jumpClonesArea
	mailArea              *mailArea
	mailTab               *container.TabItem
	overviewArea          *overviewArea
	statusArea            *statusBarArea
	service               *service.Service
	skillCatalogueArea    *skillCatalogueArea
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

	u.attributesArea = u.NewAttributesArea()
	u.biographyArea = u.NewBiographyArea()
	u.jumpClonesArea = u.NewJumpClonesArea()
	u.implantsArea = u.NewImplantsArea()
	characterTab := container.NewTabItemWithIcon("Character",
		theme.NewThemedResource(resourcePortraitSvg), container.NewAppTabs(
			container.NewTabItem("Augmentations", u.implantsArea.content),
			container.NewTabItem("Jump Clones", u.jumpClonesArea.content),
			container.NewTabItem("Attributes", u.attributesArea.content),
			container.NewTabItem("Biography", u.biographyArea.content),
		))

	u.mailArea = u.NewMailArea()
	u.mailTab = container.NewTabItemWithIcon("Mail",
		theme.MailComposeIcon(), container.NewAppTabs(
			container.NewTabItem("Mail", u.mailArea.content),
			// container.NewTabItem("Notifications", widget.NewLabel("PLACEHOLDER")),
		))

	u.overviewArea = u.NewOverviewArea()
	overviewTab := container.NewTabItemWithIcon("Characters",
		theme.NewThemedResource(resourceGroupSvg), container.NewAppTabs(
			container.NewTabItem("Overview", u.overviewArea.content),
			// container.NewTabItem("Skills", widget.NewLabel("PLACEHOLDER")),
		))

	u.skillqueueArea = u.NewSkillqueueArea()
	u.skillCatalogueArea = u.NewSkillCatalogueArea()
	u.skillqueueTab = container.NewTabItemWithIcon("Skills",
		theme.NewThemedResource(resourceSchoolSvg), container.NewAppTabs(
			container.NewTabItem("Training Queue", u.skillqueueArea.content),
			container.NewTabItem("Skill Catalogue", u.skillCatalogueArea.content),
		))

	u.walletJournalArea = u.NewWalletJournalArea()
	u.walletTransactionArea = u.NewWalletTransactionArea()
	walletTab := container.NewTabItemWithIcon("Wallet",
		theme.NewThemedResource(resourceAttachmoneySvg), container.NewAppTabs(
			container.NewTabItem("Transactions", u.walletJournalArea.content),
			container.NewTabItem("Market Transactions", u.walletTransactionArea.content),
		))

	u.statusArea = u.newBarStatusArea()
	u.toolbarArea = u.newToolbarArea()

	u.tabs = container.NewAppTabs(characterTab, u.mailTab, u.skillqueueTab, walletTab, overviewTab)
	u.tabs.SetTabLocation(container.TabLocationLeading)

	mainContent := container.NewBorder(u.toolbarArea.content, u.statusArea.content, nil, nil, u.tabs)
	w.SetContent(mainContent)
	w.SetMaster()

	var c *model.Character
	cID, ok, err := service.DictionaryInt(model.SettingLastCharacterID)
	if err == nil && ok {
		c, err = service.GetCharacter(int32(cID))
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
		u.attributesArea.StartUpdateTicker()
		u.jumpClonesArea.StartUpdateTicker()
		u.implantsArea.StartUpdateTicker()
		u.overviewArea.StartUpdateTicker()
		u.mailArea.StartUpdateTicker()
		u.skillqueueArea.StartUpdateTicker()
		u.statusArea.StartUpdateTicker()
		u.walletJournalArea.StartUpdateTicker()
		u.walletTransactionArea.StartUpdateTicker()
		u.StartUpdateTickerEveCharacters()
		u.StartUpdateTickerEveCategorySkill()
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

func (u *ui) CurrentChar() *model.Character {
	return u.currentCharacter
}

func (u *ui) LoadCurrentCharacter(characterID int32) error {
	c, err := u.service.GetCharacter(characterID)
	if err != nil {
		return err
	}
	u.setCurrentCharacter(c)
	return nil
}

func (u *ui) setCurrentCharacter(c *model.Character) {
	u.currentCharacter = c
	err := u.service.DictionarySetInt(model.SettingLastCharacterID, int(c.ID))
	if err != nil {
		slog.Error("Failed to update last character setting", "characterID", c.ID)
	}
	u.refreshCurrentCharacter()
}

func (u *ui) refreshCurrentCharacter() {
	u.attributesArea.Refresh()
	u.biographyArea.Refresh()
	u.jumpClonesArea.Redraw()
	u.implantsArea.Refresh()
	u.mailArea.Redraw()
	u.skillqueueArea.Refresh()
	u.skillCatalogueArea.Redraw()
	u.toolbarArea.Refresh()
	u.walletJournalArea.Refresh()
	u.walletTransactionArea.Refresh()
	c := u.CurrentChar()
	if c != nil {
		u.tabs.EnableIndex(0)
		u.tabs.EnableIndex(1)
		u.tabs.EnableIndex(2)
		go u.attributesArea.MaybeUpdateAndRefresh(c.ID)
		go u.jumpClonesArea.MaybeUpdateAndRefresh(c.ID)
		go u.implantsArea.MaybeUpdateAndRefresh(c.ID)
		go u.mailArea.MaybeUpdateAndRefresh(c.ID)
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
	c, err := u.service.GetAnyCharacter()
	if errors.Is(err, storage.ErrNotFound) {
		u.ResetCurrentCharacter()
		return nil
	} else if err != nil {
		return err
	}
	u.setCurrentCharacter(c)
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
	ticker := time.NewTicker(eveDataUpdateTicker)
	key := "eve-characters-last-updated"
	go func() {
		for {
			err := func() error {
				lastUpdated, ok, err := u.service.DictionaryTime(key)
				if err != nil {
					return err
				}
				if ok && time.Now().Before(lastUpdated.Add(eveCharacterUpdateTimeout)) {
					return nil
				}
				slog.Info("Started updating eve characters")
				if err := u.service.UpdateAllEveCharactersESI(); err != nil {
					return err
				}
				slog.Info("Finished updating eve characters")
				if err := u.service.DictionarySetTime(key, time.Now()); err != nil {
					return err
				}
				return nil
			}()
			if err != nil {
				slog.Error("Failed to update eve characters: %s", err)
			}
			<-ticker.C
		}
	}()
}

func (u *ui) StartUpdateTickerEveCategorySkill() {
	ticker := time.NewTicker(eveDataUpdateTicker)
	key := "eve-category-skill-last-updated"
	go func() {
		for {
			err := func() error {
				lastUpdated, ok, err := u.service.DictionaryTime(key)
				if err != nil {
					return err
				}
				if ok && time.Now().Before(lastUpdated.Add(eveCategorySkillUpdateTimeout)) {
					return nil
				}
				slog.Info("Started updating skill category")
				if err := u.service.UpdateEveCategoryWithChildrenESI(model.EveCategoryIDSkill); err != nil {
					return err
				}
				slog.Info("Finished updating skill category")
				if err := u.service.DictionarySetTime(key, time.Now()); err != nil {
					return err
				}
				return nil
			}()
			if err != nil {
				slog.Error("Failed to update skill category: %s", err)
			}
			u.skillCatalogueArea.Refresh()
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
