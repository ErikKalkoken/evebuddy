// Package ui contains the code for rendering the UI.
package ui

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"runtime"
	"time"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/sync/singleflight"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/character"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverse"
	"github.com/ErikKalkoken/evebuddy/internal/app/humanize"
)

// UI constants
const (
	myDateTime      = "2006.01.02 15:04"
	defaultIconSize = 32
	myFloatFormat   = "#,###.##"
)

// The ui is the root object of the UI and contains all UI areas.
//
// Each UI area holds a pointer of the ui instance, so that areas can
// call methods on other UI areas and access shared variables in the UI.
type ui struct {
	CacheService       app.CacheService
	CharacterService   *character.CharacterService
	DictionaryService  app.DictionaryService
	ESIStatusService   app.ESIStatusService
	EveImageService    app.EveImageService
	EveUniverseService *eveuniverse.EveUniverseService
	StatusCacheService app.StatusCacheService

	fyneApp               fyne.App
	assetsArea            *assetsArea
	assetSearchArea       *assetSearchArea
	attributesArea        *attributesArea
	biographyArea         *biographyArea
	character             *app.Character
	isDebug               bool
	implantsArea          *implantsArea
	jumpClonesArea        *jumpClonesArea
	mailArea              *mailArea
	characterMenu         *fyne.Menu
	overviewArea          *overviewArea
	sfg                   *singleflight.Group
	statusBarArea         *statusBarArea
	skillCatalogueArea    *skillCatalogueArea
	skillqueueArea        *skillqueueArea
	shipsArea             *shipsArea
	statusWindow          fyne.Window
	themeName             string
	walletJournalArea     *walletJournalArea
	walletTransactionArea *walletTransactionArea
	wealthArea            *wealthArea
	window                fyne.Window

	assetTab     *container.TabItem
	characterTab *container.TabItem
	mailTab      *container.TabItem
	overviewTab  *container.TabItem
	skillTab     *container.TabItem
	walletTab    *container.TabItem
	tabs         *container.AppTabs
}

// NewUI build the UI and returns it.
func NewUI(isDebug bool) *ui {
	fyneApp := fyneapp.New()
	u := &ui{
		fyneApp: fyneApp,
		isDebug: isDebug,
		sfg:     new(singleflight.Group),
		window:  fyneApp.NewWindow(""),
	}
	u.attributesArea = u.newAttributesArena()
	u.biographyArea = u.newBiographyArea()
	u.jumpClonesArea = u.NewJumpClonesArea()
	u.implantsArea = u.newImplantsArea()
	u.characterTab = container.NewTabItemWithIcon("Character",
		resourceCharacterplaceholder32Jpeg, container.NewAppTabs(
			container.NewTabItem("Augmentations", u.implantsArea.content),
			container.NewTabItem("Jump Clones", u.jumpClonesArea.content),
			container.NewTabItem("Attributes", u.attributesArea.content),
			container.NewTabItem("Biography", u.biographyArea.content),
		))

	u.assetsArea = u.newAssetsArea()
	u.assetTab = container.NewTabItemWithIcon("Assets",
		theme.NewThemedResource(resourceInventory2Svg), container.NewAppTabs(
			container.NewTabItem("Assets", u.assetsArea.content),
		))

	u.mailArea = u.newMailArea()
	u.mailTab = container.NewTabItemWithIcon("Mail",
		theme.MailComposeIcon(), container.NewAppTabs(
			container.NewTabItem("Mail", u.mailArea.content),
			// container.NewTabItem("Notifications", widget.NewLabel("PLACEHOLDER")),
		))

	u.overviewArea = u.newOverviewArea()
	u.assetSearchArea = u.newAssetSearchArea()
	u.wealthArea = u.newWealthArea()
	u.overviewTab = container.NewTabItemWithIcon("Characters",
		theme.NewThemedResource(resourceGroupSvg), container.NewAppTabs(
			container.NewTabItem("Overview", u.overviewArea.content),
			container.NewTabItem("Assets", u.assetSearchArea.content),
			container.NewTabItem("Wealth", u.wealthArea.content),
		))

	u.skillqueueArea = u.newSkillqueueArea()
	u.skillCatalogueArea = u.newSkillCatalogueArea()
	u.shipsArea = u.newShipArea()
	u.skillTab = container.NewTabItemWithIcon("Skills",
		theme.NewThemedResource(resourceSchoolSvg), container.NewAppTabs(
			container.NewTabItem("Training Queue", u.skillqueueArea.content),
			container.NewTabItem("Skill Catalogue", u.skillCatalogueArea.content),
			container.NewTabItem("Ships", u.shipsArea.content),
		))

	u.walletJournalArea = u.newWalletJournalArea()
	u.walletTransactionArea = u.newWalletTransactionArea()
	u.walletTab = container.NewTabItemWithIcon("Wallet",
		theme.NewThemedResource(resourceAttachmoneySvg), container.NewAppTabs(
			container.NewTabItem("Transactions", u.walletJournalArea.content),
			container.NewTabItem("Market Transactions", u.walletTransactionArea.content),
		))

	u.statusBarArea = u.newStatusBarArea()

	u.tabs = container.NewAppTabs(u.characterTab, u.assetTab, u.mailTab, u.skillTab, u.walletTab, u.overviewTab)
	u.tabs.SetTabLocation(container.TabLocationLeading)

	mainContent := container.NewBorder(nil, u.statusBarArea.content, nil, nil, u.tabs)
	u.window.SetContent(mainContent)
	menu, characterMenu := makeMenu(u)
	u.characterMenu = characterMenu
	u.window.SetMainMenu(menu)
	u.window.SetMaster()
	return u
}

