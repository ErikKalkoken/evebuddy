// Package desktop contains the code for rendering the desktop UI.
package desktop

import (
	"fmt"
	"log/slog"
	"runtime"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	fyneDesktop "fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
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
	u.BaseUI = ui.NewBaseUI(fyneApp)
	u.identifyDesktop()
	characterTab := container.NewTabItemWithIcon("Character",
		theme.AccountIcon(), container.NewAppTabs(
			container.NewTabItem("Augmentations", u.ImplantsArea.Content),
			container.NewTabItem("Jump Clones", u.JumpClonesArea.Content),
			container.NewTabItem("Attributes", u.AttributesArea.Content),
			container.NewTabItem("Biography", u.BiographyArea.Content),
		))

	u.assetTab = container.NewTabItemWithIcon("Assets",
		theme.NewThemedResource(ui.IconInventory2Svg), container.NewAppTabs(
			container.NewTabItem("Assets", u.AssetsArea.Content),
		))

	u.planetTab = container.NewTabItemWithIcon("Colonies",
		theme.NewThemedResource(ui.IconEarthSvg), container.NewAppTabs(
			container.NewTabItem("Colonies", u.PlanetArea.Content),
		))

	u.mailTab = container.NewTabItemWithIcon("",
		theme.MailComposeIcon(), container.NewAppTabs(
			container.NewTabItem("Mail", u.MailArea.Content),
			container.NewTabItem("Communications", u.NotificationsArea.Content),
		))

	contractTab := container.NewTabItemWithIcon("Contracts",
		theme.NewThemedResource(ui.IconFileSignSvg), container.NewAppTabs(
			container.NewTabItem("Contracts", u.ContractsArea.Content),
		))

	u.overviewTab = container.NewTabItemWithIcon("Characters",
		theme.NewThemedResource(ui.IconGroupSvg), container.NewAppTabs(
			container.NewTabItem("Overview", u.OverviewArea.Content),
			container.NewTabItem("Locations", u.LocationsArea.Content),
			container.NewTabItem("Training", u.TrainingArea.Content),
			container.NewTabItem("Assets", u.AssetSearchArea.Content),
			container.NewTabItem("Colonies", u.ColoniesArea.Content),
			container.NewTabItem("Wealth", u.WealthArea.Content),
		))

	u.skillTab = container.NewTabItemWithIcon("Skills",
		theme.NewThemedResource(ui.IconSchoolSvg), container.NewAppTabs(
			container.NewTabItem("Training Queue", u.SkillqueueArea.Content),
			container.NewTabItem("Skill Catalogue", u.SkillCatalogueArea.Content),
			container.NewTabItem("Ships", u.ShipsArea.Content),
		))

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
		go u.statusBarArea.StartUpdateTicker()
	})
	u.FyneApp.Lifecycle().SetOnStopped(func() {
		u.saveAppState()
	})
	width := float32(u.FyneApp.Preferences().FloatWithFallback(settingWindowWidth, settingWindowHeightDefault))
	height := float32(u.FyneApp.Preferences().FloatWithFallback(settingWindowHeight, settingWindowHeightDefault))
	u.Window.Resize(fyne.NewSize(width, height))

	u.BaseUI.ShowAndRun()
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
