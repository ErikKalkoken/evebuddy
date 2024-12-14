// Package ui contains the code for rendering the UI.
package ui

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	kxmodal "github.com/ErikKalkoken/fyne-kx/modal"
	"golang.org/x/sync/singleflight"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/character"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverse"
	"github.com/ErikKalkoken/evebuddy/internal/appdirs"
)

// UI constants
const (
	defaultIconSize = 32
	myFloatFormat   = "#,###.##"
)

// The UI is the root object of the UI and contains all UI areas.
//
// Each UI area holds a pointer of the UI instance, so that areas can
// call methods on other UI areas and access shared variables in the UI.
type UI struct {
	CacheService       app.CacheService
	CharacterService   *character.CharacterService
	ESIStatusService   app.ESIStatusService
	EveImageService    app.EveImageService
	EveUniverseService *eveuniverse.EveUniverseService
	StatusCacheService app.StatusCacheService
	// Run the app in offline mode
	IsOffline bool
	// Whether to disable update tickers (useful for debugging)
	IsUpdateTickerDisabled bool

	ad                    appdirs.AppDirs
	assetsArea            *assetsArea
	assetSearchArea       *assetSearchArea
	assetTab              *container.TabItem
	attributesArea        *attributesArea
	biographyArea         *biographyArea
	coloniesArea          *coloniesArea
	character             *app.Character
	deskApp               desktop.App
	fyneApp               fyne.App
	implantsArea          *implantsArea
	locationsArea         *locationsArea
	jumpClonesArea        *jumpClonesArea
	mailArea              *mailArea
	mailTab               *container.TabItem
	notificationsArea     *notificationsArea
	overviewArea          *overviewArea
	overviewTab           *container.TabItem
	planetArea            *planetArea
	planetTab             *container.TabItem
	settingsWindow        fyne.Window
	sfg                   *singleflight.Group
	shipsArea             *shipsArea
	skillCatalogueArea    *skillCatalogueArea
	skillqueueArea        *skillqueueArea
	skillTab              *container.TabItem
	statusBarArea         *statusBarArea
	statusWindow          fyne.Window
	tabs                  *container.AppTabs
	toolbarArea           *toolbarArea
	walletJournalArea     *walletJournalArea
	walletTab             *container.TabItem
	walletTransactionArea *walletTransactionArea
	wealthArea            *wealthArea
	window                fyne.Window
	menuItemsWithShortcut []*fyne.MenuItem
}

