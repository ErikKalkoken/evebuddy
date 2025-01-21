// Package desktop contains the code for rendering the desktop UI.
package desktop

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	fyneDesktop "fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	kxmodal "github.com/ErikKalkoken/fyne-kx/modal"
	"golang.org/x/sync/singleflight"

	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
)

// Desktop UI constants
const (
	defaultIconSize = 32
	myFloatFormat   = "#,###.##"
)

// The DesktopUI is the root object of the DesktopUI and contains all DesktopUI areas.
//
// Each DesktopUI area holds a pointer of the DesktopUI instance, so that areas can
// call methods on other DesktopUI areas and access shared variables in the DesktopUI.
type DesktopUI struct {
	*ui.BaseUI

	// Paths to user data (for information only)
	DataPaths map[string]string

	deskApp fyneDesktop.App
	sfg     *singleflight.Group

	AssetsArea            *AssetsArea
	AssetSearchArea       *AssetSearchArea
	BiographyArea         *BiographyArea
	ColoniesArea          *ColoniesArea
	ContractsArea         *ContractsArea
	ImplantsArea          *ImplantsArea
	JumpClonesArea        *JumpClonesArea
	LocationsArea         *LocationsArea
	MailArea              *MailArea
	NotificationsArea     *NotificationsArea
	OverviewArea          *OverviewArea
	PlanetArea            *PlanetArea
	ShipsArea             *ShipsArea
	SkillCatalogueArea    *SkillCatalogueArea
	SkillqueueArea        *SkillqueueArea
	TrainingArea          *TrainingArea
	WalletJournalArea     *WalletJournalArea
	WalletTransactionArea *WalletTransactionArea
	WealthArea            *WealthArea

	statusBarArea *statusBarArea
	toolbarArea   *toolbarArea

	assetTab    *container.TabItem
	mailTab     *container.TabItem
	overviewTab *container.TabItem
	planetTab   *container.TabItem
	skillTab    *container.TabItem
	tabs        *container.AppTabs
	walletTab   *container.TabItem

	menuItemsWithShortcut []*fyne.MenuItem
	accountWindow         fyne.Window
	settingsWindow        fyne.Window
	statusWindow          fyne.Window
}

