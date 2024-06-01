// Package ui contains the code for rendering the UI.
package ui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/eveonline/images"
	ihttp "github.com/ErikKalkoken/evebuddy/internal/helper/http"
	"github.com/ErikKalkoken/evebuddy/internal/helper/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/service"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

// UI constants
const (
	myDateTime                    = "2006.01.02 15:04"
	defaultIconSize               = 32
	myFloatFormat                 = "#,###.##"
	characterSectionsUpdateTicker = 10 * time.Second
	eveDataUpdateTicker           = 60 * time.Second
	eveCharacterUpdateTimeout     = 3600 * time.Second
	eveCategoriesUpdateTimeout    = 24 * time.Hour
	eveCategoriesKeyLastUpdated   = "eve-categories-last-updated"
)

// The ui is the root object of the UI and contains all UI areas.
//
// Each UI area holds a pointer of the ui instance, so that areas can
// call methods on other UI areas and access shared variables in the UI.
type ui struct {
	app                   fyne.App
	assetsArea            *assetsArea
	attributesArea        *attributesArea
	biographyArea         *biographyArea
	currentCharacter      *model.Character
	imageManager          *images.Manager
	implantsArea          *implantsArea
	jumpClonesArea        *jumpClonesArea
	mailArea              *mailArea
	mailTab               *container.TabItem
	overviewArea          *overviewArea
	statusBarArea         *statusBarArea
	service               *service.Service
	skillCatalogueArea    *skillCatalogueArea
	skillqueueArea        *skillqueueArea
	skillqueueTab         *container.TabItem
	shipsArea             *shipsArea
	statusWindow          fyne.Window
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
	httpClient := &http.Client{
		Timeout:   time.Second * 30,
		Transport: ihttp.LoggedTransport{},
	}
	u := &ui{app: app,
		imageManager: images.New(imageCachePath, httpClient),
		service:      service,
		window:       w,
	}

	u.assetsArea = u.newAssetsArea()
	assetsTab := container.NewTabItemWithIcon("Assets",
		theme.NewThemedResource(resourceInventory2Svg), container.NewAppTabs(
			container.NewTabItem("Assets", u.assetsArea.content),
		))

	u.attributesArea = u.newAttributesArena()
	u.biographyArea = u.newBiographyArea()
	u.jumpClonesArea = u.NewJumpClonesArea()
	u.implantsArea = u.newImplantsArea()
	characterTab := container.NewTabItemWithIcon("Character",
		theme.NewThemedResource(resourcePortraitSvg), container.NewAppTabs(
			container.NewTabItem("Augmentations", u.implantsArea.content),
			container.NewTabItem("Jump Clones", u.jumpClonesArea.content),
			container.NewTabItem("Attributes", u.attributesArea.content),
			container.NewTabItem("Biography", u.biographyArea.content),
		))

	u.mailArea = u.newMailArea()
	u.mailTab = container.NewTabItemWithIcon("Mail",
		theme.MailComposeIcon(), container.NewAppTabs(
			container.NewTabItem("Mail", u.mailArea.content),
			// container.NewTabItem("Notifications", widget.NewLabel("PLACEHOLDER")),
		))

	u.overviewArea = u.newOverviewArea()
	overviewTab := container.NewTabItemWithIcon("Characters",
		theme.NewThemedResource(resourceGroupSvg), container.NewAppTabs(
			container.NewTabItem("Overview", u.overviewArea.content),
			// container.NewTabItem("Skills", widget.NewLabel("PLACEHOLDER")),
		))

	u.skillqueueArea = u.newSkillqueueArea()
	u.skillCatalogueArea = u.newSkillCatalogueArea()
	u.shipsArea = u.newShipArea()
	u.skillqueueTab = container.NewTabItemWithIcon("Skills",
		theme.NewThemedResource(resourceSchoolSvg), container.NewAppTabs(
			container.NewTabItem("Training Queue", u.skillqueueArea.content),
			container.NewTabItem("Skill Catalogue", u.skillCatalogueArea.content),
			container.NewTabItem("Ships", u.shipsArea.content),
		))

	u.walletJournalArea = u.newWalletJournalArea()
	u.walletTransactionArea = u.newWalletTransactionArea()
	walletTab := container.NewTabItemWithIcon("Wallet",
		theme.NewThemedResource(resourceAttachmoneySvg), container.NewAppTabs(
			container.NewTabItem("Transactions", u.walletJournalArea.content),
			container.NewTabItem("Market Transactions", u.walletTransactionArea.content),
		))

	u.statusBarArea = u.newStatusBarArea()
	u.toolbarArea = u.newToolbarArea()

	u.tabs = container.NewAppTabs(assetsTab, characterTab, u.mailTab, u.skillqueueTab, walletTab, overviewTab)
	u.tabs.SetTabLocation(container.TabLocationLeading)

	// for experiments
	// btn := widget.NewButton("Show experiment", func() {
	// 	err := errors.New("dummy")
	// 	u.showErrorDialog("An error has occurred.", err)
	// })

	mainContent := container.NewBorder(u.toolbarArea.content, u.statusBarArea.content, nil, nil, u.tabs)
	w.SetContent(mainContent)
	w.SetMaster()

	var c *model.Character
	cID, ok, err := service.Dictionary.GetInt(model.SettingLastCharacterID)
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
		u.resetCurrentCharacter()
	}
	keyW := "window-width"
	width, ok, err := u.service.Dictionary.GetFloat32(keyW)
	if err != nil || !ok {
		width = 1000
	}
	keyH := "window-height"
	height, ok, err := u.service.Dictionary.GetFloat32(keyH)
	if err != nil || !ok {
		width = 600
	}
	w.Resize(fyne.NewSize(width, height))

	keyTabsMainID := "tabs-main-id"
	index, ok, err := u.service.Dictionary.GetInt(keyTabsMainID)
	if err == nil && ok {
		u.tabs.SelectIndex(index)
	}
	makeSubTabsKey := func(i int) string {
		return fmt.Sprintf("tabs-sub%d-id", i)
	}
	for i, o := range u.tabs.Items {
		tabs, ok := o.Content.(*container.AppTabs)
		if !ok {
			continue
		}
		key := makeSubTabsKey(i)
		index, ok, err := u.service.Dictionary.GetInt(key)
		if err == nil && ok {
			tabs.SelectIndex(index)
		}
	}
	w.SetOnClosed(func() {
		s := w.Canvas().Size()
		u.service.Dictionary.SetFloat32(keyW, s.Width)
		u.service.Dictionary.SetFloat32(keyH, s.Height)
		index := u.tabs.SelectedIndex()
		u.service.Dictionary.SetInt(keyTabsMainID, index)
		for i, o := range u.tabs.Items {
			tabs, ok := o.Content.(*container.AppTabs)
			if !ok {
				continue
			}
			key := makeSubTabsKey(i)
			index := tabs.SelectedIndex()
			u.service.Dictionary.SetInt(key, index)
		}
	})

	name, ok, err := u.service.Dictionary.GetString(model.SettingTheme)
	if err != nil || !ok {
		name = model.ThemeAuto
	}
	u.setTheme(name)
	return u
}