func (u *ui) Init() {
	var c *app.Character
	cID, ok, err := u.DictionaryService.Int(app.SettingLastCharacterID)
	if err == nil && ok {
		c, err = u.CharacterService.GetCharacter(context.TODO(), int32(cID))
		if err != nil {
			if !errors.Is(err, character.ErrNotFound) {
				slog.Error("Failed to load character", "error", err)
			}
		}
	}
	if c != nil {
		u.setCharacter(c)
	} else {
		u.resetCharacter()
	}

	keyW := "window-width"
	width, ok, err := u.DictionaryService.Float32(keyW)
	if err != nil || !ok {
		width = 1000
	}
	keyH := "window-height"
	height, ok, err := u.DictionaryService.Float32(keyH)
	if err != nil || !ok {
		width = 600
	}
	u.window.Resize(fyne.NewSize(width, height))

	keyTabsMainID := "tabs-main-id"
	index, ok, err := u.DictionaryService.Int(keyTabsMainID)
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
		index, ok, err := u.DictionaryService.Int(key)
		if err == nil && ok {
			tabs.SelectIndex(index)
		}
	}
	u.window.SetOnClosed(func() {
		s := u.window.Canvas().Size()
		u.DictionaryService.SetFloat32(keyW, s.Width)
		u.DictionaryService.SetFloat32(keyH, s.Height)
		index := u.tabs.SelectedIndex()
		u.DictionaryService.SetInt(keyTabsMainID, index)
		for i, o := range u.tabs.Items {
			tabs, ok := o.Content.(*container.AppTabs)
			if !ok {
				continue
			}
			key := makeSubTabsKey(i)
			index := tabs.SelectedIndex()
			u.DictionaryService.SetInt(key, index)
		}
	})

	name, err := u.DictionaryService.StringWithFallback(app.SettingTheme, app.SettingThemeDefault)
	if err != nil {
		name = app.SettingThemeDefault
	}
	u.themeSet(name)
}

func (u *ui) themeSet(name string) {
	switch name {
	case app.ThemeAuto:
		switch u.fyneApp.Settings().ThemeVariant() {
		case 0:
			u.themeName = app.ThemeDark
		default:
			u.themeName = app.ThemeLight
		}
	case app.ThemeLight:
		u.themeName = app.ThemeLight
	case app.ThemeDark:
		u.themeName = app.ThemeDark
	}
	switch u.themeName {
	case app.ThemeDark:
		u.fyneApp.Settings().SetTheme(theme.DarkTheme())
	case app.ThemeLight:
		u.fyneApp.Settings().SetTheme(theme.LightTheme())
	}
}

