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
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type shortcutDef struct {
	shortcut fyne.Shortcut
	handler  func(shortcut fyne.Shortcut)
}

// The DesktopUI creates the UI for desktop.
type DesktopUI struct {
	*BaseUI

	accountWindow  fyne.Window
	searchWindow   fyne.Window
	settingsWindow fyne.Window

	shortcuts map[string]shortcutDef
	sfg       *singleflight.Group
}

// NewUIDesktop build the UI and returns it.
func NewUIDesktop(bui *BaseUI) *DesktopUI {
	u := &DesktopUI{
		sfg:    new(singleflight.Group),
		BaseUI: bui,
	}
	deskApp, ok := u.App().(desktop.App)
	if !ok {
		panic("Could not start in desktop mode")
	}

	u.ShowMailIndicator = func() {
		deskApp.SetSystemTrayIcon(icons.IconmarkedPng)
	}
	u.HideMailIndicator = func() {
		deskApp.SetSystemTrayIcon(icons.IconPng)
	}
	u.EnableMenuShortcuts = u.enableShortcuts
	u.DisableMenuShortcuts = u.disableShortcuts

	u.showManageCharacters = u.showManageCharactersWindow

	u.defineShortcuts()
	statusBar := NewStatusBar(u)
	pageBars := NewPageBarCollection(u)

	var characterNav *iwidget.NavDrawer

	formatBadge := func(v, mx int) string {
		if v == 0 {
			return ""
		}
		if v >= mx {
			return fmt.Sprintf("%d+", mx)
		}
		return fmt.Sprint(v)
	}

	makePageWithPageBar := func(title string, content fyne.CanvasObject, buttons ...*widget.Button) fyne.CanvasObject {
		bar := pageBars.NewPageBar(title, buttons...)
		return container.NewBorder(
			bar,
			nil,
			nil,
			nil,
			content,
		)
	}

	// current character

	characterAssets := iwidget.NewNavPage(
		"Assets",
		theme.NewThemedResource(icons.Inventory2Svg),
		makePageWithPageBar("Assets", u.characterAssets),
	)
	character := iwidget.NewNavPage(
		"Character Sheet",
		theme.NewThemedResource(icons.PortraitSvg),
		makePageWithPageBar("Character Sheet", container.NewAppTabs(
			container.NewTabItem("Augmentations", u.characterImplants),
			container.NewTabItem("Jump Clones", u.characterJumpClones),
			container.NewTabItem("Attributes", u.characterAttributes),
			container.NewTabItem("Biography", u.characterBiography),
		)))

	colonies := iwidget.NewNavPage(
		"Colonies",
		theme.NewThemedResource(icons.EarthSvg),
		makePageWithPageBar("Colonies", u.characterPlanets),
	)
	u.characterPlanets.OnUpdate = func(_, expired int) {
		characterNav.SetItemBadge(colonies, formatBadge(expired, 10))
	}

	r, f := u.characterMail.MakeComposeMessageAction()
	compose := widget.NewButtonWithIcon("Compose", r, f)
	compose.Importance = widget.HighImportance
	mail := iwidget.NewNavPage(
		"Mail",
		theme.MailComposeIcon(),
		makePageWithPageBar("Mail", u.characterMail, compose),
	)
	u.characterMail.OnUpdate = func(count int) {
		characterNav.SetItemBadge(mail, formatBadge(count, 99))
	}
	u.characterMail.OnSendMessage = u.showSendMailWindow

	communications := iwidget.NewNavPage(
		"Communications",
		theme.NewThemedResource(icons.MessageSvg),
		makePageWithPageBar("Communications", u.characterCommunications),
	)
	u.characterCommunications.OnUpdate = func(count int) {
		characterNav.SetItemBadge(communications, formatBadge(count, 999))
	}

	contracts := iwidget.NewNavPage(
		"Contracts",
		theme.NewThemedResource(icons.FileSignSvg),
		makePageWithPageBar("Contracts", u.characterContracts),
	)

	skills := iwidget.NewNavPage(
		"Skills",
		theme.NewThemedResource(icons.SchoolSvg),
		makePageWithPageBar(
			"Skills",
			container.NewAppTabs(
				container.NewTabItem("Training Queue", u.characterSkillQueue),
				container.NewTabItem("Skill Catalogue", u.characterSkillCatalogue),
				container.NewTabItem("Ships", u.characterShips),
			)))

	u.characterSkillQueue.OnUpdate = func(status, _ string) {
		characterNav.SetItemBadge(skills, status)
	}

	wallet := iwidget.NewNavPage("Wallet",
		theme.NewThemedResource(icons.AttachmoneySvg),
		makePageWithPageBar("Wallet", container.NewAppTabs(
			container.NewTabItem("Transactions", u.characterWalletJournal),
			container.NewTabItem("Market Transactions", u.characterWalletTransaction),
		)))

	// nav bar

	characterNav = iwidget.NewNavDrawer("Current Character",
		character,
		characterAssets,
		contracts,
		communications,
		colonies,
		mail,
		skills,
		wallet,
	)

	makePageWithTitle := func(title string, content fyne.CanvasObject, buttons ...*widget.Button) fyne.CanvasObject {
		c := container.NewHBox(iwidget.NewLabelWithSize(title, theme.SizeNameSubHeadingText))
		if len(buttons) > 0 {
			c.Add(layout.NewSpacer())
			for _, b := range buttons {
				c.Add(b)
			}
		}
		return container.NewBorder(
			c,
			nil,
			nil,
			nil,
			content,
		)
	}

	// All Characters

	overview := iwidget.NewNavPage(
		"Characters",
		theme.NewThemedResource(icons.PortraitSvg),
		makePageWithTitle("Characters", u.characterOverview),
	)

	wealth := iwidget.NewNavPage(
		"Wealth",
		theme.NewThemedResource(icons.GoldSvg),
		makePageWithTitle("Wealth", u.wealthOverview),
	)

	u.wealthOverview.OnUpdate = func(wallet, assets float64) {
		characterNav.SetItemBadge(wealth, ihumanize.Number(wallet+assets, 1))
	}

	allAssets := iwidget.NewNavPage(
		"Assets",
		theme.NewThemedResource(icons.Inventory2Svg),
		makePageWithTitle("Assets", u.allAssetSearch),
	)
	collectiveNav := iwidget.NewNavDrawer("All Characters",
		overview,
		allAssets,
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
			"Training",
			theme.NewThemedResource(icons.SchoolSvg),
			makePageWithTitle("Training", u.trainingOverview),
		),
		wealth,
	)
	collectiveNav.OnSelectItem = func(it *iwidget.NavItem) {
		if it == allAssets {
			u.allAssetSearch.Focus()
		}
	}

	mainContent := container.NewBorder(
		NewToolbar(u),
		statusBar,
		nil,
		nil,
		container.NewAppTabs(
			container.NewTabItemWithIcon("All Characters", theme.NewThemedResource(icons.GroupSvg), collectiveNav),
			container.NewTabItemWithIcon("Current Character", theme.AccountIcon(), characterNav),
		))
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
	u.onInit = func(_ *app.Character) {
		go u.UpdateMailIndicator()
		u.enableShortcuts()
	}
	u.onUpdateCharacter = func(c *app.Character) {
		go func() {
			characterPages := []*iwidget.NavItem{
				characterAssets,
				character,
				contracts,
				communications,
				colonies,
				mail,
				skills,
				wallet,
			}
			if !u.HasCharacter() {
				for _, it := range characterPages {
					characterNav.DisableItem(it)
				}
				characterNav.Select(overview)
				return
			}
			for _, it := range characterPages {
				characterNav.EnableItem(it)
			}
		}()
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
		go statusBar.StartUpdateTicker()

	}
	u.onAppStopped = func() {
		u.saveAppState()
	}
	u.onUpdateStatus = func() {
		go statusBar.Update()
		go pageBars.Update()
	}
	return u
}

