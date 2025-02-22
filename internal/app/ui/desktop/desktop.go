// Package desktop contains the code for rendering the desktop UI.
package desktop

import (
	"fmt"
	"log/slog"
	"runtime"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"
	"golang.org/x/sync/singleflight"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

// The DesktopUI is the root object of the DesktopUI and contains all DesktopUI areas.
//
// Each DesktopUI area holds a pointer of the DesktopUI instance, so that areas can
// call methods on other DesktopUI areas and access shared variables in the DesktopUI.
type DesktopUI struct {
	*ui.BaseUI

	sfg *singleflight.Group

	statusBarArea *statusBarArea
	toolbarArea   *toolbarArea

	overviewTab *container.TabItem
	tabs        *container.AppTabs

	menuItemsWithShortcut []*fyne.MenuItem
	accountWindow         fyne.Window
	settingsWindow        fyne.Window
}

// NewDesktopUI build the UI and returns it.
func NewDesktopUI(bui *ui.BaseUI) *DesktopUI {
	u := &DesktopUI{
		sfg:    new(singleflight.Group),
		BaseUI: bui,
	}
	if u.DeskApp == nil {
		panic("Could not start in desktop mode")
	}
	u.OnInit = func(_ *app.Character) {
		index := u.FyneApp.Preferences().IntWithFallback(ui.SettingTabsMainID, -1)
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
		go u.UpdateMailIndicator()
	}
	u.OnShowAndRun = func() {
		width := float32(u.FyneApp.Preferences().FloatWithFallback(ui.SettingWindowWidth, ui.SettingWindowHeightDefault))
		height := float32(u.FyneApp.Preferences().FloatWithFallback(ui.SettingWindowHeight, ui.SettingWindowHeightDefault))
		u.Window.Resize(fyne.NewSize(width, height))
	}
	u.OnAppStarted = func() {
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
		sc := &desktop.CustomShortcut{
			KeyName:  fyne.KeyS,
			Modifier: fyne.KeyModifierAlt + fyne.KeyModifierControl + fyne.KeyModifierShift,
		}
		u.Window.Canvas().AddShortcut(sc, func(shortcut fyne.Shortcut) {
			sb := iwidget.NewSnackbar("This is a test snack bar!", u.Window)
			sb.Show()
		})
	}
	u.OnAppStopped = func() {
		u.saveAppState()
	}
	u.OnRefreshCharacter = func(c *app.Character) {
		go u.toolbarArea.refresh()
		go u.toogleTabs(c != nil)
		go u.statusBarArea.refreshUpdateStatus()
		go u.statusBarArea.refreshCharacterCount()
	}
	u.ShowMailIndicator = func() {
		u.DeskApp.SetSystemTrayIcon(ui.IconIconmarkedPng)
	}
	u.HideMailIndicator = func() {
		u.DeskApp.SetSystemTrayIcon(ui.IconIconPng)
	}

	showItemWindow := func(iw *ui.ItemInfoArea, err error) {
		if err != nil {
			t := "Failed to open info window"
			slog.Error(t, "err", err)
			d := ui.NewErrorDialog(t, err, u.Window)
			d.Show()
			return
		}
		if iw == nil {
			return
		}
		w := u.FyneApp.NewWindow(u.MakeWindowTitle(iw.MakeTitle("Information")))
		iw.Window = w
		w.SetContent(iw.Content)
		w.Resize(fyne.Size{Width: 500, Height: 500})
		w.Show()
	}
	u.ShowTypeInfoWindow = func(typeID, characterID int32, selectTab ui.TypeWindowTab) {
		showItemWindow(u.NewItemInfoArea(typeID, characterID, 0, selectTab))
	}
	u.ShowLocationInfoWindow = func(locationID int64) {
		showItemWindow(u.NewItemInfoArea(0, 0, locationID, ui.DescriptionTab))
	}

	makeTitleWithCount := func(title string, count int) string {
		if count > 0 {
			title += fmt.Sprintf(" (%s)", humanize.Comma(int64(count)))
		}
		return title
	}

	assetTab := container.NewTabItemWithIcon("Assets",
		theme.NewThemedResource(ui.IconInventory2Svg), container.NewAppTabs(
			container.NewTabItem("Assets", u.AssetsArea.Content),
		))

	planetTab := container.NewTabItemWithIcon("Colonies",
		theme.NewThemedResource(ui.IconEarthSvg), container.NewAppTabs(
			container.NewTabItem("Colonies", u.PlanetArea.Content),
		))
	u.PlanetArea.OnRefresh = func(_, expired int) {
		planetTab.Text = makeTitleWithCount("Colonies", expired)
		u.tabs.Refresh()
	}

	mailTab := container.NewTabItemWithIcon("",
		theme.MailComposeIcon(), container.NewAppTabs(
			container.NewTabItem("Mail", u.MailArea.Content),
			container.NewTabItem("Communications", u.NotificationsArea.Content),
		))
	u.MailArea.OnRefresh = func(count int) {
		mailTab.Text = makeTitleWithCount("Comm.", count)
		u.tabs.Refresh()
	}
	u.MailArea.OnSendMessage = u.showSendMailWindow

	clonesTab := container.NewTabItemWithIcon("Clones",
		theme.NewThemedResource(ui.IconHeadSnowflakeSvg), container.NewAppTabs(
			container.NewTabItem("Current Clone", u.ImplantsArea.Content),
			container.NewTabItem("Jump Clones", u.JumpClonesArea.Content),
		))

	contractTab := container.NewTabItemWithIcon("Contracts",
		theme.NewThemedResource(ui.IconFileSignSvg), container.NewAppTabs(
			container.NewTabItem("Contracts", u.ContractsArea.Content),
		))

	overviewAssets := container.NewTabItem("Assets", u.AssetSearchArea.Content)
	overviewTabs := container.NewAppTabs(
		container.NewTabItem("Overview", u.OverviewArea.Content),
		container.NewTabItem("Locations", u.LocationsArea.Content),
		container.NewTabItem("Training", u.TrainingArea.Content),
		overviewAssets,
		container.NewTabItem("Colonies", u.ColoniesArea.Content),
		container.NewTabItem("Wealth", u.WealthArea.Content),
	)
	overviewTabs.OnSelected = func(ti *container.TabItem) {
		if ti != overviewAssets {
			return
		}
		u.AssetSearchArea.Focus()
	}
	u.overviewTab = container.NewTabItemWithIcon("Characters",
		theme.NewThemedResource(ui.IconGroupSvg), overviewTabs,
	)

	skillTab := container.NewTabItemWithIcon("Skills",
		theme.NewThemedResource(ui.IconSchoolSvg), container.NewAppTabs(
			container.NewTabItem("Training Queue", u.SkillqueueArea.Content),
			container.NewTabItem("Skill Catalogue", u.SkillCatalogueArea.Content),
			container.NewTabItem("Ships", u.ShipsArea.Content),
			container.NewTabItem("Attributes", u.AttributesArea.Content),
		))
	u.SkillqueueArea.OnRefresh = func(status, _ string) {
		skillTab.Text = fmt.Sprintf("Skills (%s)", status)
		u.tabs.Refresh()
	}

	walletTab := container.NewTabItemWithIcon("Wallet",
		theme.NewThemedResource(ui.IconAttachmoneySvg), container.NewAppTabs(
			container.NewTabItem("Transactions", u.WalletJournalArea.Content),
			container.NewTabItem("Market Transactions", u.WalletTransactionArea.Content),
		))

	u.tabs = container.NewAppTabs(
		assetTab,
		clonesTab,
		contractTab,
		mailTab,
		planetTab,
		skillTab,
		walletTab,
		u.overviewTab,
	)
	u.tabs.SetTabLocation(container.TabLocationLeading)

	u.toolbarArea = u.newToolbarArea()
	u.statusBarArea = u.newStatusBarArea()
	mainContent := container.NewBorder(u.toolbarArea.content, u.statusBarArea.content, nil, nil, u.tabs)
	u.Window.SetContent(mainContent)

	// system tray menu
	if u.FyneApp.Preferences().BoolWithFallback(ui.SettingSysTrayEnabled, ui.SettingSysTrayEnabledDefault) {
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
		u.DeskApp.SetSystemTrayMenu(m)
		u.Window.SetCloseIntercept(func() {
			u.Window.Hide()
		})
	}
	u.HideMailIndicator() // init system tray icon

	menu := makeMenu(u)
	u.Window.SetMainMenu(menu)
	u.Window.SetMaster()
	return u
}

func (u *DesktopUI) saveAppState() {
	if u.Window == nil || u.FyneApp == nil {
		slog.Warn("Failed to save app state")
	}
	s := u.Window.Canvas().Size()
	u.FyneApp.Preferences().SetFloat(ui.SettingWindowWidth, float64(s.Width))
	u.FyneApp.Preferences().SetFloat(ui.SettingWindowHeight, float64(s.Height))
	if u.tabs == nil {
		slog.Warn("Failed to save tabs in app state")
	}
	index := u.tabs.SelectedIndex()
	u.FyneApp.Preferences().SetInt(ui.SettingTabsMainID, index)
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

func (u *DesktopUI) ResetDesktopSettings() {
	u.FyneApp.Preferences().SetBool(ui.SettingSysTrayEnabled, ui.SettingSysTrayEnabledDefault)
	u.FyneApp.Preferences().SetBool(ui.SettingSysTrayEnabled, ui.SettingSysTrayEnabledDefault)
	u.FyneApp.Preferences().SetInt(ui.SettingTabsMainID, 0)
	u.FyneApp.Preferences().SetFloat(ui.SettingWindowHeight, ui.SettingWindowHeightDefault)
}

func makeSubTabsKey(i int) string {
	return fmt.Sprintf("tabs-sub%d-id", i)
}

func (u *DesktopUI) showSettingsWindow() {
	if u.settingsWindow != nil {
		u.settingsWindow.Show()
		return
	}
	w := u.FyneApp.NewWindow(u.MakeWindowTitle("Settings"))
	u.SettingsArea.SetWindow(w)
	w.SetContent(u.SettingsArea.Content)
	w.Resize(fyne.Size{Width: 700, Height: 500})
	w.SetOnClosed(func() {
		u.settingsWindow = nil
	})
	w.Show()
}

func (u *DesktopUI) showSendMailWindow(character *app.Character, mode ui.SendMailMode, mail *app.CharacterMail) {
	title := u.MakeWindowTitle(fmt.Sprintf("New message [%s]", character.EveCharacter.Name))
	w := u.FyneApp.NewWindow(title)
	page, icon, action := u.MakeSendMailPage(character, mode, mail, w)
	send := widget.NewButtonWithIcon("Send", icon, func() {
		if action() {
			w.Hide()
		}
	})
	send.Importance = widget.HighImportance
	c := container.NewBorder(nil, container.NewHBox(send), nil, nil, page)
	w.SetContent(c)
	w.Resize(fyne.NewSize(600, 500))
	w.Show()
}
