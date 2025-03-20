// Package desktop contains the code for rendering the desktop UI.
package desktop

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"
	"github.com/icrowley/fake"
	"golang.org/x/sync/singleflight"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
)

// The DesktopUI is the root object of the DesktopUI and contains all DesktopUI areas.
type DesktopUI struct {
	*ui.BaseUI

	sfg *singleflight.Group

	statusBar *StatusBar
	toolbar   *Toolbar

	overviewTab *container.TabItem
	tabs        *container.AppTabs

	menuItemsWithShortcut []*fyne.MenuItem
	accountWindow         fyne.Window
	searchWindow          fyne.Window
	settingsWindow        fyne.Window
}

// NewDesktopUI build the UI and returns it.
func NewDesktopUI(bui *ui.BaseUI) *DesktopUI {
	u := &DesktopUI{
		sfg:    new(singleflight.Group),
		BaseUI: bui,
	}
	deskApp, ok := u.App().(desktop.App)
	if !ok {
		panic("Could not start in desktop mode")
	}
	u.OnInit = func(_ *app.Character) {
		index := u.App().Preferences().IntWithFallback(ui.SettingTabsMainID, -1)
		if index != -1 {
			u.tabs.SelectIndex(index)
			for i, o := range u.tabs.Items {
				tabs, ok := o.Content.(*container.AppTabs)
				if !ok {
					continue
				}
				key := makeSubTabsKey(i)
				index := u.App().Preferences().IntWithFallback(key, -1)
				if index != -1 {
					tabs.SelectIndex(index)
				}
			}
		}
		go u.UpdateMailIndicator()
	}
	u.OnShowAndRun = func() {
		width := float32(u.App().Preferences().FloatWithFallback(ui.SettingWindowWidth, ui.SettingWindowHeightDefault))
		height := float32(u.App().Preferences().FloatWithFallback(ui.SettingWindowHeight, ui.SettingWindowHeightDefault))
		u.MainWindow().Resize(fyne.NewSize(width, height))
	}
	u.OnAppFirstStarted = func() {
		// FIXME: Workaround to mitigate a bug that causes the window to sometimes render
		// only in parts and freeze. The issue is known to happen on Linux desktops.
		if runtime.GOOS == "linux" {
			go func() {
				time.Sleep(500 * time.Millisecond)
				s := u.MainWindow().Canvas().Size()
				u.MainWindow().Resize(fyne.NewSize(s.Width-0.2, s.Height-0.2))
				u.MainWindow().Resize(fyne.NewSize(s.Width, s.Height))
			}()
		}
		go u.statusBar.StartUpdateTicker()
		u.MainWindow().Canvas().AddShortcut(
			&desktop.CustomShortcut{
				KeyName:  fyne.KeyS,
				Modifier: fyne.KeyModifierAlt + fyne.KeyModifierControl,
			},
			func(fyne.Shortcut) {
				u.ShowSnackbar(fmt.Sprintf(
					"%s. This is a test snack bar at %s",
					fake.WordsN(10),
					time.Now().Format("15:04:05.999999999"),
				))
				u.ShowSnackbar(fmt.Sprintf(
					"This is a test snack bar at %s",
					time.Now().Format("15:04:05.999999999"),
				))
			})
	}
	u.OnAppStopped = func() {
		u.saveAppState()
	}
	u.OnUpdateCharacter = func(c *app.Character) {
		go u.toogleTabs(c != nil)
	}
	u.OnUpdateStatus = func() {
		go u.toolbar.Update()
		go u.statusBar.updateUpdateStatus()
		go u.statusBar.updateCharacterCount()
	}
	u.ShowMailIndicator = func() {
		deskApp.SetSystemTrayIcon(icons.IconmarkedPng)
	}
	u.HideMailIndicator = func() {
		deskApp.SetSystemTrayIcon(icons.IconPng)
	}
	u.EnableMenuShortcuts = u.enableMenuShortcuts
	u.DisableMenuShortcuts = u.disableMenuShortcuts

	makeTitleWithCount := func(title string, count int) string {
		if count > 0 {
			title += fmt.Sprintf(" (%s)", humanize.Comma(int64(count)))
		}
		return title
	}

	assetTab := container.NewTabItemWithIcon("Assets",
		theme.NewThemedResource(icons.Inventory2Svg), container.NewAppTabs(
			container.NewTabItem("Assets", u.CharacterAssets),
		))

	planetTab := container.NewTabItemWithIcon("Colonies",
		theme.NewThemedResource(icons.EarthSvg), container.NewAppTabs(
			container.NewTabItem("Colonies", u.CharacterPlanets),
		))
	u.CharacterPlanets.OnUpdate = func(_, expired int) {
		planetTab.Text = makeTitleWithCount("Colonies", expired)
		u.tabs.Refresh()
	}

	mailTab := container.NewTabItemWithIcon("Mail",
		theme.MailComposeIcon(), container.NewAppTabs(
			container.NewTabItem("Mail", u.CharacterMail),
			container.NewTabItem("Communications", u.CharacterCommunications),
		))
	u.CharacterMail.OnUpdate = func(count int) {
		mailTab.Text = makeTitleWithCount("Comm.", count)
		u.tabs.Refresh()
	}
	u.CharacterMail.OnSendMessage = u.showSendMailWindow

	clonesTab := container.NewTabItemWithIcon("Clones",
		theme.NewThemedResource(icons.HeadSnowflakeSvg), container.NewAppTabs(
			container.NewTabItem("Current Clone", u.CharacterImplants),
			container.NewTabItem("Jump Clones", u.CharacterJumpClones),
		))

	contractTab := container.NewTabItemWithIcon("Contracts",
		theme.NewThemedResource(icons.FileSignSvg), container.NewAppTabs(
			container.NewTabItem("Contracts", u.CharacterContracts),
		))

	overviewAssets := container.NewTabItem("Assets", u.AllAssetSearch)
	overviewTabs := container.NewAppTabs(
		container.NewTabItem("Overview", u.CharacterOverview),
		container.NewTabItem("Locations", u.LocationOverview),
		container.NewTabItem("Training", u.TrainingOverview),
		overviewAssets,
		container.NewTabItem("Colonies", u.ColonyOverview),
		container.NewTabItem("Wealth", u.WealthOverview),
	)
	overviewTabs.OnSelected = func(ti *container.TabItem) {
		if ti != overviewAssets {
			return
		}
		u.AllAssetSearch.Focus()
	}
	u.overviewTab = container.NewTabItemWithIcon("Characters",
		theme.NewThemedResource(icons.GroupSvg), overviewTabs,
	)

	skillTab := container.NewTabItemWithIcon("Skills",
		theme.NewThemedResource(icons.SchoolSvg), container.NewAppTabs(
			container.NewTabItem("Training Queue", u.CharacterSkillQueue),
			container.NewTabItem("Skill Catalogue", u.CharacterSkillCatalogue),
			container.NewTabItem("Ships", u.CharacterShips),
			container.NewTabItem("Attributes", u.CharacterAttributes),
		))
	u.CharacterSkillQueue.OnUpdate = func(status, _ string) {
		skillTab.Text = fmt.Sprintf("Skills (%s)", status)
		u.tabs.Refresh()
	}

	walletTab := container.NewTabItemWithIcon("Wallet",
		theme.NewThemedResource(icons.AttachmoneySvg), container.NewAppTabs(
			container.NewTabItem("Transactions", u.CharacterWalletJournal),
			container.NewTabItem("Market Transactions", u.CharacterWalletTransaction),
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

	u.toolbar = NewToolbar(u)
	u.statusBar = NewStatusBar(u)
	mainContent := container.NewBorder(u.toolbar, u.statusBar, nil, nil, u.tabs)
	u.MainWindow().SetContent(mainContent)

	// system tray menu
	if u.App().Preferences().BoolWithFallback(ui.SettingSysTrayEnabled, ui.SettingSysTrayEnabledDefault) {
		name := u.AppName()
		item := fyne.NewMenuItem(name, nil)
		item.Disabled = true
		m := fyne.NewMenu(
			"MyApp",
			item,
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem(fmt.Sprintf("Open %s", name), func() {
				u.MainWindow().Show()
			}),
		)
		deskApp.SetSystemTrayMenu(m)
		u.MainWindow().SetCloseIntercept(func() {
			u.MainWindow().Hide()
		})
	}
	u.HideMailIndicator() // init system tray icon

	menu := u.makeMenu()
	u.MainWindow().SetMainMenu(menu)
	u.MainWindow().SetMaster()
	return u
}

func (u *DesktopUI) saveAppState() {
	if u.MainWindow() == nil || u.App() == nil {
		slog.Warn("Failed to save app state")
	}
	s := u.MainWindow().Canvas().Size()
	u.App().Preferences().SetFloat(ui.SettingWindowWidth, float64(s.Width))
	u.App().Preferences().SetFloat(ui.SettingWindowHeight, float64(s.Height))
	if u.tabs == nil {
		slog.Warn("Failed to save tabs in app state")
	}
	index := u.tabs.SelectedIndex()
	u.App().Preferences().SetInt(ui.SettingTabsMainID, index)
	for i, o := range u.tabs.Items {
		tabs, ok := o.Content.(*container.AppTabs)
		if !ok {
			continue
		}
		key := makeSubTabsKey(i)
		index := tabs.SelectedIndex()
		u.App().Preferences().SetInt(key, index)
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
	u.App().Preferences().SetBool(ui.SettingSysTrayEnabled, ui.SettingSysTrayEnabledDefault)
	u.App().Preferences().SetBool(ui.SettingSysTrayEnabled, ui.SettingSysTrayEnabledDefault)
	u.App().Preferences().SetInt(ui.SettingTabsMainID, 0)
	u.App().Preferences().SetFloat(ui.SettingWindowHeight, ui.SettingWindowHeightDefault)
}

func makeSubTabsKey(i int) string {
	return fmt.Sprintf("tabs-sub%d-id", i)
}

func (u *DesktopUI) showSettingsWindow() {
	if u.settingsWindow != nil {
		u.settingsWindow.Show()
		return
	}
	w := u.App().NewWindow(u.MakeWindowTitle("Settings"))
	u.Settings.SetWindow(w)
	w.SetContent(u.Settings)
	w.Resize(fyne.Size{Width: 700, Height: 500})
	w.SetOnClosed(func() {
		u.settingsWindow = nil
	})
	w.Show()
}

func (u *DesktopUI) showSendMailWindow(character *app.Character, mode ui.SendMailMode, mail *app.CharacterMail) {
	title := u.MakeWindowTitle(fmt.Sprintf("New message [%s]", character.EveCharacter.Name))
	w := u.App().NewWindow(title)
	page, icon, action := ui.MakeSendMailPage(u.BaseUI, character, mode, mail, w)
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

func (u *DesktopUI) showAccountWindow() {
	if u.accountWindow != nil {
		u.accountWindow.Show()
		return
	}
	w := u.App().NewWindow(u.MakeWindowTitle("Characters"))
	u.accountWindow = w
	w.SetOnClosed(func() {
		u.accountWindow = nil
	})
	w.Resize(fyne.Size{Width: 500, Height: 300})
	w.SetContent(u.ManagerCharacters)
	u.ManagerCharacters.SetWindow(w)
	w.Show()
	u.ManagerCharacters.OnSelectCharacter = func() {
		w.Hide()
	}
}

func (u *DesktopUI) showSearchWindow() {
	if u.searchWindow != nil {
		u.searchWindow.Show()
		return
	}
	c := u.CurrentCharacter()
	var n string
	if c != nil {
		n = c.EveCharacter.Name
	} else {
		n = "No Character"
	}
	w := u.App().NewWindow(u.MakeWindowTitle(fmt.Sprintf("Search New Eden [%s]", n)))
	u.searchWindow = w
	w.SetOnClosed(func() {
		u.searchWindow = nil
	})
	w.Resize(fyne.Size{Width: 700, Height: 400})
	w.SetContent(u.GameSearch)
	w.Show()
	u.GameSearch.SetWindow(w)
	u.GameSearch.Focus()
}

func (u *DesktopUI) makeMenu() *fyne.MainMenu {
	// File menu
	fileMenu := fyne.NewMenu("File")

	// Info menu
	characterItem := fyne.NewMenuItem("Current character...", func() {
		characterID := u.CurrentCharacterID()
		if characterID == 0 {
			u.ShowSnackbar("ERROR: No character selected")
			return
		}
		u.ShowInfoWindow(app.EveEntityCharacter, characterID)
	})
	characterItem.Shortcut = &desktop.CustomShortcut{
		KeyName:  fyne.KeyC,
		Modifier: fyne.KeyModifierAlt + fyne.KeyModifierShift,
	}
	u.menuItemsWithShortcut = append(u.menuItemsWithShortcut, characterItem)

	locationItem := fyne.NewMenuItem("Current location...", func() {
		c := u.CurrentCharacter()
		if c == nil {
			u.ShowSnackbar("ERROR: No character selected")
			return
		}
		if c.Location == nil {
			u.ShowSnackbar("ERROR: Missing location for current character.")
			return
		}
		u.ShowLocationInfoWindow(c.Location.ID)
	})
	locationItem.Shortcut = &desktop.CustomShortcut{
		KeyName:  fyne.KeyL,
		Modifier: fyne.KeyModifierAlt + fyne.KeyModifierShift,
	}
	u.menuItemsWithShortcut = append(u.menuItemsWithShortcut, locationItem)

	shipItem := fyne.NewMenuItem("Current ship...", func() {
		c := u.CurrentCharacter()
		if c == nil {
			u.ShowSnackbar("ERROR: No character selected")
			return
		}
		if c.Ship == nil {
			u.ShowSnackbar("ERROR: Missing ship for current character.")
			return
		}
		u.ShowTypeInfoWindow(c.Ship.ID)
	})
	shipItem.Shortcut = &desktop.CustomShortcut{
		KeyName:  fyne.KeyS,
		Modifier: fyne.KeyModifierAlt + fyne.KeyModifierShift,
	}
	u.menuItemsWithShortcut = append(u.menuItemsWithShortcut, shipItem)

	searchItem := fyne.NewMenuItem("Search New Eden...", u.showSearchWindow)
	searchItem.Shortcut = &desktop.CustomShortcut{
		KeyName:  fyne.KeyS,
		Modifier: fyne.KeyModifierAlt,
	}
	u.menuItemsWithShortcut = append(u.menuItemsWithShortcut, searchItem)

	infoMenu := fyne.NewMenu(
		"Info",
		searchItem,
		fyne.NewMenuItemSeparator(),
		characterItem,
		locationItem,
		shipItem,
	)

	// Tools menu
	settingsItem := fyne.NewMenuItem("Settings...", u.showSettingsWindow)
	settingsItem.Shortcut = &desktop.CustomShortcut{
		KeyName:  fyne.KeyComma,
		Modifier: fyne.KeyModifierControl,
	}
	u.menuItemsWithShortcut = append(u.menuItemsWithShortcut, settingsItem)

	charactersItem := fyne.NewMenuItem("Manage characters...", u.showAccountWindow)
	charactersItem.Shortcut = &desktop.CustomShortcut{
		KeyName:  fyne.KeyC,
		Modifier: fyne.KeyModifierAlt,
	}
	u.menuItemsWithShortcut = append(u.menuItemsWithShortcut, charactersItem)

	statusItem := fyne.NewMenuItem("Update status...", u.ShowUpdateStatusWindow)
	statusItem.Shortcut = &desktop.CustomShortcut{
		KeyName:  fyne.KeyU,
		Modifier: fyne.KeyModifierAlt,
	}
	u.menuItemsWithShortcut = append(u.menuItemsWithShortcut, statusItem)

	toolsMenu := fyne.NewMenu(
		"Tools",
		charactersItem,
		fyne.NewMenuItemSeparator(),
		statusItem,
		fyne.NewMenuItemSeparator(),
		settingsItem,
	)

	// Help menu
	website := fyne.NewMenuItem("Website", func() {
		if err := u.App().OpenURL(u.WebsiteRootURL()); err != nil {
			slog.Error("open main website", "error", err)
		}
	})
	report := fyne.NewMenuItem("Report a bug", func() {
		url := u.WebsiteRootURL().JoinPath("issues")
		if err := u.App().OpenURL(url); err != nil {
			slog.Error("open issue website", "error", err)
		}
	})
	if u.IsOffline() {
		website.Disabled = true
		report.Disabled = true
	}
	helpMenu := fyne.NewMenu(
		"Help",
		website,
		report,
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("User data...", func() {
			u.showUserDataDialog()
		}), fyne.NewMenuItem("About...", func() {
			u.showAboutDialog()
		}),
	)

	u.enableMenuShortcuts()
	main := fyne.NewMainMenu(fileMenu, infoMenu, toolsMenu, helpMenu)
	return main
}

// enableMenuShortcuts enables all registered menu shortcuts.
func (u *DesktopUI) enableMenuShortcuts() {
	addShortcutFromMenuItem := func(item *fyne.MenuItem) (fyne.Shortcut, func(fyne.Shortcut)) {
		return item.Shortcut, func(s fyne.Shortcut) {
			item.Action()
		}
	}
	for _, mi := range u.menuItemsWithShortcut {
		u.MainWindow().Canvas().AddShortcut(addShortcutFromMenuItem(mi))
	}
}

// disableMenuShortcuts disabled all registered menu shortcuts.
func (u *DesktopUI) disableMenuShortcuts() {
	for _, mi := range u.menuItemsWithShortcut {
		u.MainWindow().Canvas().RemoveShortcut(mi.Shortcut)
	}
}

func (u *DesktopUI) showAboutDialog() {
	d := dialog.NewCustom("About", "Close", u.MakeAboutPage(), u.MainWindow())
	u.ModifyShortcutsForDialog(d, u.MainWindow())
	d.Show()
}

func (u *DesktopUI) showUserDataDialog() {
	f := widget.NewForm()
	type item struct {
		name string
		path string
	}
	items := make([]item, 0)
	for n, p := range u.DataPaths() {
		items = append(items, item{n, p})
	}
	items = append(items, item{"settings", u.App().Storage().RootURI().Path()})
	slices.SortFunc(items, func(a, b item) int {
		return strings.Compare(a.name, b.name)
	})
	for _, it := range items {
		f.Append(it.name, makePathEntry(u.MainWindow().Clipboard(), it.path))
	}
	d := dialog.NewCustom("User data", "Close", f, u.MainWindow())
	u.ModifyShortcutsForDialog(d, u.MainWindow())
	d.Show()
}

func makePathEntry(cb fyne.Clipboard, path string) *fyne.Container {
	p := filepath.Clean(path)
	return container.NewHBox(
		widget.NewLabel(p),
		layout.NewSpacer(),
		widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
			cb.SetContent(p)
		}))
}
