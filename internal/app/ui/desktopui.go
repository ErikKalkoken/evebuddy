package ui

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"path/filepath"
	"slices"
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

	fynetooltip "github.com/dweymouth/fyne-tooltip"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

const (
	navDrawerMinWidth = 250
)

type shortcutDef struct {
	shortcut fyne.Shortcut
	handler  func(shortcut fyne.Shortcut)
}

// The DesktopUI creates the UI for desktop.
type DesktopUI struct {
	*baseUI

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

	u.showManageCharacters = func() {
		showManageCharactersWindow(u.baseUI)
	}

	u.defineShortcuts()

	formatBadge := func(v, mx int) string {
		if v == 0 {
			return ""
		}
		if v >= mx {
			return fmt.Sprintf("%d+", mx)
		}
		return fmt.Sprint(v)
	}

	// Home

	var homeNav *iwidget.NavDrawer
	overview := iwidget.NewNavPage(
		"Character Overview",
		theme.NewThemedResource(icons.PortraitSvg),
		newContentPage("Character Overview", u.characterOverview),
	)

	wealth := iwidget.NewNavPage(
		"Wealth",
		theme.NewThemedResource(icons.GoldSvg),
		newContentPage("Wealth", u.wealth),
	)
	u.wealth.onUpdate = func(wallet, assets float64) {
		fyne.Do(func() {
			x := ihumanize.Number(wallet+assets, 1)
			homeNav.SetItemBadge(wealth, x)
		})
	}

	const assetsTitle = "Character Assets"
	allAssets := iwidget.NewNavPage(
		assetsTitle,
		theme.NewThemedResource(icons.Inventory2Svg),
		newContentPage(assetsTitle, u.assets),
	)
	u.assets.onUpdate = func(total string) {
		fyne.Do(func() {
			homeNav.SetItemBadge(allAssets, total)
		})
	}

	contracts := iwidget.NewNavPage(
		"Contracts",
		theme.NewThemedResource(icons.FileSignSvg),
		newContentPage("Contracts", u.contracts),
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
		newContentPage("Colonies", u.colonies),
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
		newContentPage("Industry", container.NewAppTabs(
			container.NewTabItem("Jobs", u.industryJobs),
			container.NewTabItem("Slots", container.NewAppTabs(
				container.NewTabItem("Manufacturing", u.slotsManufacturing),
				container.NewTabItem("Science", u.slotsResearch),
				container.NewTabItem("Reactions", u.slotsReactions),
			))),
		),
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

	marketOrders := iwidget.NewNavPage(
		"Market Orders",
		theme.NewThemedResource(icons.ChartAreasplineSvg),
		newContentPage("Market Orders", container.NewAppTabs(
			container.NewTabItem("Buy", u.marketOrdersBuy),
			container.NewTabItem("Sell", u.marketOrdersSell),
		)),
	)

	homeNav = iwidget.NewNavDrawer(
		overview,
		allAssets,
		iwidget.NewNavPage(
			"Clones",
			theme.NewThemedResource(icons.HeadSnowflakeSvg),
			newContentPage("Clones", container.NewAppTabs(
				container.NewTabItem("Augmentations", u.augmentations),
				container.NewTabItem("Jump Clones", u.clones),
			)),
		),
		contracts,
		overviewColonies,
		industry,
		marketOrders,
		iwidget.NewNavPage(
			"Character Locations",
			theme.NewThemedResource(icons.MapMarkerSvg),
			newContentPage("Character Locations", u.characterLocations),
		),
		iwidget.NewNavPage(
			"Training",
			theme.NewThemedResource(icons.SchoolSvg),
			newContentPage("Training", u.training),
		),
		wealth,
	)
	homeNav.OnSelectItem = func(it *iwidget.NavItem) {
		if it == allAssets {
			u.assets.focus()
		}
	}
	homeNav.MinWidth = navDrawerMinWidth

	// current character
	var characterNav *iwidget.NavDrawer

	characterMailNav := iwidget.NewNavPage(
		"Mail",
		theme.MailComposeIcon(),
		newContentPage("Mail", u.characterMails),
	)
	u.characterMails.onUpdate = func(count int) {
		fyne.Do(func() {
			characterNav.SetItemBadge(characterMailNav, formatBadge(count, 99))
		})
	}
	u.characterMails.onSendMessage = u.showSendMailWindow

	characterCommunicationsNav := iwidget.NewNavPage(
		"Communications",
		theme.NewThemedResource(icons.MessageSvg),
		newContentPage("Communications", u.characterCommunications),
	)
	u.characterCommunications.OnUpdate = func(count optional.Optional[int]) {
		var s string
		if count.IsEmpty() {
			s = "?"
		} else if count.ValueOrZero() > 0 {
			s = formatBadge(count.ValueOrZero(), 999)
		}
		fyne.Do(func() {
			characterNav.SetItemBadge(characterCommunicationsNav, s)
		})
	}

	characterSkillsNav := iwidget.NewNavPage(
		"Skills",
		theme.NewThemedResource(icons.SchoolSvg),
		newContentPage(
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
			characterNav.SetItemBadge(characterSkillsNav, status)
		})
	}

	characterWalletNav := iwidget.NewNavPage("Wallet",
		theme.NewThemedResource(icons.CashSvg),
		newContentPage("Wallet", u.characterWallet),
	)
	characterAssetsNav := iwidget.NewNavPage(
		"Assets",
		theme.NewThemedResource(icons.Inventory2Svg),
		newContentPage("Assets", u.characterAsset),
	)
	characterNav = iwidget.NewNavDrawer(
		iwidget.NewNavPage(
			"Character Sheet",
			theme.NewThemedResource(icons.PortraitSvg),
			newContentPage("Character Sheet", container.NewAppTabs(
				container.NewTabItem("Character", u.characterSheet),
				container.NewTabItem("Corporation", u.characterCorporation),
				container.NewTabItem("Augmentations", u.characterAugmentations),
				container.NewTabItem("Jump Clones", u.characterJumpClones),
				container.NewTabItem("Attributes", u.characterAttributes),
				container.NewTabItem("Biography", u.characterBiography),
			)),
		),
		characterAssetsNav,
		characterCommunicationsNav,
		characterMailNav,
		characterSkillsNav,
		characterWalletNav,
	)
	characterNav.MinWidth = navDrawerMinWidth
	u.characterWallet.onUpdate = func(balance string) {
		fyne.Do(func() {
			characterNav.SetItemBadge(characterWalletNav, balance)
		})
	}
	u.characterAsset.OnUpdate = func(s string) {
		fyne.Do(func() {
			characterNav.SetItemBadge(characterAssetsNav, s)
		})
	}

	// Corporation
	var corporationNav *iwidget.NavDrawer
	walletsNav := iwidget.NewNavSectionLabel("Wallets")
	corpWalletItems := []*iwidget.NavItem{walletsNav}
	corporationWalletNavs := make(map[app.Division]*iwidget.NavItem)
	corporationWalletPages := make(map[app.Division]*contentPage)
	for _, d := range app.Divisions {
		name := d.DefaultWalletName()
		corporationWalletPages[d] = newContentPage(name, u.corporationWallets[d])
		corporationWalletNavs[d] = iwidget.NewNavPage(
			name,
			theme.NewThemedResource(icons.CashSvg),
			corporationWalletPages[d],
		)
		corpWalletItems = append(corpWalletItems, corporationWalletNavs[d])
	}

	corpIndustryItem := iwidget.NewNavPage(
		"Industry",
		theme.NewThemedResource(icons.FactorySvg),
		newContentPage("Industry", u.corporationIndyJobs),
	)
	u.corporationIndyJobs.OnUpdate = func(count int) {
		var badge string
		if count > 0 {
			badge = ihumanize.Comma(count)
		}
		fyne.Do(func() {
			corporationNav.SetItemBadge(corpIndustryItem, badge)
		})
	}

	corpStructuresItem := iwidget.NewNavPage(
		"Structures",
		theme.NewThemedResource(icons.OfficeBuildingSvg),
		newContentPage("Structures", u.corporationStructures),
	)
	u.corporationStructures.OnUpdate = func(count int) {
		var badge string
		if count > 0 {
			badge = ihumanize.Comma(count)
		}
		fyne.Do(func() {
			corporationNav.SetItemBadge(corpStructuresItem, badge)
		})
	}

	corpSheetItem := iwidget.NewNavPage(
		"Corporation Sheet",
		theme.NewThemedResource(icons.StarCircleOutlineSvg),
		newContentPage("Corporation Sheet", container.NewAppTabs(
			container.NewTabItem("Corporation", u.corporationSheet),
			container.NewTabItem("Members", u.corporationMember),
		)),
	)
	corporationNav = iwidget.NewNavDrawer(slices.Concat(
		[]*iwidget.NavItem{corpSheetItem, corpIndustryItem, corpStructuresItem},
		corpWalletItems,
	)...)
	corporationNav.MinWidth = navDrawerMinWidth

	for _, d := range app.Divisions {
		u.corporationWallets[d].onBalanceUpdate = func(balance string) {
			fyne.Do(func() {
				corporationNav.SetItemBadge(corporationWalletNavs[d], balance)
			})
		}
		u.corporationWallets[d].onNameUpdate = func(name string) {
			fyne.Do(func() {
				corporationNav.SetItemText(corporationWalletNavs[d], name)
				corporationWalletPages[d].SetTitle(name)
			})
		}
	}
	u.onUpdateCorporationWalletTotals = func(balance string) {
		fyne.Do(func() {
			corporationNav.Refresh()
			corporationNav.SetItemBadge(walletsNav, balance)
		})
	}

	// Make overall UI
	makeTabContent := func(header *PageHeader, content fyne.CanvasObject) fyne.CanvasObject {
		p := theme.Padding()
		return container.NewBorder(
			container.New(layout.NewCustomPaddedLayout(p, 0, 0, 0),
				container.NewVBox(header, widget.NewSeparator()),
			),
			nil,
			nil,
			nil,
			content,
		)
	}

	homeTab := container.NewTabItemWithIcon(
		"Home",
		theme.NewThemedResource(theme.HomeIcon()),
		makeTabContent(NewPageHeader("Home"), homeNav),
	)

	characterHeader := NewPageHeaderExtended("Characters")
	characterHeader.ButtonIcon = theme.NewThemedResource(icons.SwitchaccountSvg)
	characterHeader.ButtonTooltip = "Switch character"
	characterHeader.TitleTooltip = "Show character information"
	characterTab := container.NewTabItemWithIcon(
		"Characters",
		theme.AccountIcon(),
		makeTabContent(characterHeader, characterNav),
	)

	corporationHeader := NewPageHeaderExtended("Corporations")
	corporationHeader.ButtonIcon = theme.NewThemedResource(icons.SwitchaccountSvg)
	corporationHeader.ButtonTooltip = "Switch corporation"
	corporationHeader.TitleTooltip = "Show corporation information"
	corporationTab := container.NewTabItemWithIcon(
		"Corporations",
		theme.NewThemedResource(icons.StarCircleOutlineSvg),
		makeTabContent(corporationHeader, corporationNav),
	)
	tabs := container.NewAppTabs(homeTab, characterTab, corporationTab)

	statusBar := newStatusBar(u)
	toolbar := newToolbar(u)
	mainContent := container.NewBorder(
		toolbar,
		statusBar,
		nil,
		nil,
		tabs,
	)

	// initial state is disabled
	tabs.DisableItem(characterTab)
	tabs.DisableItem(corporationTab)
	homeNav.Disable()
	toolbar.ToogleSearchBar(false)

	w := u.MainWindow()
	w.SetContent(fynetooltip.AddWindowToolTipLayer(mainContent, w.Canvas()))

	u.snackbar.Bottom = statusBar.MinSize().Height

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

	u.onSetCharacter = func(c *app.Character) {
		s := fmt.Sprintf("%s (%s)", c.EveCharacter.Name, c.EveCharacter.Corporation.Name)
		fyne.Do(func() {
			characterHeader.SetTitle(s)
			characterHeader.SetTitleAction(func() {
				u.ShowInfoWindow(app.EveEntityCharacter, c.ID)
			})
		})
		go func() {
			u.updateCharacterAvatar(c.ID, func(r fyne.Resource) {
				fyne.Do(func() {
					characterHeader.SetIcon(r)
				})
			})
		}()
	}

	togglePermittedSections := func() {
		sections, err := u.rs.PermittedSections(context.Background(), u.currentCorporationID())
		if err != nil {
			slog.Error("Failed to identify permitted sections", "error", err)
			sections.Clear()
		}
		fyne.Do(func() {
			var hasDisabled bool
			if sections.Contains(app.SectionCorporationWalletBalances) {
				for _, it := range corpWalletItems {
					it.Enable()
				}
			} else {
				for _, it := range corpWalletItems {
					it.Disable()
					hasDisabled = true
				}
			}
			if sections.Contains(app.SectionCorporationIndustryJobs) {
				corpIndustryItem.Enable()
			} else {
				corpIndustryItem.Disable()
				hasDisabled = true
			}
			if sections.Contains(app.SectionCorporationStructures) {
				corpStructuresItem.Enable()
			} else {
				corpStructuresItem.Disable()
				hasDisabled = true
			}
			if hasDisabled {
				corporationNav.Select(corpSheetItem)
			}
		})
	}

	u.onSetCorporation = func(c *app.Corporation) {
		s := c.EveCorporation.Name
		if c.EveCorporation.HasAlliance() {
			s += fmt.Sprintf(" (%s)", c.EveCorporation.Alliance.Name)
		}
		fyne.Do(func() {
			corporationHeader.SetTitle(s)
			corporationHeader.SetTitleAction(func() {
				u.ShowInfoWindow(app.EveEntityCorporation, c.ID)
			})
		})
		go func() {
			u.updateCorporationAvatar(c.ID, func(r fyne.Resource) {
				fyne.Do(func() {
					corporationHeader.SetIcon(r)
				})
			})
		}()
		go togglePermittedSections()
	}

	u.onUpdateCharacter = func(character *app.Character) {
		go func() {
			if character == nil {
				fyne.Do(func() {
					tabs.DisableItem(characterTab)
					homeNav.Disable()
					toolbar.ToogleSearchBar(false)
					characterNav.SelectIndex(0)
				})
				return
			}
			fyne.Do(func() {
				tabs.EnableItem(characterTab)
				homeNav.Enable()
				toolbar.ToogleSearchBar(true)
			})
		}()
	}

	// u.onUpdateCorporation = func(c *app.Corporation) {
	// }

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
		go fyne.Do(func() {
			characterHeader.SetButtonMenu(u.makeCharacterSwitchMenu(func() {
				characterHeader.Refresh()
			}))
		})
		go fyne.Do(func() {
			corporationHeader.SetButtonMenu(u.makeCorporationSwitchMenu(func() {
				corporationHeader.Refresh()
			}))
		})
		go togglePermittedSections()
		go func() {
			cc, err := u.ListCorporationsForSelection()
			if err != nil {
				slog.Error("Failed to fetch corporations", "error", err)
				return
			}
			if len(cc) == 0 {
				fyne.Do(func() {
					corporationNav.Select(corpSheetItem)
					tabs.DisableItem(corporationTab)
				})
				return
			}
			fyne.Do(func() {
				tabs.EnableItem(corporationTab)
			})

		}()
	}
	u.onSectionUpdateStarted = func() {
		statusBar.ShowUpdating()
	}
	u.onSectionUpdateCompleted = func() {
		statusBar.HideUpdating()
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

func (u *DesktopUI) showSendMailWindow(c *app.Character, mode app.SendMailMode, mail *app.CharacterMail) {
	title := fmt.Sprintf("New message [%s]", c.EveCharacter.Name)
	w := u.App().NewWindow(u.makeWindowTitle(title))
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
	c := u.currentCharacter()
	var n string
	if c != nil {
		n = c.EveCharacter.Name
	} else {
		n = "No Character"
	}
	w, created := u.getOrCreateWindow(fmt.Sprintf("search-%s", n), fmt.Sprintf("Search New Eden [%s]", n))
	if !created {
		w.Show()
		return
	}
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
				showSettingsWindow(u.baseUI)
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
				showUpdateStatusWindow(u.baseUI)
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
	for name, path := range u.dataPaths.All() {
		f.Append(name, makePathEntry(u.App().Clipboard(), path))
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

type contentPage struct {
	widget.BaseWidget

	title   *widget.Label
	content fyne.CanvasObject
}

func newContentPage(title string, content fyne.CanvasObject) *contentPage {
	l := widget.NewLabel(title)
	l.SizeName = theme.SizeNameSubHeadingText
	w := &contentPage{
		content: content,
		title:   l,
	}
	return w
}

func (w *contentPage) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		w.title,
		nil,
		nil,
		nil,
		w.content,
	)
	return widget.NewSimpleRenderer(c)
}

func (w *contentPage) SetTitle(s string) {
	w.title.SetText(s)
}

// PageHeader is a widget for rendering the header on a page.
// Headers contain a title and can also have a leading icon and a trailing button.
type PageHeader struct {
	widget.BaseWidget

	ButtonIcon    fyne.Resource
	ButtonTooltip string
	TitleTooltip  string

	button *iwidget.ContextMenuButton
	title  *iwidget.TappableLabel
	icon   *canvas.Image
}

func NewPageHeader(title string) *PageHeader {
	return newPageHeader(title, false)
}

func NewPageHeaderExtended(title string) *PageHeader {
	return newPageHeader(title, true)
}

func newPageHeader(title string, isExtended bool) *PageHeader {
	title2 := iwidget.NewTappableLabel(title, nil)
	title2.SizeName = theme.SizeNameSubHeadingText
	icon := iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize))
	button := iwidget.NewContextMenuButtonWithIcon("", icons.BlankSvg, fyne.NewMenu(""))
	if !isExtended {
		icon.Hide()
		button.Hide()
	}
	w := &PageHeader{
		title:  title2,
		button: button,
		icon:   icon,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *PageHeader) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(w.button.MinSize())
	c := container.NewHBox(
		container.New(layout.NewCustomPaddedLayout(0, 0, p, 0), w.icon),
		w.title,
		container.New(layout.NewCustomPaddedLayout(p, p, 0, 0), container.NewStack(spacer, container.NewCenter(w.button))),
	)
	return widget.NewSimpleRenderer(c)
}

func (w *PageHeader) SetIcon(r fyne.Resource) {
	if r == nil {
		return
	}
	w.icon.Resource = r
	w.icon.Refresh()
}

func (w *PageHeader) SetButtonMenu(it []*fyne.MenuItem) {
	if w.button.Hidden {
		return // does not have a menu button button
	}
	w.button.SetMenuItems(it)
	if w.ButtonIcon != nil {
		w.button.SetIcon(w.ButtonIcon)
	}
	if w.ButtonTooltip != "" {
		w.button.SetToolTip(w.ButtonTooltip)
	}
}

func (w *PageHeader) SetTitle(s string) {
	w.title.SetText(s)
}

func (w *PageHeader) SetTitleAction(f func()) {
	w.title.OnTapped = f
	if w.TitleTooltip != "" {
		w.title.SetToolTip(w.TitleTooltip)
	}
}