func (u *ui) themeGet() string {
	return u.themeName
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
		u.startUpdateTickerGeneralSections()
		u.startUpdateTickerCharacters()
	}()
	u.refreshOverview()
	u.window.ShowAndRun()
}

// characterID returns the ID of the current character or 0 if non it set.
func (u *ui) characterID() int32 {
	if u.character == nil {
		return 0
	}
	return u.character.ID
}

func (u *ui) currentCharacter() *app.Character {
	return u.character
}

func (u *ui) hasCharacter() bool {
	return u.character != nil
}

func (u *ui) loadCharacter(ctx context.Context, characterID int32) error {
	c, err := u.CharacterService.GetCharacter(ctx, characterID)
	if err != nil {
		return err
	}
	u.setCharacter(c)
	return nil
}

func (u *ui) setCharacter(c *app.Character) {
	u.character = c
	var s string
	if c != nil {
		s = c.EveCharacter.Name
	} else {
		s = "[No character]"
	}
	u.window.SetTitle(fmt.Sprintf("%s - %s", s, u.appName()))
	if c != nil {
		r, _ := u.EveImageService.CharacterPortrait(c.ID, 128)
		u.characterTab.Icon = r
	} else {
		u.characterTab.Icon = resourceCharacterplaceholder32Jpeg
	}
	u.refreshCharacter()
	u.tabs.Refresh()
	err := u.DictionaryService.SetInt(app.SettingLastCharacterID, int(c.ID))
	if err != nil {
		slog.Error("Failed to update last character setting", "characterID", c.ID)
	}
}

func (u *ui) refreshCharacter() {
	if err := u.refreshCharacterMenu(); err != nil {
		log.Fatalf("failed to refresh character menu: %s", err)
	}
	u.assetsArea.redraw()
	u.assetSearchArea.refresh()
	u.attributesArea.refresh()
	u.biographyArea.refresh()
	u.jumpClonesArea.redraw()
	u.implantsArea.refresh()
	u.mailArea.redraw()
	u.shipsArea.refresh()
	u.skillqueueArea.refresh()
	u.skillCatalogueArea.redraw()
	u.walletJournalArea.refresh()
	u.walletTransactionArea.refresh()
	u.wealthArea.refresh()
	c := u.currentCharacter()
	if c != nil {
		for i := range u.tabs.Items {
			u.tabs.EnableIndex(i)
		}
		subTabs := u.overviewTab.Content.(*container.AppTabs)
		for i := range subTabs.Items {
			subTabs.EnableIndex(i)
		}
		u.updateCharacterAndRefreshIfNeeded(context.TODO(), c.ID, false)
	} else {
		for i := range u.tabs.Items {
			u.tabs.DisableIndex(i)
		}
		u.tabs.Select(u.overviewTab)
		subTabs := u.overviewTab.Content.(*container.AppTabs)
		for i := range subTabs.Items {
			subTabs.DisableIndex(i)
		}
	}
	go u.statusBarArea.characterUpdateStatusArea.refresh()
	u.window.Content().Refresh()
}

func (u *ui) setAnyCharacter() error {
	c, err := u.CharacterService.GetAnyCharacter(context.TODO())
	if errors.Is(err, character.ErrNotFound) {
		u.resetCharacter()
		return nil
	} else if err != nil {
		return err
	}
	u.setCharacter(c)
	return nil
}

func (u *ui) refreshOverview() {
	u.overviewArea.refresh()
}

func (u *ui) resetCharacter() {
	u.character = nil
	err := u.DictionaryService.Delete(app.SettingLastCharacterID)
	if err != nil {
		slog.Error("Failed to delete last character setting")
	}
	u.refreshCharacter()
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

func (u *ui) appName() string {
	info := u.fyneApp.Metadata()
	name := info.Name
	if name == "" {
		return "EVE Buddy"
	}
	return name
}

func (u *ui) makeWindowTitle(subTitle string) string {
	return fmt.Sprintf("%s - %s", subTitle, u.appName())
}