// NewDesktopUI build the UI and returns it.
func NewDesktopUI(fyneApp fyne.App) *DesktopUI {
	u := &DesktopUI{
		sfg: new(singleflight.Group),
	}
	u.BaseUI = ui.NewBaseUI(fyneApp, u.RefreshCharacter, u.RefreshCrossPages)
	u.identifyDesktop()
	u.AccountArea = u.NewAccountArea(u.UpdateCharacterAndRefreshIfNeeded)
	u.BiographyArea = u.NewBiographyArea()
	u.JumpClonesArea = u.NewJumpClonesArea()
	u.ImplantsArea = u.NewImplantsArea()
	characterTab := container.NewTabItemWithIcon("Character",
		theme.AccountIcon(), container.NewAppTabs(
			container.NewTabItem("Augmentations", u.ImplantsArea.Content),
			container.NewTabItem("Jump Clones", u.JumpClonesArea.Content),
			container.NewTabItem("Attributes", u.AttributesArea.Content),
			container.NewTabItem("Biography", u.BiographyArea.Content),
		))

	u.AssetsArea = u.NewAssetsArea()
	u.assetTab = container.NewTabItemWithIcon("Assets",
		theme.NewThemedResource(ui.IconInventory2Svg), container.NewAppTabs(
			container.NewTabItem("Assets", u.AssetsArea.Content),
		))

	u.PlanetArea = u.NewPlanetArea()
	u.planetTab = container.NewTabItemWithIcon("Colonies",
		theme.NewThemedResource(ui.IconEarthSvg), container.NewAppTabs(
			container.NewTabItem("Colonies", u.PlanetArea.Content),
		))

	u.MailArea = u.NewMailArea()
	u.NotificationsArea = u.NewNotificationsArea()
	u.mailTab = container.NewTabItemWithIcon("",
		theme.MailComposeIcon(), container.NewAppTabs(
			container.NewTabItem("Mail", u.MailArea.Content),
			container.NewTabItem("Communications", u.NotificationsArea.Content),
		))

	u.ContractsArea = u.NewContractsArea()
	contractTab := container.NewTabItemWithIcon("Contracts",
		theme.NewThemedResource(ui.IconFileSignSvg), container.NewAppTabs(
			container.NewTabItem("Contracts", u.ContractsArea.Content),
		))

	u.OverviewArea = u.NewOverviewArea()
	u.LocationsArea = u.NewLocationsArea()
	u.TrainingArea = u.NewTrainingArea()
	u.AssetSearchArea = u.NewAssetSearchArea()
	u.ColoniesArea = u.NewColoniesArea()
	u.WealthArea = u.NewWealthArea()
	u.overviewTab = container.NewTabItemWithIcon("Characters",
		theme.NewThemedResource(ui.IconGroupSvg), container.NewAppTabs(
			container.NewTabItem("Overview", u.OverviewArea.Content),
			container.NewTabItem("Locations", u.LocationsArea.Content),
			container.NewTabItem("Training", u.TrainingArea.Content),
			container.NewTabItem("Assets", u.AssetSearchArea.Content),
			container.NewTabItem("Colonies", u.ColoniesArea.Content),
			container.NewTabItem("Wealth", u.WealthArea.Content),
		))

	u.SkillqueueArea = u.NewSkillqueueArea()
	u.SkillCatalogueArea = u.NewSkillCatalogueArea()
	u.ShipsArea = u.newShipArea()
	u.skillTab = container.NewTabItemWithIcon("Skills",
		theme.NewThemedResource(ui.IconSchoolSvg), container.NewAppTabs(
			container.NewTabItem("Training Queue", u.SkillqueueArea.Content),
			container.NewTabItem("Skill Catalogue", u.SkillCatalogueArea.Content),
			container.NewTabItem("Ships", u.ShipsArea.Content),
		))

	u.WalletJournalArea = u.NewWalletJournalArea()
	u.WalletTransactionArea = u.NewWalletTransactionArea()
	u.walletTab = container.NewTabItemWithIcon("Wallet",
		theme.NewThemedResource(ui.IconAttachmoneySvg), container.NewAppTabs(
			container.NewTabItem("Transactions", u.WalletJournalArea.Content),
			container.NewTabItem("Market Transactions", u.WalletTransactionArea.Content),
		))

	u.tabs = container.NewAppTabs(
		characterTab,
		u.assetTab,
		contractTab,
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
	u.Window.SetContent(mainContent)

	// system tray menu
	if u.isDesktop() && fyneApp.Preferences().BoolWithFallback(settingSysTrayEnabled, settingSysTrayEnabledDefault) {
		name := u.AppName()
		item := fyne.NewMenuItem(name, nil)
		item.Disabled = true
		m := fyne.NewMenu(
			"MyApp",
			item,
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem(fmt.Sprintf("Open %s", name), func() {
				u.Window.Show()
			}),
		)
		u.deskApp.SetSystemTrayMenu(m)
		u.Window.SetCloseIntercept(func() {
			u.Window.Hide()
		})
	}
	u.hideMailIndicator() // init system tray icon

	menu := makeMenu(u)
	u.Window.SetMainMenu(menu)
	u.Window.SetMaster()
	return u
}

func (u *DesktopUI) identifyDesktop() {
	desk, ok := u.FyneApp.(fyneDesktop.App)
	if ok {
		slog.Debug("Running in desktop mode")
		u.deskApp = desk
	} else {
		slog.Debug("Running in mobile mode")
	}
}

func (u *DesktopUI) isDesktop() bool {
	return u.deskApp != nil
}

func (u *DesktopUI) Init() {
	u.BaseUI.Init()
	index := u.FyneApp.Preferences().IntWithFallback(settingTabsMainID, -1)
	if index != -1 {
		u.tabs.SelectIndex(index)
		for i, o := range u.tabs.Items {
			tabs, ok := o.Content.(*container.AppTabs)
			if !ok {
				continue
			}
			key := makeSubTabsKey(i)
			index := u.FyneApp.Preferences().IntWithFallback(key, -1)
			if index != -1 {
				tabs.SelectIndex(index)
			}
		}
	}
}

// ShowAndRun shows the UI and runs it (blocking).
func (u *DesktopUI) ShowAndRun() {
	u.FyneApp.Lifecycle().SetOnStarted(func() {
		slog.Info("App started")

		// FIXME: Workaround to mitigate a bug that causes the window to sometimes render
		// only in parts and freeze. The issue is known to happen on Linux desktops.
		if runtime.GOOS == "linux" {
			go func() {
				time.Sleep(500 * time.Millisecond)
				s := u.Window.Canvas().Size()
				u.Window.Resize(fyne.NewSize(s.Width-0.2, s.Height-0.2))
				u.Window.Resize(fyne.NewSize(s.Width, s.Height))
			}()
		}
		if u.IsOffline {
			slog.Info("Started in offline mode")
		}
		if u.IsUpdateTickerDisabled {
			slog.Info("Update ticker disabled")
		}
		go func() {
			u.RefreshCrossPages()
			if u.HasCharacter() {
				u.SetCharacter(u.Character)
			} else {
				u.ResetCharacter()
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
	u.FyneApp.Lifecycle().SetOnStopped(func() {
		u.saveAppState()
		slog.Info("App shut down complete")
	})
	width := float32(u.FyneApp.Preferences().FloatWithFallback(settingWindowWidth, settingWindowHeightDefault))
	height := float32(u.FyneApp.Preferences().FloatWithFallback(settingWindowHeight, settingWindowHeightDefault))
	u.Window.Resize(fyne.NewSize(width, height))

	u.Window.ShowAndRun()
}

func (u *DesktopUI) saveAppState() {
	a := u.FyneApp
	if u.Window == nil || a == nil {
		slog.Warn("Failed to save app state")
	}
	s := u.Window.Canvas().Size()
	u.FyneApp.Preferences().SetFloat(settingWindowWidth, float64(s.Width))
	u.FyneApp.Preferences().SetFloat(settingWindowHeight, float64(s.Height))
	if u.tabs == nil {
		slog.Warn("Failed to save tabs in app state")
	}
	index := u.tabs.SelectedIndex()
	u.FyneApp.Preferences().SetInt(settingTabsMainID, index)
	for i, o := range u.tabs.Items {
		tabs, ok := o.Content.(*container.AppTabs)
		if !ok {
			continue
		}
		key := makeSubTabsKey(i)
		index := tabs.SelectedIndex()
		u.FyneApp.Preferences().SetInt(key, index)
	}
	slog.Info("Saved app state")
}

func (u *DesktopUI) RefreshCharacter() {
	ff := map[string]func(){
		"assets":            u.AssetsArea.Redraw,
		"attributes":        u.AttributesArea.Refresh,
		"bio":               u.BiographyArea.refresh,
		"contracts":         u.ContractsArea.Refresh,
		"implants":          u.ImplantsArea.Refresh,
		"jumpClones":        u.JumpClonesArea.redraw,
		"mail":              u.MailArea.Redraw,
		"notifications":     u.NotificationsArea.Refresh,
		"planets":           u.PlanetArea.Refresh,
		"ships":             u.ShipsArea.Refresh,
		"skillCatalogue":    u.SkillCatalogueArea.redraw,
		"skillqueue":        u.SkillqueueArea.Refresh,
		"toolbar":           u.toolbarArea.refresh,
		"walletJournal":     u.WalletJournalArea.Refresh,
		"walletTransaction": u.WalletTransactionArea.Refresh,
	}
	c := u.CurrentCharacter()
	ff["toogleTabs"] = func() {
		u.toogleTabs(c != nil)
	}
	if c != nil {
		slog.Debug("Refreshing character", "ID", c.EveCharacter.ID, "name", c.EveCharacter.Name)
	}
	runFunctionsWithProgressModal("Loading character", ff, u.Window)
	if c != nil && !u.IsUpdateTickerDisabled {
		u.UpdateCharacterAndRefreshIfNeeded(context.TODO(), c.ID, false)
	}
	go u.statusBarArea.refreshUpdateStatus()
}

func (u *DesktopUI) toogleTabs(enabled bool) {
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

// RefreshCrossPages refreshed all pages under the characters tab.
func (u *DesktopUI) RefreshCrossPages() {
	ff := map[string]func(){
		"assetSearch": u.AssetSearchArea.Refresh,
		"colony":      u.ColoniesArea.Refresh,
		"locations":   u.LocationsArea.Refresh,
		"overview":    u.OverviewArea.Refresh,
		"statusBar":   u.statusBarArea.refreshCharacterCount,
		"toolbar":     u.toolbarArea.refresh,
		"training":    u.TrainingArea.Refresh,
		"wealth":      u.WealthArea.Refresh,
	}
	runFunctionsWithProgressModal("Updating characters", ff, u.Window)
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

func (u *DesktopUI) showMailIndicator() {
	if !u.isDesktop() {
		return
	}
	u.deskApp.SetSystemTrayIcon(ui.IconIconmarkedPng)
}

func (u *DesktopUI) hideMailIndicator() {
	if !u.isDesktop() {
		return
	}
	u.deskApp.SetSystemTrayIcon(ui.IconIconPng)
}

func (u *DesktopUI) makeWindowTitle(subTitle string) string {
	return fmt.Sprintf("%s - %s", subTitle, u.AppName())
}

func makeSubTabsKey(i int) string {
	return fmt.Sprintf("tabs-sub%d-id", i)
}

// func (u *DesktopUI) ShowAccountDialog() {
// 	err := func() error {
// 		currentChars := set.New[int32]()
// 		cc, err := u.CharacterService.ListCharactersShort(context.Background())
// 		if err != nil {
// 			return err
// 		}
// 		for _, c := range cc {
// 			currentChars.Add(c.ID)
// 		}
// 		a := u.NewAccountArea(u.updateCharacterAndRefreshIfNeeded)
// 		d := dialog.NewCustom("Manage Characters", "Close", a.Content, u.Window)
// 		kxdialog.AddDialogKeyHandler(d, u.Window)
// 		a.OnSelectCharacter = func() {
// 			d.Hide()
// 		}
// 		d.SetOnClosed(func() {
// 			defer u.enableMenuShortcuts()
// 			// incomingChars := set.New[int32]()
// 			// for _, c := range a.characters {
// 			// 	incomingChars.Add(c.id)
// 			// }
// 			// if currentChars.Equal(incomingChars) {
// 			// 	return
// 			// }
// 			// if !incomingChars.Contains(u.CharacterID()) {
// 			// 	if err := u.SetAnyCharacter(); err != nil {
// 			// 		slog.Error("Failed to set any character", "error", err)
// 			// 	}
// 			// }
// 			u.refreshCrossPages()
// 		})
// 		u.disableMenuShortcuts()
// 		d.Show()
// 		d.Resize(fyne.Size{Width: 500, Height: 500})
// 		if err := a.Refresh(); err != nil {
// 			d.Hide()
// 			return err
// 		}
// 		return nil
// 	}()
// 	if err != nil {
// 		d := ui.NewErrorDialog("Failed to show account dialog", err, u.Window)
// 		d.Show()
// 	}
// }