func (u *ui) setTheme(name string) {
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
		u.statusBarArea.StartUpdateTicker()

		u.startUpdateTickerEveCategorySkill()
		u.startUpdateTickerCharacterSections()
		u.startUpdateTickerEveCharacters()
	}()
	u.refreshOverview()
	u.window.ShowAndRun()
}

func (u *ui) currentCharID() int32 {
	if u.currentCharacter == nil {
		return 0
	}
	return u.currentCharacter.ID
}

func (u *ui) currentChar() *model.Character {
	return u.currentCharacter
}

func (u *ui) hasCharacter() bool {
	return u.currentCharacter != nil
}

func (u *ui) loadCurrentCharacter(characterID int32) error {
	c, err := u.service.GetCharacter(characterID)
	if err != nil {
		return err
	}
	u.setCurrentCharacter(c)
	return nil
}

func (u *ui) setCurrentCharacter(c *model.Character) {
	u.currentCharacter = c
	err := u.service.Dictionary.SetInt(model.SettingLastCharacterID, int(c.ID))
	if err != nil {
		slog.Error("Failed to update last character setting", "characterID", c.ID)
	}
	u.refreshCurrentCharacter()
}

func (u *ui) refreshCurrentCharacter() {
	u.assetsArea.redraw()
	u.attributesArea.refresh()
	u.biographyArea.refresh()
	u.jumpClonesArea.redraw()
	u.implantsArea.refresh()
	u.mailArea.redraw()
	u.shipsArea.refresh()
	u.skillqueueArea.refresh()
	u.skillCatalogueArea.redraw()
	u.toolbarArea.refresh()
	u.walletJournalArea.refresh()
	u.walletTransactionArea.refresh()
	c := u.currentChar()
	if c != nil {
		u.tabs.EnableIndex(0)
		u.tabs.EnableIndex(1)
		u.tabs.EnableIndex(2)
		u.tabs.EnableIndex(3)
		u.updateCharacterAndRefreshIfNeeded(c.ID, false)
	} else {
		u.tabs.DisableIndex(0)
		u.tabs.DisableIndex(1)
		u.tabs.DisableIndex(2)
		u.tabs.DisableIndex(3)
		u.tabs.SelectIndex(4)
	}
	go u.statusBarArea.characterUpdateStatusArea.refresh()
	u.window.Content().Refresh()
}

