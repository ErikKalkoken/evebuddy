package ui

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
	"github.com/icrowley/fake"
	"golang.org/x/sync/singleflight"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterui"
	"github.com/ErikKalkoken/evebuddy/internal/app/desktopui"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

// The UIDesktop creates the UI for desktop.
type UIDesktop struct {
	*UIBase

	accountWindow         fyne.Window
	menuItemsWithShortcut []*fyne.MenuItem
	nav                   *iwidget.NavDrawer
	overviewTab           *container.TabItem
	searchWindow          fyne.Window
	settingsWindow        fyne.Window
	sfg                   *singleflight.Group
	statusBar             *desktopui.StatusBar
	pageBars              *desktopui.PageBarCollection
}

// NewUIDesktop build the UI and returns it.
func NewUIDesktop(bui *UIBase) *UIDesktop {
	u := &UIDesktop{
		sfg:    new(singleflight.Group),
		UIBase: bui,
	}
	deskApp, ok := u.App().(desktop.App)
	if !ok {
		panic("Could not start in desktop mode")
	}

	u.statusBar = desktopui.NewStatusBar(u)
	u.pageBars = desktopui.NewPageBarCollection(u)

	u.onInit = func(_ *app.Character) {
		index := u.Settings().TabsMainID()
		if index != -1 {
			u.nav.SelectIndex(index)
		}

		go u.UpdateMailIndicator()
	}
	u.onShowAndRun = func() {
		u.MainWindow().Resize(u.Settings().WindowSize())
	}
	u.onAppFirstStarted = func() {
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
	u.onAppStopped = func() {
		u.saveAppState()
	}
	u.onUpdateCharacter = func(c *app.Character) {
		// FIXME
		// go u.toogleTabs(c != nil)
	}
	u.onUpdateStatus = func() {
		go u.pageBars.Update()
		go u.statusBar.Update()
	}
	u.ShowMailIndicator = func() {
		deskApp.SetSystemTrayIcon(icons.IconmarkedPng)
	}
	u.HideMailIndicator = func() {
		deskApp.SetSystemTrayIcon(icons.IconPng)
	}
	u.EnableMenuShortcuts = u.enableMenuShortcuts
	u.DisableMenuShortcuts = u.disableMenuShortcuts

	formatBadge := func(v, mx int) string {
		if v == 0 {
			return ""
		}
		if v >= mx {
			return fmt.Sprintf("%d+", mx)
		}
		return fmt.Sprint(v)
	}

	// makeTitleWithCount := func(title string, count int) string {
	// 	if count > 0 {
	// 		title += fmt.Sprintf(" (%s)", humanize.Comma(int64(count)))
	// 	}
	// 	return title
	// }

	makePageWithTitle := func(title string, content fyne.CanvasObject) fyne.CanvasObject {
		bar := u.pageBars.NewPageBar(title)
		return container.NewBorder(
			bar,
			nil,
			nil,
			nil,
			content,
		)
	}

	colonies := iwidget.NewNavPage(
		"Colonies",
		theme.NewThemedResource(icons.EarthSvg),
		makePageWithTitle("Colonies", u.characterPlanets),
	)
	u.characterPlanets.OnUpdate = func(_, expired int) {
		u.nav.SetItemBadge(colonies, formatBadge(expired, 10))
	}

	mail := iwidget.NewNavPage("Mail", theme.MailComposeIcon(), makePageWithTitle("Mail", u.characterMail))
	u.characterMail.OnUpdate = func(count int) {
		u.nav.SetItemBadge(mail, formatBadge(count, 99))
	}
	u.characterMail.OnSendMessage = u.showSendMailWindow

	communications := iwidget.NewNavPage(
		"Communications",
		theme.NewThemedResource(icons.MessageSvg),
		makePageWithTitle("Communications", u.characterCommunications),
	)
	u.characterCommunications.OnUpdate = func(count int) {
		u.nav.SetItemBadge(communications, formatBadge(count, 999))
	}

	contracts := iwidget.NewNavPage(
		"Contracts",
		theme.NewThemedResource(icons.FileSignSvg),
		makePageWithTitle("Contracts", u.characterContracts),
	)

	skills := iwidget.NewNavPage(
		"Skills",
		theme.NewThemedResource(icons.SchoolSvg),
		makePageWithTitle(
			"Skills",
			container.NewAppTabs(
				container.NewTabItem("Training Queue", u.characterSkillQueue),
				container.NewTabItem("Skill Catalogue", u.characterSkillCatalogue),
				container.NewTabItem("Ships", u.characterShips),
				container.NewTabItem("Attributes", u.characterAttributes),
			)))

	u.characterSkillQueue.OnUpdate = func(status, _ string) {
		u.nav.SetItemBadge(skills, status)
	}

	// All characters

	// FIXME
	// overviewTabs.OnSelected = func(ti *container.TabItem) {
	// 	if ti != overviewAssets {
	// 		return
	// 	}
	// 	u.allAssetSearch.Focus()
	// }

	u.nav = iwidget.NewNavDrawer(
		iwidget.NewNavSectionLabel("Current Character"),
		iwidget.NewNavPage(
			"Assets",
			theme.NewThemedResource(icons.Inventory2Svg),
			makePageWithTitle("Assets", u.characterAssets),
		),
		iwidget.NewNavPage(
			"Clones",
			theme.NewThemedResource(icons.HeadSnowflakeSvg),
			makePageWithTitle("Clones", container.NewAppTabs(
				container.NewTabItem("Current Clone", u.characterImplants),
				container.NewTabItem("Jump Clones", u.characterJumpClones),
			))),
		contracts,
		communications,
		colonies,
		mail,
		skills,
		iwidget.NewNavPage("Wallet",
			theme.NewThemedResource(icons.AttachmoneySvg),
			makePageWithTitle("Wallet", container.NewAppTabs(
				container.NewTabItem("Transactions", u.characterWalletJournal),
				container.NewTabItem("Market Transactions", u.characterWalletTransaction),
			))),
		// -----------------------
		iwidget.NewNavSectionLabel("All Characters"),
		iwidget.NewNavPage(
			"Assets",
			theme.NewThemedResource(icons.Inventory2Svg),
			makePageWithTitle("Assets", u.allAssetSearch),
		),
		iwidget.NewNavPage(
			"Clones",
			theme.NewThemedResource(icons.HeadSnowflakeSvg),
			makePageWithTitle("Clones", u.cloneSearch),
		),
		iwidget.NewNavPage(
			"Colonies",
			theme.NewThemedResource(icons.EarthSvg),
			makePageWithTitle("Colonies", u.colonyOverview),
		),
		iwidget.NewNavPage(
			"Locations",
			theme.NewThemedResource(icons.MapMarkerSvg),
			makePageWithTitle("Locations", u.locationOverview),
		),
		iwidget.NewNavPage(
			"Overview",
			theme.NewThemedResource(icons.AccountMultipleSvg),
			makePageWithTitle("Overview", u.characterOverview),
		),
		iwidget.NewNavPage(
			"Training",
			theme.NewThemedResource(icons.SchoolSvg),
			makePageWithTitle("Training", u.trainingOverview),
		),
		iwidget.NewNavPage(
			"Wealth",
			theme.NewThemedResource(icons.AccountMultipleSvg),
			makePageWithTitle("Wealth", u.wealthOverview),
		),
	)

	mainContent := container.NewBorder(nil, u.statusBar, nil, nil, u.nav)
	u.MainWindow().SetContent(mainContent)

	// system tray menu
	if u.Settings().SysTrayEnabled() {
		name := u.appName()
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
	return u
}

func (u *UIDesktop) saveAppState() {
	if u.MainWindow() == nil || u.App() == nil {
		slog.Warn("Failed to save app state")
	}
	u.Settings().SetWindowSize(u.MainWindow().Canvas().Size())
	u.Settings().SetTabsMainID(u.nav.SelectedIndex())

	slog.Info("Saved app state")
}

func (u *UIDesktop) toogleTabs(enabled bool) {
	// FIXME
	// if enabled {
	// 	for i := range u.tabs.Items {
	// 		u.tabs.EnableIndex(i)
	// 	}
	// 	subTabs := u.overviewTab.Content.(*container.AppTabs)
	// 	for i := range subTabs.Items {
	// 		subTabs.EnableIndex(i)
	// 	}
	// } else {
	// 	for i := range u.tabs.Items {
	// 		u.tabs.DisableIndex(i)
	// 	}
	// 	u.tabs.Select(u.overviewTab)
	// 	subTabs := u.overviewTab.Content.(*container.AppTabs)
	// 	for i := range subTabs.Items {
	// 		subTabs.DisableIndex(i)
	// 	}
	// 	u.overviewTab.Content.(*container.AppTabs).SelectIndex(0)
	// }
	// u.tabs.Refresh()
}

func (u *UIDesktop) ResetDesktopSettings() {
	u.Settings().ResetTabsMainID()
	u.Settings().ResetWindowSize()
	u.Settings().ResetSysTrayEnabled()
}

func makeSubTabsKey(i int) string {
	return fmt.Sprintf("tabs-sub%d-id", i)
}

func (u *UIDesktop) showSettingsWindow() {
	if u.settingsWindow != nil {
		u.settingsWindow.Show()
		return
	}
	w := u.App().NewWindow(u.MakeWindowTitle("Settings"))
	u.userSettings.SetWindow(w)
	w.SetContent(u.userSettings)
	w.Resize(fyne.Size{Width: 700, Height: 500})
	w.SetOnClosed(func() {
		u.settingsWindow = nil
	})
	w.Show()
}

func (u *UIDesktop) showSendMailWindow(c *app.Character, mode app.SendMailMode, mail *app.CharacterMail) {
	title := fmt.Sprintf("New message [%s]", c.EveCharacter.Name)
	w := u.App().NewWindow(u.MakeWindowTitle(title))
	page := characterui.NewSendMail(u, c, mode, mail)
	page.SetWindow(w)
	send := widget.NewButtonWithIcon("Send", theme.MailSendIcon(), func() {
		if page.SendAction() {
			w.Hide()
		}
	})
	send.Importance = widget.HighImportance
	p := theme.Padding()
	x := container.NewBorder(
		nil,
		container.NewCenter(container.New(layout.NewCustomPaddedLayout(p, p, 0, 0), send)),
		nil,
		nil,
		page,
	)
	w.SetContent(x)
	w.Resize(fyne.NewSize(600, 500))
	w.Show()
}

func (u *UIDesktop) ShowManageCharactersWindow() {
	if u.accountWindow != nil {
		u.accountWindow.Show()
		return
	}
	w := u.App().NewWindow(u.MakeWindowTitle("Manage Characters"))
	u.accountWindow = w
	w.SetOnClosed(func() {
		u.accountWindow = nil
	})
	w.Resize(fyne.Size{Width: 500, Height: 300})
	w.SetContent(u.managerCharacters)
	u.managerCharacters.SetWindow(w)
	w.Show()
	u.managerCharacters.OnSelectCharacter = func() {
		w.Hide()
	}
}

func (u *UIDesktop) showSearchWindow() {
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
	w.SetContent(u.gameSearch)
	w.Show()
	u.gameSearch.SetWindow(w)
	u.gameSearch.Focus()
}

func (u *UIDesktop) makeMenu() *fyne.MainMenu {
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

	charactersItem := fyne.NewMenuItem("Manage characters...", u.ShowManageCharactersWindow)
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
func (u *UIDesktop) enableMenuShortcuts() {
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
func (u *UIDesktop) disableMenuShortcuts() {
	for _, mi := range u.menuItemsWithShortcut {
		u.MainWindow().Canvas().RemoveShortcut(mi.Shortcut)
	}
}

func (u *UIDesktop) showAboutDialog() {
	d := dialog.NewCustom("About", "Close", u.makeAboutPage(), u.MainWindow())
	u.ModifyShortcutsForDialog(d, u.MainWindow())
	d.Show()
}

func (u *UIDesktop) showUserDataDialog() {
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
