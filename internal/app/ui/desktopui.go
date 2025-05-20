package ui

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
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
	"github.com/ErikKalkoken/evebuddy/internal/fynetools"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	kwidget "github.com/ErikKalkoken/fyne-kx/widget"
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
	*baseUI

	accountWindow  fyne.Window
	searchWindow   fyne.Window
	settingsWindow fyne.Window

	shortcuts map[string]shortcutDef
	sfg       *singleflight.Group
}

// NewDesktopUI build the UI and returns it.
func NewDesktopUI(bu *baseUI) *DesktopUI {
	u := &DesktopUI{
		sfg:    new(singleflight.Group),
		baseUI: bu,
	}
	deskApp, ok := u.App().(desktop.App)
	if !ok {
		panic("Could not start in desktop mode")
	}

	u.showMailIndicator = func() {
		fyne.Do(func() {
			deskApp.SetSystemTrayIcon(icons.IconmarkedPng)
		})
	}
	u.hideMailIndicator = func() {
		fyne.Do(func() {
			deskApp.SetSystemTrayIcon(icons.IconPng)
		})
	}
	u.enableMenuShortcuts = u.enableShortcuts
	u.disableMenuShortcuts = u.disableShortcuts

	u.showManageCharacters = u.showManageCharactersWindow

	u.defineShortcuts()
	pageBars := newPageBarCollection(u)

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
	u.characterMail.onUpdate = func(count int) {
		fyne.Do(func() {
			characterNav.SetItemBadge(mail, formatBadge(count, 99))
		})
	}
	u.characterMail.onSendMessage = u.showSendMailWindow

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

	// Home
	var homeNav *iwidget.NavDrawer
	overview := iwidget.NewNavPage(
		"Characters",
		theme.NewThemedResource(icons.PortraitSvg),
		makePageWithTitle("Characters", u.characters),
	)

	wealth := iwidget.NewNavPage(
		"Wealth",
		theme.NewThemedResource(icons.GoldSvg),
		makePageWithTitle("Wealth", u.wealth),
	)

	allAssets := iwidget.NewNavPage(
		"Assets",
		theme.NewThemedResource(icons.Inventory2Svg),
		makePageWithTitle("Assets", u.assets),
	)

	contracts := iwidget.NewNavPage(
		"Contracts",
		theme.NewThemedResource(icons.FileSignSvg),
		makePageWithTitle("Contracts", u.contracts),
	)
	u.contracts.OnUpdate = func(count int) {
		var s string
		if count > 0 {
			s += ihumanize.Comma(count)
		}
		fyne.Do(func() {
			homeNav.SetItemBadge(contracts, s)
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
			homeNav.SetItemBadge(overviewColonies, s)
		})
	}

	industry := iwidget.NewNavPage(
		"Industry",
		theme.NewThemedResource(icons.FactorySvg),
		makePageWithTitle("Industry", u.industryJobs),
	)
	u.industryJobs.OnUpdate = func(count int) {
		var badge string
		if count > 0 {
			badge = ihumanize.Comma(count)
		}
		fyne.Do(func() {
			homeNav.SetItemBadge(industry, badge)
		})
	}
	homeNav = iwidget.NewNavDrawer("Home",
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
			makePageWithTitle("Locations", u.locations),
		),
		iwidget.NewNavPage(
			"Training",
			theme.NewThemedResource(icons.SchoolSvg),
			makePageWithTitle("Training", u.training),
		),
		wealth,
	)
	homeNav.OnSelectItem = func(it *iwidget.NavItem) {
		if it == allAssets {
			u.assets.focus()
		}
	}
	homeNav.MinWidth = minNavCharacterWidth

	statusBar := newStatusBar(u)
	toolbar := newToolbar(u)
	characterTab := container.NewTabItemWithIcon("Character", theme.AccountIcon(), characterNav)
	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon("Home", theme.NewThemedResource(theme.HomeIcon()), homeNav),
		characterTab,
	)
	mainContent := container.NewBorder(
		toolbar,
		statusBar,
		nil,
		nil,
		tabs,
	)
	u.MainWindow().SetContent(mainContent)

	u.onSetCharacter = func(id int32) {
		name := u.scs.CharacterName(id)
		fyne.Do(func() {
			characterNav.SetTitle(name)
			characterTab.Text = name
			tabs.Refresh()
		})
	}

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
	u.hideMailIndicator() // init system tray icon
	u.onUpdateCharacter = func(c *app.Character) {
		go func() {
			if !u.hasCharacter() {
				fyne.Do(func() {
					characterNav.Disable()
					homeNav.Disable()
					toolbar.ToogleSearchBar(false)
				})
				return
			}
			fyne.Do(func() {
				characterNav.Enable()
				homeNav.Enable()
				toolbar.ToogleSearchBar(true)
			})
		}()
	}
	u.onShowAndRun = func() {
		u.MainWindow().Resize(u.settings.WindowSize())
	}
	u.onAppFirstStarted = func() {
		u.enableShortcuts()
		go u.updateMailIndicator()
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
	page := newCharacterSendMail(u.baseUI, c, mode, mail)
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
	u.gameSearch.resetOptions()
	u.gameSearch.toogleOptions(false)
	u.gameSearch.doSearch(s)
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
	u.gameSearch.setWindow(w)
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
				characterID := u.currentCharacterID()
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

type pageBar struct {
	widget.BaseWidget

	buttons []*widget.Button
	icon    *kwidget.TappableImage
	title   *iwidget.Label
	u       *baseUI
}

func newPageBar(title string, icon fyne.Resource, u *baseUI, buttons ...*widget.Button) *pageBar {
	i := kwidget.NewTappableImageWithMenu(icon, fyne.NewMenu(""))
	i.SetFillMode(canvas.ImageFillContain)
	i.SetMinSize(fyne.NewSquareSize(app.IconUnitSize))
	w := &pageBar{
		buttons: buttons,
		icon:    i,
		title:   iwidget.NewLabelWithSize(title, theme.SizeNameSubHeadingText),
		u:       u,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *pageBar) SetIcon(r fyne.Resource) {
	w.icon.SetResource(r)
}

func (w *pageBar) SetMenu(items []*fyne.MenuItem) {
	w.icon.SetMenuItems(items)
}

func (w *pageBar) CreateRenderer() fyne.WidgetRenderer {
	box := container.NewHBox(w.title, layout.NewSpacer())
	if len(w.buttons) > 0 {
		for _, b := range w.buttons {
			box.Add(container.NewCenter(b))
		}
	}
	box.Add(container.NewCenter(w.icon))
	return widget.NewSimpleRenderer(box)
}

type pageBarCollection struct {
	bars         []*pageBar
	fallbackIcon fyne.Resource
	u            *DesktopUI
}

func newPageBarCollection(u *DesktopUI) *pageBarCollection {
	fallback := icons.Characterplaceholder64Jpeg
	icon, err := fynetools.MakeAvatar(fallback)
	if err != nil {
		slog.Error("failed to make avatar", "error", err)
		icon = fallback
	}
	c := &pageBarCollection{
		bars:         make([]*pageBar, 0),
		fallbackIcon: icon,
		u:            u,
	}
	return c
}

func (c *pageBarCollection) NewPageBar(title string, buttons ...*widget.Button) *pageBar {
	pb := newPageBar(title, c.fallbackIcon, c.u.baseUI, buttons...)
	c.bars = append(c.bars, pb)
	return pb
}

func (c *pageBarCollection) update() {
	if !c.u.hasCharacter() {
		for _, pb := range c.bars {
			fyne.Do(func() {
				pb.SetIcon(c.fallbackIcon)
			})
		}
		return
	}
	go c.u.updateAvatar(c.u.currentCharacterID(), func(r fyne.Resource) {
		for _, pb := range c.bars {
			fyne.Do(func() {
				pb.SetIcon(r)
			})
		}
	})
	items := c.u.makeCharacterSwitchMenu(func() {
		for _, pb := range c.bars {
			fyne.Do(func() {
				pb.Refresh()
			})
		}
	})
	for _, pb := range c.bars {
		fyne.Do(func() {
			pb.SetMenu(items)
		})
	}
}
