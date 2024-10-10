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
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/sync/singleflight"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/character"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverse"
	"github.com/ErikKalkoken/evebuddy/internal/app/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/appdirs"
)

// UI constants
const (
	defaultIconSize = 32
	myFloatFormat   = "#,###.##"
	keyTabsMainID   = "tabs-main-id"
	keyWindowWidth  = "window-width"
	keyWindowHeight = "window-height"
)

// The ui is the root object of the UI and contains all UI areas.
//
// Each UI area holds a pointer of the ui instance, so that areas can
// call methods on other UI areas and access shared variables in the UI.
type ui struct {
	CacheService       app.CacheService
	CharacterService   *character.CharacterService
	ESIStatusService   app.ESIStatusService
	EveImageService    app.EveImageService
	EveUniverseService *eveuniverse.EveUniverseService
	StatusCacheService app.StatusCacheService
	fyneApp            fyne.App
	deskApp            desktop.App
	window             fyne.Window
	ad                 appdirs.AppDirs

	assetsArea            *assetsArea
	assetSearchArea       *assetSearchArea
	attributesArea        *attributesArea
	biographyArea         *biographyArea
	character             *app.Character
	isDebug               bool
	implantsArea          *implantsArea
	jumpClonesArea        *jumpClonesArea
	mailArea              *mailArea
	notificationsArea     *notificationsArea
	overviewArea          *overviewArea
	sfg                   *singleflight.Group
	statusBarArea         *statusBarArea
	skillCatalogueArea    *skillCatalogueArea
	skillqueueArea        *skillqueueArea
	shipsArea             *shipsArea
	statusWindow          fyne.Window
	settingsWindow        fyne.Window
	themeName             string
	toolbarArea           *toolbarArea
	walletJournalArea     *walletJournalArea
	walletTransactionArea *walletTransactionArea
	wealthArea            *wealthArea

	assetTab    *container.TabItem
	mailTab     *container.TabItem
	overviewTab *container.TabItem
	skillTab    *container.TabItem
	walletTab   *container.TabItem
	tabs        *container.AppTabs
}