// NewUI build the UI and returns it.
func NewUI(fyneApp fyne.App, ad appdirs.AppDirs) *UI {
	desk, ok := fyneApp.(desktop.App)
	if !ok {
		log.Fatal("Failed to initialize as desktop app")
	}
	u := &UI{
		ad:      ad,
		deskApp: desk,
		fyneApp: fyneApp,
		sfg:     new(singleflight.Group),
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

	u.planetArea = u.newPlanetArea()
	u.planetTab = container.NewTabItemWithIcon("Colonies",
		theme.NewThemedResource(resourceEarthSvg), container.NewAppTabs(
			container.NewTabItem("Colonies", u.planetArea.content),
		))

	u.mailArea = u.newMailArea()
	u.notificationsArea = u.newNotificationsArea()
	u.mailTab = container.NewTabItemWithIcon("",
		theme.MailComposeIcon(), container.NewAppTabs(
			container.NewTabItem("Mail", u.mailArea.content),
			container.NewTabItem("Communications", u.notificationsArea.content),
		))

	u.overviewArea = u.newOverviewArea()
	u.locationsArea = u.newLocationsArea()
	u.assetSearchArea = u.newAssetSearchArea()
	u.coloniesArea = u.newColoniesArea()
	u.wealthArea = u.newWealthArea()
	u.overviewTab = container.NewTabItemWithIcon("Characters",
		theme.NewThemedResource(resourceGroupSvg), container.NewAppTabs(
			container.NewTabItem("Overview", u.overviewArea.content),
			container.NewTabItem("Locations", u.locationsArea.content),
			container.NewTabItem("Assets", u.assetSearchArea.content),
			container.NewTabItem("Colonies", u.coloniesArea.content),
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

	u.tabs = container.NewAppTabs(
		characterTab,
		u.assetTab,
		u.mailTab,
		u.planetTab,
		u.skillTab,
		u.walletTab,
		u.overviewTab,
	)
	u.tabs.SetTabLocation(container.TabLocationLeading)

	u.toolbarArea = u.newToolbarArea()
	u.statusBarArea = u.newStatusBarArea()
	mainContent := container.NewBorder(u.toolbarArea.content, u.statusBarArea.content, nil, nil, u.tabs)
	u.window.SetContent(mainContent)

	// system tray menu
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
	u.hideMailIndicator() // init system tray icon

	menu := makeMenu(u)
	u.window.SetMainMenu(menu)
	u.window.SetMaster()
	return u
}

func (u *UI) Init() {
	var c *app.Character
	var err error
	ctx := context.Background()
	if cID := u.fyneApp.Preferences().Int(settingLastCharacterID); cID != 0 {
		c, err = u.CharacterService.GetCharacter(ctx, int32(cID))
		if err != nil {
			if !errors.Is(err, character.ErrNotFound) {
				slog.Error("Failed to load character", "error", err)
			}
		}
	}
	if c == nil {
		c, err = u.CharacterService.GetAnyCharacter(ctx)
		if err != nil {
			if !errors.Is(err, character.ErrNotFound) {
				slog.Error("Failed to load character", "error", err)
			}
		}
	}
	if c == nil {
		return
	}

	u.character = c
	index := u.fyneApp.Preferences().IntWithFallback(settingTabsMainID, -1)
	if index != -1 {
		u.tabs.SelectIndex(index)
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
}

// ShowAndRun shows the UI and runs it (blocking).
func (u *UI) ShowAndRun() {
	u.fyneApp.Lifecycle().SetOnStarted(func() {
		slog.Info("App started")

		// FIXME: Workaround to mitigate a bug that causes the window to sometimes render
		// only in parts and freeze. The issue is known to happen on Linux desktops.
		if runtime.GOOS == "linux" {
			go func() {
				time.Sleep(500 * time.Millisecond)
				s := u.window.Canvas().Size()
				u.window.Resize(fyne.NewSize(s.Width-0.2, s.Height-0.2))
				u.window.Resize(fyne.NewSize(s.Width, s.Height))
			}()
		}
		if u.IsOffline {
			slog.Info("Started in offline mode")
		}
		if u.IsUpdateTickerDisabled {
			slog.Info("Update ticker disabled")
		}
		go func() {
			u.refreshCrossPages()
			if u.hasCharacter() {
				u.setCharacter(u.character)
			} else {
				u.resetCharacter()
			}
		}()
		if !u.IsOffline && !u.IsUpdateTickerDisabled {
			go func() {
				u.startUpdateTickerGeneralSections()
				u.startUpdateTickerCharacters()
			}()
		}
		go u.statusBarArea.StartUpdateTicker()
	})
	u.fyneApp.Lifecycle().SetOnStopped(func() {
		u.saveAppState()
		slog.Info("App shut down complete")
	})
	width := float32(u.fyneApp.Preferences().FloatWithFallback(settingWindowWidth, settingWindowHeightDefault))
	height := float32(u.fyneApp.Preferences().FloatWithFallback(settingWindowHeight, settingWindowHeightDefault))
	u.window.Resize(fyne.NewSize(width, height))

	u.window.ShowAndRun()
}

func (u *UI) saveAppState() {
	a := u.fyneApp
	if u.window == nil || a == nil {
		slog.Warn("Failed to save app state")
	}
	s := u.window.Canvas().Size()
	u.fyneApp.Preferences().SetFloat(settingWindowWidth, float64(s.Width))
	u.fyneApp.Preferences().SetFloat(settingWindowHeight, float64(s.Height))
	if u.tabs == nil {
		slog.Warn("Failed to save tabs in app state")
	}
	index := u.tabs.SelectedIndex()
	u.fyneApp.Preferences().SetInt(settingTabsMainID, index)
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

// characterID returns the ID of the current character or 0 if non it set.
func (u *UI) characterID() int32 {
	if u.character == nil {
		return 0
	}
	return u.character.ID
}

func (u *UI) currentCharacter() *app.Character {
	return u.character
}

func (u *UI) hasCharacter() bool {
	return u.character != nil
}

func (u *UI) loadCharacter(ctx context.Context, characterID int32) error {
	c, err := u.CharacterService.GetCharacter(ctx, characterID)
	if err != nil {
		return err
	}
	u.setCharacter(c)
	return nil
}

func (u *UI) setCharacter(c *app.Character) {
	u.character = c
	u.refreshCharacter()
	u.fyneApp.Preferences().SetInt(settingLastCharacterID, int(c.ID))
}

func (u *UI) resetCharacter() {
	u.character = nil
	u.fyneApp.Preferences().SetInt(settingLastCharacterID, 0)
	u.refreshCharacter()
}

func (u *UI) refreshCharacter() {
	ff := map[string]func(){
		"assets":            u.assetsArea.redraw,
		"attributes":        u.attributesArea.refresh,
		"bio":               u.biographyArea.refresh,
		"implants":          u.implantsArea.refresh,
		"jumpClones":        u.jumpClonesArea.redraw,
		"mail":              u.mailArea.redraw,
		"notifications":     u.notificationsArea.refresh,
		"planets":           u.planetArea.refresh,
		"ships":             u.shipsArea.refresh,
		"skillCatalogue":    u.skillCatalogueArea.redraw,
		"skillqueue":        u.skillqueueArea.refresh,
		"toolbar":           u.toolbarArea.refresh,
		"walletJournal":     u.walletJournalArea.refresh,
		"walletTransaction": u.walletTransactionArea.refresh,
	}
	c := u.currentCharacter()
	ff["toogleTabs"] = func() {
		u.toogleTabs(c != nil)
	}
	if c != nil {
		slog.Debug("Refreshing character", "ID", c.EveCharacter.ID, "name", c.EveCharacter.Name)
	}
	runFunctionsWithProgressModal("Loading character", ff, u.window)
	if c != nil {
		u.updateCharacterAndRefreshIfNeeded(context.TODO(), c.ID, false)
	}
	go u.statusBarArea.refreshUpdateStatus()
}

func (u *UI) toogleTabs(enabled bool) {
	if enabled {
		for i := range u.tabs.Items {
			u.tabs.EnableIndex(i)
		}
		subTabs := u.overviewTab.Content.(*container.AppTabs)
		for i := range subTabs.Items {
			subTabs.EnableIndex(i)
		}
	} else {
		for i := range u.tabs.Items {
			u.tabs.DisableIndex(i)
		}
		u.tabs.Select(u.overviewTab)
		subTabs := u.overviewTab.Content.(*container.AppTabs)
		for i := range subTabs.Items {
			subTabs.DisableIndex(i)
		}
		u.overviewTab.Content.(*container.AppTabs).SelectIndex(0)
	}
	u.tabs.Refresh()
}

func (u *UI) setAnyCharacter() error {
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

// refreshCrossPages refreshed all pages under the characters tab.
func (u *UI) refreshCrossPages() {
	ff := map[string]func(){
		"assetSearch": u.assetSearchArea.refresh,
		"overview":    u.overviewArea.refresh,
		"locations":   u.locationsArea.refresh,
		"toolbar":     u.toolbarArea.refresh,
		"colony":      u.coloniesArea.refresh,
		"wealth":      u.wealthArea.refresh,
		"statusBar":   u.statusBarArea.refreshCharacterCount,
	}
	runFunctionsWithProgressModal("Updating characters", ff, u.window)
}

func runFunctionsWithProgressModal(title string, ff map[string]func(), w fyne.Window) {
	m := kxmodal.NewProgress("Updating", title, func(p binding.Float) error {
		start := time.Now()
		myLog := slog.With("title", title)
		myLog.Debug("started")
		var wg sync.WaitGroup
		var completed atomic.Int64
		for name, f := range ff {
			wg.Add(1)
			go func() {
				defer wg.Done()
				start2 := time.Now()
				f()
				x := completed.Add(1)
				if err := p.Set(float64(x)); err != nil {
					myLog.Warn("failed set progress", "error", err)
				}
				myLog.Debug("part completed", "name", name, "duration", time.Since(start2).Milliseconds())
			}()
		}
		wg.Wait()
		myLog.Debug("completed", "duration", time.Since(start).Milliseconds())
		return nil
	}, float64(len(ff)), w)
	m.Start()
}

func (u *UI) showMailIndicator() {
	u.deskApp.SetSystemTrayIcon(resourceIconmarkedPng)
}

func (u *UI) hideMailIndicator() {
	u.deskApp.SetSystemTrayIcon(resourceIconPng)
}

func (u *UI) appName() string {
	info := u.fyneApp.Metadata()
	name := info.Name
	if name == "" {
		return "EVE Buddy"
	}
	return name
}

func (u *UI) makeWindowTitle(subTitle string) string {
	return fmt.Sprintf("%s - %s", subTitle, u.appName())
}

func makeSubTabsKey(i int) string {
	return fmt.Sprintf("tabs-sub%d-id", i)
}
