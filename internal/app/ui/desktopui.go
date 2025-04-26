package ui

import (
	"fmt"
	"log/slog"
	"path/filepath"
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
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

const (
	minNavCharacterWidth = 250
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

// NewDesktopUI build the UI and returns it.
func NewDesktopUI(bu *BaseUI) *DesktopUI {
	u := &DesktopUI{
		sfg:    new(singleflight.Group),
		BaseUI: bu,
	}
	deskApp, ok := u.App().(desktop.App)
	if !ok {
		panic("Could not start in desktop mode")
	}

	u.ShowMailIndicator = func() {
		fyne.Do(func() {
			deskApp.SetSystemTrayIcon(icons.IconmarkedPng)
		})
	}
	u.HideMailIndicator = func() {
		fyne.Do(func() {
			deskApp.SetSystemTrayIcon(icons.IconPng)
		})
	}
	u.EnableMenuShortcuts = u.enableShortcuts
	u.DisableMenuShortcuts = u.disableShortcuts

	u.showManageCharacters = u.showManageCharactersWindow

	u.defineShortcuts()
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

	mail := iwidget.NewNavPage(
		"Mail",
		theme.MailComposeIcon(),
		makePageWithPageBar("Mail", u.characterMail),
	)
	u.characterMail.OnUpdate = func(count int) {
		fyne.Do(func() {
			characterNav.SetItemBadge(mail, formatBadge(count, 99))
		})
	}
	u.characterMail.OnSendMessage = u.showSendMailWindow

	communications := iwidget.NewNavPage(
		"Communications",
		theme.NewThemedResource(icons.MessageSvg),
		makePageWithPageBar("Communications", u.characterCommunications),
	)
	u.characterCommunications.OnUpdate = func(count optional.Optional[int]) {
		var s string
		if count.IsEmpty() {
			s = "?"
		} else if count.ValueOrZero() > 0 {
			s = formatBadge(count.ValueOrZero(), 999)
		}
		fyne.Do(func() {
			characterNav.SetItemBadge(communications, s)
		})
	}

	skills := iwidget.NewNavPage(
		"Skills",
		theme.NewThemedResource(icons.SchoolSvg),
		makePageWithPageBar(
			"Skills",
			container.NewAppTabs(
				container.NewTabItem("Training Queue", u.characterSkillQueue),
				container.NewTabItem("Skill Catalogue", u.characterSkillCatalogue),
				container.NewTabItem("Ships", u.characterShips),
			),
		),
	)

	u.characterSkillQueue.OnUpdate = func(status, _ string) {
		fyne.Do(func() {
			characterNav.SetItemBadge(skills, status)
		})
	}

	wallet := iwidget.NewNavPage("Wallet",
		theme.NewThemedResource(icons.AttachmoneySvg),
		makePageWithPageBar("Wallet", container.NewAppTabs(
			container.NewTabItem("Transactions", u.characterWalletJournal),
			container.NewTabItem("Market Transactions", u.characterWalletTransaction),
		)))

	characterNav = iwidget.NewNavDrawer("Current Character",
		iwidget.NewNavPage(
			"Character Sheet",
			theme.NewThemedResource(icons.PortraitSvg),
			makePageWithPageBar("Character Sheet", container.NewAppTabs(
				container.NewTabItem("Character", u.characterSheet),
				container.NewTabItem("Augmentations", u.characterImplants),
				container.NewTabItem("Jump Clones", u.characterJumpClones),
				container.NewTabItem("Attributes", u.characterAttributes),
				container.NewTabItem("Biography", u.characterBiography),
			))),
		iwidget.NewNavPage(
			"Assets",
			theme.NewThemedResource(icons.Inventory2Svg),
			makePageWithPageBar("Assets", u.characterAsset),
		),
		communications,
		mail,
		skills,
		wallet,
	)
	characterNav.MinWidth = minNavCharacterWidth

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
	var collectiveNav *iwidget.NavDrawer
	overview := iwidget.NewNavPage(
		"Characters",
		theme.NewThemedResource(icons.PortraitSvg),
		makePageWithTitle("Characters", u.overviewCharacters),
	)

	wealth := iwidget.NewNavPage(
		"Wealth",
		theme.NewThemedResource(icons.GoldSvg),
		makePageWithTitle("Wealth", u.overviewWealth),
	)

	allAssets := iwidget.NewNavPage(
		"Assets",
		theme.NewThemedResource(icons.Inventory2Svg),
		makePageWithTitle("Assets", u.overviewAssets),
	)

	contractActive := container.NewTabItem("Active", u.contractsActive)
	contractTabs := container.NewAppTabs(contractActive, container.NewTabItem("All", u.contractsAll))
	contracts := iwidget.NewNavPage(
		"Contracts",
		theme.NewThemedResource(icons.FileSignSvg),
		makePageWithTitle("Contracts", contractTabs),
	)
	u.contractsActive.OnUpdate = func(count int) {
		s := "Active"
		if count > 0 {
			s += fmt.Sprintf(" (%d)", count)
		}
		fyne.Do(func() {
			contractActive.Text = s
			contractTabs.Refresh()
		})
	}

	overviewColonies := iwidget.NewNavPage(
		"Colonies",
		theme.NewThemedResource(icons.EarthSvg),
		makePageWithTitle("Colonies", u.colonies),
	)
	u.colonies.OnUpdate = func(_, expired int) {
		var s string
		if expired > 0 {
			s = fmt.Sprint(expired)
		}
		fyne.Do(func() {
			collectiveNav.SetItemBadge(overviewColonies, s)
		})
	}

	industryJobsActive := container.NewTabItem("Active", u.industryJobsActive)
	industryTabs := container.NewAppTabs(
		industryJobsActive,
		container.NewTabItem("All", u.industryJobsAll),
	)
	industry := iwidget.NewNavPage(
		"Industry",
		theme.NewThemedResource(icons.FactorySvg),
		makePageWithTitle("Industry", industryTabs),
	)
	u.industryJobsActive.OnUpdate = func(count int) {
		s := "Active"
		c := ihumanize.Comma(count)
		if count > 0 {
			s += fmt.Sprintf(" (%s)", c)
		}
		fyne.Do(func() {
			industryJobsActive.Text = s
			industryTabs.Refresh()
			collectiveNav.SetItemBadge(industry, c)
		})
	}
	collectiveNav = iwidget.NewNavDrawer("All Characters",
		overview,
		allAssets,
		iwidget.NewNavPage(
			"Clones",
			theme.NewThemedResource(icons.HeadSnowflakeSvg),
			makePageWithTitle("Clones", u.overviewClones),
		),
		contracts,
		overviewColonies,
		industry,
		iwidget.NewNavPage(
			"Locations",
			theme.NewThemedResource(icons.MapMarkerSvg),
			makePageWithTitle("Locations", u.overviewLocations),
		),
		iwidget.NewNavPage(
			"Training",
			theme.NewThemedResource(icons.SchoolSvg),
			makePageWithTitle("Training", u.overviewTraining),
		),
		wealth,
	)
	collectiveNav.OnSelectItem = func(it *iwidget.NavItem) {
		if it == allAssets {
			u.overviewAssets.Focus()
		}
	}
	collectiveNav.MinWidth = minNavCharacterWidth

	statusBar := newStatusBar(u)
	toolbar := NewToolbar(u)
	mainContent := container.NewBorder(
		toolbar,
		statusBar,
		nil,
		nil,
		container.NewAppTabs(
			container.NewTabItemWithIcon("All Characters", theme.NewThemedResource(icons.GroupSvg), collectiveNav),
			container.NewTabItemWithIcon("Current Character", theme.AccountIcon(), characterNav),
		))

	u.MainWindow().SetContent(mainContent)

	// system tray menu
	if u.settings.SysTrayEnabled() {
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
		go u.updateMailIndicator()
		u.enableShortcuts()
	}
	u.onUpdateCharacter = func(c *app.Character) {
		go func() {
			if !u.hasCharacter() {
				fyne.Do(func() {
					characterNav.Disable()
					collectiveNav.Disable()
					toolbar.ToogleSearchBar(false)
				})
				return
			}
			fyne.Do(func() {
				characterNav.Enable()
				collectiveNav.Enable()
				toolbar.ToogleSearchBar(true)
			})
		}()
	}
	u.onShowAndRun = func() {
		u.MainWindow().Resize(u.settings.WindowSize())
	}
	u.onAppFirstStarted = func() {
		go statusBar.startUpdateTicker()

	}
	u.onAppStopped = func() {
		u.saveAppState()
	}
	u.onUpdateStatus = func() {
		go statusBar.update()
		go pageBars.update()
	}
	return u
}

func (u *DesktopUI) saveAppState() {
	if u.MainWindow() == nil || u.App() == nil {
		slog.Warn("Failed to save app state")
	}
	u.settings.SetWindowSize(u.MainWindow().Canvas().Size())
	slog.Debug("Saved app state")
}

func (u *DesktopUI) ResetDesktopSettings() {
	u.settings.ResetTabsMainID()
	u.settings.ResetWindowSize()
	u.settings.ResetSysTrayEnabled()
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
	page := NewCharacterSendMail(u.BaseUI, c, mode, mail)
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
	w.SetContent(u.manageCharacters)
	u.manageCharacters.SetWindow(w)
	w.Show()
	u.manageCharacters.OnSelectCharacter = func() {
		w.Hide()
	}
}

func (u *DesktopUI) PerformSearch(s string) {
	u.gameSearch.ResetOptions()
	u.gameSearch.toogleOptions(false)
	u.gameSearch.DoSearch(s)
	u.showSearchWindow()
}

func (u *DesktopUI) showAdvancedSearch() {
	u.gameSearch.toogleOptions(true)
	u.showSearchWindow()
}

func (u *DesktopUI) showSearchWindow() {
	if u.searchWindow != nil {
		u.searchWindow.Show()
		return
	}
	c := u.currentCharacter()
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
	u.gameSearch.focus()
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
				c := u.currentCharacter()
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
				c := u.currentCharacter()
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
		"quit": {
			&desktop.CustomShortcut{
				KeyName:  fyne.KeyQ,
				Modifier: fyne.KeyModifierControl,
			},
			func(fyne.Shortcut) {
				u.App().Quit()
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
		f.Append(it.name, makePathEntry(u.App().Clipboard(), it.path))
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