// NewUI build the UI and returns it.
func NewUI(fyneApp fyne.App, ad appdirs.AppDirs, isDebug bool) *ui {
	desk, ok := fyneApp.(desktop.App)
	if !ok {
		log.Fatal("Failed to initialize as desktop app")
	}
	u := &ui{
		fyneApp: fyneApp,
		isDebug: isDebug,
		sfg:     new(singleflight.Group),
		deskApp: desk,
		ad:      ad,
	}
	u.window = fyneApp.NewWindow(u.appName())
	u.attributesArea = u.newAttributesArena()
	u.biographyArea = u.newBiographyArea()
	u.jumpClonesArea = u.NewJumpClonesArea()
	u.implantsArea = u.newImplantsArea()
	characterTab := container.NewTabItemWithIcon("Character",
		theme.AccountIcon(), container.NewAppTabs(
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
	u.notificationsArea = u.newNotificationsArea()
	u.mailTab = container.NewTabItemWithIcon("",
		theme.MailComposeIcon(), container.NewAppTabs(
			container.NewTabItem("Mail", u.mailArea.content),
			container.NewTabItem("Communications", u.notificationsArea.content),
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

	u.tabs = container.NewAppTabs(characterTab, u.assetTab, u.mailTab, u.skillTab, u.walletTab, u.overviewTab)
	u.tabs.SetTabLocation(container.TabLocationLeading)

	u.toolbarArea = u.newToolbarArea()
	u.statusBarArea = u.newStatusBarArea()
	mainContent := container.NewBorder(u.toolbarArea.content, u.statusBarArea.content, nil, nil, u.tabs)
	u.window.SetContent(mainContent)

	// Define system tray menu
	if fyneApp.Preferences().BoolWithFallback(settingSysTrayEnabled, settingSysTrayEnabledDefault) {
		name := u.appName()
		item := fyne.NewMenuItem(name, nil)
		item.Disabled = true
		m := fyne.NewMenu(
			"MyApp",
			item,
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem(fmt.Sprintf("Open %s", name), func() {
				u.window.Show()
			}),
		)
		u.deskApp.SetSystemTrayMenu(m)
		u.window.SetCloseIntercept(func() {
			u.window.Hide()
		})
	}
	menu := makeMenu(u)
	u.window.SetMainMenu(menu)
	u.window.SetMaster()
	return u
}

func (u *ui) Init() {
	var c *app.Character
	var err error
	cID := u.fyneApp.Preferences().Int(settingLastCharacterID)
	if cID != 0 {
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

	width := u.fyneApp.Preferences().FloatWithFallback(keyWindowWidth, 1000)
	height := u.fyneApp.Preferences().FloatWithFallback(keyWindowHeight, 600)
	u.window.Resize(fyne.NewSize(float32(width), float32(height)))

	if !u.hasCharacter() {
		// reset to overview tab if no character
		u.tabs.Select(u.overviewTab)
		u.overviewTab.Content.(*container.AppTabs).SelectIndex(0)
	} else {
		index := u.fyneApp.Preferences().IntWithFallback(keyTabsMainID, -1)
		if index != -1 {
			u.tabs.SelectIndex(index)
		}
		for i, o := range u.tabs.Items {
			tabs, ok := o.Content.(*container.AppTabs)
			if !ok {
				continue
			}
			key := makeSubTabsKey(i)
			index := u.fyneApp.Preferences().IntWithFallback(key, -1)
			if index != -1 {
				tabs.SelectIndex(index)
			}
		}
	}

	u.themeSet(u.fyneApp.Preferences().StringWithFallback(settingTheme, settingThemeDefault))
	u.hideMailIndicator() // init system tray icon

	u.fyneApp.Lifecycle().SetOnStopped(func() {
		slog.Info("App is shutting down")
		u.saveAppState()
	})
}

func (u *ui) saveAppState() {
	a := u.fyneApp
	if u.window == nil || a == nil {
		slog.Warn("Failed to save app state")
	}
	s := u.window.Canvas().Size()
	u.fyneApp.Preferences().SetFloat(keyWindowWidth, float64(s.Width))
	u.fyneApp.Preferences().SetFloat(keyWindowHeight, float64(s.Height))
	if u.tabs == nil {
		slog.Warn("Failed to save tabs in app state")
	}
	index := u.tabs.SelectedIndex()
	u.fyneApp.Preferences().SetInt(keyTabsMainID, index)
	for i, o := range u.tabs.Items {
		tabs, ok := o.Content.(*container.AppTabs)
		if !ok {
			continue
		}
		key := makeSubTabsKey(i)
		index := tabs.SelectedIndex()
		u.fyneApp.Preferences().SetInt(key, index)
	}
	slog.Info("Saved app state")
}

func (u *ui) themeSet(name string) {
	switch name {
	case themeAuto:
		switch u.fyneApp.Settings().ThemeVariant() {
		case 0:
			u.themeName = themeDark
		default:
			u.themeName = themeLight
		}
	case themeLight:
		u.themeName = themeLight
	case themeDark:
		u.themeName = themeDark
	}
	switch u.themeName {
	case themeDark:
		u.fyneApp.Settings().SetTheme(theme.DarkTheme())
	case themeLight:
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
	go u.refreshOverview()
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
	u.refreshCharacter()
	u.tabs.Refresh()
	u.fyneApp.Preferences().SetInt(settingLastCharacterID, int(c.ID))
}

func (u *ui) refreshCharacter() {
	u.assetsArea.redraw()
	u.assetSearchArea.refresh()
	u.attributesArea.refresh()
	u.biographyArea.refresh()
	u.jumpClonesArea.redraw()
	u.implantsArea.refresh()
	u.mailArea.redraw()
	u.notificationsArea.refresh()
	u.shipsArea.refresh()
	u.skillqueueArea.refresh()
	u.skillCatalogueArea.redraw()
	u.toolbarArea.refresh()
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
	go u.statusBarArea.refreshUpdateStatus()
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
	u.fyneApp.Preferences().SetInt(settingLastCharacterID, 0)
	u.refreshCharacter()
}

func (u *ui) showMailIndicator() {
	u.deskApp.SetSystemTrayIcon(resourceIconmarkedPng)
}

func (u *ui) hideMailIndicator() {
	u.deskApp.SetSystemTrayIcon(resourceIconPng)
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

func makeSubTabsKey(i int) string {
	return fmt.Sprintf("tabs-sub%d-id", i)
}