func (u *DesktopUI) saveAppState() {
	if u.MainWindow() == nil || u.App() == nil {
		slog.Warn("Failed to save app state")
	}
	u.Settings().SetWindowSize(u.MainWindow().Canvas().Size())
	slog.Debug("Saved app state")
}

func (u *DesktopUI) ResetDesktopSettings() {
	u.Settings().ResetTabsMainID()
	u.Settings().ResetWindowSize()
	u.Settings().ResetSysTrayEnabled()
}

func (u *DesktopUI) ShowSettingsWindow() {
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

func (u *DesktopUI) showSendMailWindow(c *app.Character, mode app.SendMailMode, mail *app.CharacterMail) {
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

func (u *DesktopUI) showManageCharactersWindow() {
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

func (u *DesktopUI) PerformSearch(s string) {
	u.gameSearch.ResetOptions()
	u.gameSearch.ToogleOptions(false)
	u.gameSearch.DoSearch(s)
	u.showSearchWindow()
}

func (u *DesktopUI) showAdvancedSearch() {
	u.gameSearch.ToogleOptions(true)
	u.showSearchWindow()
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
	w.SetContent(u.gameSearch)
	w.Show()
	u.gameSearch.SetWindow(w)
	u.gameSearch.Focus()
}

func (u *DesktopUI) defineShortcuts() {
	u.shortcuts = map[string]shortcutDef{
		"snackbar": {
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
			}},
		"currentCharacter": {
			&desktop.CustomShortcut{
				KeyName:  fyne.KeyC,
				Modifier: fyne.KeyModifierAlt + fyne.KeyModifierShift,
			},
			func(fyne.Shortcut) {
				characterID := u.CurrentCharacterID()
				if characterID == 0 {
					u.ShowSnackbar("ERROR: No character selected")
					return
				}
				u.ShowInfoWindow(app.EveEntityCharacter, characterID)
			}},
		"currentLocation": {
			&desktop.CustomShortcut{
				KeyName:  fyne.KeyL,
				Modifier: fyne.KeyModifierAlt + fyne.KeyModifierShift,
			},
			func(fyne.Shortcut) {
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
			}},
		"currentShip": {
			&desktop.CustomShortcut{
				KeyName:  fyne.KeyS,
				Modifier: fyne.KeyModifierAlt + fyne.KeyModifierShift,
			},
			func(fyne.Shortcut) {
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
			}},
		"search": {
			&desktop.CustomShortcut{
				KeyName:  fyne.KeyS,
				Modifier: fyne.KeyModifierAlt,
			},
			func(fyne.Shortcut) {
				u.showSearchWindow()
			}},
		"settings": {
			&desktop.CustomShortcut{
				KeyName:  fyne.KeyComma,
				Modifier: fyne.KeyModifierControl,
			},
			func(fyne.Shortcut) {
				u.ShowSettingsWindow()
			}},
		"manageCharacters": {
			&desktop.CustomShortcut{
				KeyName:  fyne.KeyC,
				Modifier: fyne.KeyModifierAlt,
			},
			func(fyne.Shortcut) {
				u.showManageCharacters()
			}},
		"updateStatus": {
			&desktop.CustomShortcut{
				KeyName:  fyne.KeyU,
				Modifier: fyne.KeyModifierAlt,
			},
			func(fyne.Shortcut) {
				u.showUpdateStatusWindow()
			}},
	}
}

// enableShortcuts enables all registered menu shortcuts.
func (u *DesktopUI) enableShortcuts() {
	for _, sc := range u.shortcuts {
		u.MainWindow().Canvas().AddShortcut(sc.shortcut, sc.handler)
	}
}

// disableShortcuts disabled all registered menu shortcuts.
func (u *DesktopUI) disableShortcuts() {
	for _, sc := range u.shortcuts {
		u.MainWindow().Canvas().RemoveShortcut(sc.shortcut)
	}
}

func (u *DesktopUI) ShowAboutDialog() {
	d := dialog.NewCustom("About", "Close", u.makeAboutPage(), u.MainWindow())
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
	for n, p := range u.dataPaths {
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