func (u *ui) setAnyCharacter() error {
	c, err := u.service.GetAnyCharacter()
	if errors.Is(err, storage.ErrNotFound) {
		u.resetCurrentCharacter()
		return nil
	} else if err != nil {
		return err
	}
	u.setCurrentCharacter(c)
	return nil
}

func (u *ui) refreshOverview() {
	u.overviewArea.refresh()
}

func (u *ui) resetCurrentCharacter() {
	u.currentCharacter = nil
	err := u.service.Dictionary.Delete(model.SettingLastCharacterID)
	if err != nil {
		slog.Error("Failed to delete last character setting")
	}
	u.refreshCurrentCharacter()
}

func (u *ui) startUpdateTickerEveCharacters() {
	ticker := time.NewTicker(eveDataUpdateTicker)
	key := "eve-characters-last-updated"
	go func() {
		for {
			err := func() error {
				lastUpdated, ok, err := u.service.Dictionary.GetTime(key)
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
				if err := u.service.Dictionary.SetTime(key, time.Now()); err != nil {
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

func (u *ui) startUpdateTickerEveCategorySkill() {
	ticker := time.NewTicker(eveDataUpdateTicker)
	go func() {
		ctx := context.TODO()
		for {
			err := func() error {
				lastUpdated, ok, err := u.service.Dictionary.GetTime(eveCategoriesKeyLastUpdated)
				if err != nil {
					return err
				}
				if ok && time.Now().Before(lastUpdated.Add(eveCategoriesUpdateTimeout)) {
					return nil
				}
				slog.Info("Started updating categories")
				if err := u.service.UpdateEveCategoryWithChildrenESI(ctx, model.EveCategoryIDSkill); err != nil {
					return err
				}
				if err := u.service.UpdateEveCategoryWithChildrenESI(ctx, model.EveCategoryIDShip); err != nil {
					return err
				}
				if err := u.service.UpdateShipSkills(); err != nil {
					return err
				}
				slog.Info("Finished updating categories")
				if err := u.service.Dictionary.SetTime(eveCategoriesKeyLastUpdated, time.Now()); err != nil {
					return err
				}
				u.shipsArea.refresh()
				u.skillCatalogueArea.redraw()
				return nil
			}()
			if err != nil {
				slog.Error("Failed to update skill category: %s", err)
			}
			u.skillCatalogueArea.refresh()
			<-ticker.C
		}
	}()
}

func (u *ui) startUpdateTickerCharacterSections() {
	ticker := time.NewTicker(characterSectionsUpdateTicker)
	go func() {
		for {
			func() {
				cc, err := u.service.ListCharactersShort()
				if err != nil {
					slog.Error("Failed to fetch list of characters", "err", err)
					return
				}
				for _, c := range cc {
					u.updateCharacterAndRefreshIfNeeded(c.ID, false)
				}
			}()
			<-ticker.C
		}
	}()
}

// updateCharacterAndRefreshIfNeeded runs update for all sections of a character if needed
// and refreshes the UI accordingly.
//
// All UI areas showing data based on character sections needs to be included
// to make sure they are refreshed when data changes.
func (u *ui) updateCharacterAndRefreshIfNeeded(characterID int32, forceUpdate bool) {
	for _, s := range model.CharacterSections {
		go func(s model.CharacterSection) {
			u.updateCharacterSectionAndRefreshIfNeeded(characterID, s, forceUpdate)
		}(s)
	}
}

func (u *ui) updateCharacterSectionAndRefreshIfNeeded(characterID int32, s model.CharacterSection, forceUpdate bool) {
	hasChanged, err := u.service.UpdateCharacterSection(
		service.UpdateCharacterSectionParams{
			CharacterID: characterID,
			Section:     s,
			ForceUpdate: forceUpdate,
		})
	if err != nil {
		slog.Error("Failed to update character section", "characterID", characterID, "section", s, "err", err)
		return
	}
	isCurrent := characterID == u.currentCharID()
	switch s {
	case model.CharacterSectionAssets:
		if isCurrent && hasChanged {
			u.assetsArea.redraw()
		}
	case model.CharacterSectionAttributes:
		if isCurrent && hasChanged {
			u.attributesArea.refresh()
		}
	case model.CharacterSectionImplants:
		if isCurrent && hasChanged {
			u.implantsArea.refresh()
		}
	case model.CharacterSectionJumpClones:
		if isCurrent && hasChanged {
			u.jumpClonesArea.redraw()
		}
		if hasChanged {
			u.overviewArea.refresh()
		}
	case model.CharacterSectionLocation,
		model.CharacterSectionOnline,
		model.CharacterSectionShip,
		model.CharacterSectionWalletBalance:
		if hasChanged {
			u.overviewArea.refresh()
		}
	case model.CharacterSectionMailLabels,
		model.CharacterSectionMailLists,
		model.CharacterSectionMails:
		if isCurrent && hasChanged {
			u.mailArea.refresh()
		}
		if hasChanged {
			u.overviewArea.refresh()
		}
	case model.CharacterSectionSkills:
		if isCurrent && hasChanged {
			u.skillCatalogueArea.refresh()
			u.shipsArea.refresh()
			u.overviewArea.refresh()
		}
	case model.CharacterSectionSkillqueue:
		if isCurrent {
			u.skillqueueArea.refresh()
		}
	case model.CharacterSectionWalletJournal:
		if isCurrent && hasChanged {
			u.walletJournalArea.refresh()
		}
	case model.CharacterSectionWalletTransactions:
		if isCurrent && hasChanged {
			u.walletTransactionArea.refresh()
		}
	default:
		slog.Warn(fmt.Sprintf("section not part of the update ticker: %s", s))
	}
}

func (u *ui) showErrorDialog(message string, err error) {
	text := widget.NewLabel(fmt.Sprintf("%s\n\n%s", message, humanize.Error(err)))
	text.Wrapping = fyne.TextWrapWord
	text.Importance = widget.DangerImportance
	x := container.NewVScroll(text)
	x.SetMinSize(fyne.Size{Width: 400, Height: 100})
	d := dialog.NewCustom("Error", "OK", x, u.window)
	d.Show()
}

func appName(a fyne.App) string {
	info := a.Metadata()
	name := info.Name
	if name == "" {
		return "EVE Buddy"
	}
	return name
}
