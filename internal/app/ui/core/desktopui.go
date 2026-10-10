package core

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	fynetooltip "github.com/dweymouth/fyne-tooltip"
	"github.com/icrowley/fake"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/charactermanager"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/settings"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/updatestatus"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xdesktop"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

const (
	navDrawerMinWidth = 220
)

type shortcutDef struct {
	shortcut fyne.Shortcut
	handler  func(shortcut fyne.Shortcut)
}

// The DesktopUI creates the UI for desktop.
type DesktopUI struct {
	*baseUI
}

// NewDesktopUI build the UI and returns it.
func NewDesktopUI(params UIParams) *DesktopUI {
	u := &DesktopUI{
		baseUI: newBaseUI(params),
	}
	deskApp, ok := u.app.(desktop.App)
	if !ok {
		panic("Could not start in desktop mode")
	}

	u.showMailIndicator = func() {
		deskApp.SetSystemTrayIcon(icons.IconmarkedPng)
	}
	u.hideMailIndicator = func() {
		deskApp.SetSystemTrayIcon(icons.IconPng)
	}

	u.showManageCharacters = func() {
		charactermanager.Show(u)
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

	var homeNav *xwidget.NavDrawer
	overview := xwidget.NewNavPage(
		"Characters",
		theme.NewThemedResource(icons.PortraitSvg),
		newContentPage("Characters", u.characterOverview),
	)

	corporationOverview := xwidget.NewNavPage(
		"Corporations",
		theme.NewThemedResource(icons.StarCircleOutlineSvg),
		newContentPage("Corporations", u.corporationOverview),
	)

	wealth := xwidget.NewNavPage(
		"Wealth",
		theme.NewThemedResource(icons.GoldSvg),
		newContentPage("Wealth", u.wealth),
	)

	u.wealth.OnUpdate = func(balance optional.Optional[float64]) {
		homeNav.SetItemBadge(wealth, formatISKValueShort(balance))
		homeNav.Refresh()
	}

	const assetsTitle = "Assets"
	x := xwidget.NewContextMenuButtonWithIcon(
		"",
		theme.MoreHorizontalIcon(),
		fyne.NewMenu("", u.assetSearchAll.MoreItems()...),
	)
	x.Importance = widget.LowImportance
	assetSearch := xwidget.NewNavPage(
		assetsTitle,
		theme.NewThemedResource(icons.Inventory2Svg),
		newContentPage(assetsTitle, u.assetSearchAll, x),
	)

	unifiedCommunications := xwidget.NewNavPage(
		"Communications",
		theme.NewThemedResource(icons.MessageSvg),
		newContentPage("Communications", u.unifiedCommunications),
	)
	u.unifiedCommunications.OnUpdate = func(count optional.Optional[int]) {
		var s string
		if v, ok := count.Value(); !ok {
			s = "?"
		} else if v > 0 {
			s = formatBadge(count.ValueOrZero(), 999)
		}
		homeNav.SetItemBadge(unifiedCommunications, s)
	}

	unifiedMails := xwidget.NewNavPage(
		"Mail",
		theme.MailComposeIcon(),
		newContentPage("Mail", u.unifiedMails),
	)
	u.unifiedMails.OnUpdate = func(unread, _ int) {
		homeNav.SetItemBadge(unifiedMails, formatBadge(unread, 999))
	}

	unifiedStructures := xwidget.NewNavPage(
		"Structures",
		theme.NewThemedResource(icons.OfficeBuildingSvg),
		newContentPage("Structures", u.unifiedStructures),
	)
	u.unifiedStructures.OnUpdate = func(count int) {
		var badge string
		if count > 0 {
			badge = ihumanize.Comma(count)
		}
		homeNav.SetItemBadge(unifiedStructures, badge)
	}

	contracts := xwidget.NewNavPage(
		"Contracts",
		theme.NewThemedResource(icons.FileSignSvg),
		newContentPage("Contracts", container.NewAppTabs(
			ui.NewTabItem("Contracts", u.contractList),
			ui.NewTabItem("Slots", container.NewAppTabs(
				ui.NewTabItem("Personal Contracts", u.contractSlotsPersonal),
				ui.NewTabItem("Corporation Contracts", u.contractSlotsCorporation),
			)),
		)),
	)
	u.contractList.OnUpdate = func(count int) {
		var s string
		if count > 0 {
			s += ihumanize.Comma(count)
		}
		homeNav.SetItemBadge(contracts, s)
	}

	overviewColonies := xwidget.NewNavPage(
		"Colonies",
		theme.NewThemedResource(icons.EarthSvg),
		newContentPage("Colonies", u.colonies),
	)
	u.colonies.OnUpdate = func(_, notWorking int) {
		var s string
		if notWorking > 0 {
			s = fmt.Sprint(notWorking)
		}
		homeNav.SetItemBadge(overviewColonies, s)
	}

	industry := xwidget.NewNavPage(
		"Industry",
		theme.NewThemedResource(icons.FactorySvg),
		newContentPage("Industry", container.NewAppTabs(
			ui.NewTabItem("Jobs", u.industryJobs),
			ui.NewTabItem("Slots", container.NewAppTabs(
				ui.NewTabItem("Manufacturing", u.industrySlotsManufacturing),
				ui.NewTabItem("Science", u.industrySlotsResearch),
				ui.NewTabItem("Reactions", u.industrySlotsReactions),
			))),
		),
	)
	u.industryJobs.OnUpdate = func(count int) {
		var badge string
		if count > 0 {
			badge = ihumanize.Comma(count)
		}
		homeNav.SetItemBadge(industry, badge)
	}

	marketOrders := xwidget.NewNavPage(
		"Market Orders",
		theme.NewThemedResource(icons.ChartAreasplineSvg),
		newContentPage("Market Orders", container.NewAppTabs(
			ui.NewTabItem("Buy", u.marketOrdersBuy),
			ui.NewTabItem("Sell", u.marketOrdersSell),
		)),
	)

	trainingMore := xwidget.NewIconButtonWithMenu(
		theme.MoreHorizontalIcon(),
		fyne.NewMenu("", u.training.MoreItems()...),
	)

	skills := xwidget.NewNavPage(
		"Skills",
		theme.NewThemedResource(icons.SchoolSvg),
		newContentPage("Skills", container.NewAppTabs(
			ui.NewTabItem("Training", u.training),
			ui.NewTabItem("Search", u.skillSearch),
		), trainingMore),
	)
	u.training.OnUpdate = func(expired int) {
		var badge string
		if expired > 0 {
			badge = ihumanize.Comma(expired)
		}
		homeNav.SetItemBadge(skills, badge)
	}

	homeNav = xwidget.NewNavDrawer(
		overview,
		corporationOverview,
		assetSearch,
		xwidget.NewNavPage(
			"Clones",
			theme.NewThemedResource(icons.HeadSnowflakeSvg),
			newContentPage("Clones", container.NewAppTabs(
				ui.NewTabItem("Augmentations", u.augmentations),
				ui.NewTabItem("Jump Clones", u.clones),
			)),
		),
		unifiedCommunications,
		contracts,
		overviewColonies,
		industry,
		xwidget.NewNavPage(
			"Loyalty Points",
			theme.NewThemedResource(icons.HandHeartSvg),
			newContentPage("Loyalty Points", u.loyaltyPoints),
		),
		unifiedMails,
		marketOrders,
		unifiedStructures,
		skills,
		wealth,
	)
	homeNav.OnSelectItem = func(it *xwidget.NavItem) {
		if it == assetSearch {
			u.assetSearchAll.Focus()
		}
	}
	homeNav.MinWidth = navDrawerMinWidth

	// current character
	var characterNav *xwidget.NavDrawer

	characterMailNav := xwidget.NewNavPage(
		"Mail",
		theme.MailComposeIcon(),
		newContentPage("Mail", u.characterMails),
	)
	u.characterMails.OnUpdate = func(unread, _ int) {
		characterNav.SetItemBadge(characterMailNav, formatBadge(unread, 99))
	}
	// u.characterMails.OnSendMessage = func(character *app.Character, mode app.SendMailMode, mail *app.CharacterMail) {
	// 	characters.ShowSendMailWindow(u, character, mode, mail)
	// }

	characterCommunicationsNav := xwidget.NewNavPage(
		"Communications",
		theme.NewThemedResource(icons.MessageSvg),
		newContentPage("Communications", u.characterCommunications),
	)
	u.characterCommunications.OnUpdate = func(count optional.Optional[int]) {
		var s string
		if v, ok := count.Value(); !ok {
			s = "?"
		} else if v > 0 {
			s = formatBadge(count.ValueOrZero(), 999)
		}
		characterNav.SetItemBadge(characterCommunicationsNav, s)
	}

	characterSkillsNav := xwidget.NewNavPage(
		"Skills",
		theme.NewThemedResource(icons.SchoolSvg),
		newContentPage(
			"Skills",
			container.NewAppTabs(
				ui.NewTabItem("Catalogue", u.characterSkillCatalogue),
				ui.NewTabItem("Training", u.characterSkillQueue),
				ui.NewTabItem("Ships", u.characterShips),
			),
		),
	)

	u.characterSkillQueue.OnUpdate = func(status, _ string) {
		characterNav.SetItemBadge(characterSkillsNav, status)
	}

	characterWalletNav := xwidget.NewNavPage("Wallet",
		theme.NewThemedResource(icons.CashSvg),
		newContentPage("Wallet", u.characterWallet),
	)
	characterAssetsNav := xwidget.NewNavPage(
		"Assets",
		theme.NewThemedResource(icons.Inventory2Svg),
		newContentPage("Assets", u.characterAssetBrowser),
	)
	characterNav = xwidget.NewNavDrawer(
		xwidget.NewNavPage(
			"Character",
			theme.NewThemedResource(icons.PortraitSvg),
			newContentPage("Character", container.NewAppTabs(
				ui.NewTabItem("Character", u.characterSheet),
				ui.NewTabItem("Corporation", u.characterCorporation),
				ui.NewTabItem("Augmentations", u.characterAugmentations),
				ui.NewTabItem("Jump Clones", u.characterJumpClones),
				ui.NewTabItem("Attributes", u.characterAttributes),
				ui.NewTabItem("Biography", u.characterBiography),
			)),
		),
		characterAssetsNav,
		xwidget.NewNavPage(
			"Contacts",
			theme.NewThemedResource(icons.AccountSearchSvg),
			newContentPage("Contacts", u.characterContacts),
		),
		characterCommunicationsNav,
		characterMailNav,
		characterSkillsNav,
		characterWalletNav,
	)
	characterNav.MinWidth = navDrawerMinWidth
	u.characterWallet.OnBalanceUpdate = func(balance optional.Optional[float64]) {
		characterNav.SetItemBadge(characterWalletNav, formatISKValueShort(balance))
	}

	// Corporation
	corpAssetsItem := xwidget.NewNavPage(
		"Assets",
		theme.NewThemedResource(icons.Inventory2Svg),
		newContentPage("Assets", container.NewAppTabs(
			ui.NewTabItem("Browse", u.corporationAssetBrowser),
			ui.NewTabItem("Search", u.corporationAssetSearch),
		), xwidget.NewIconButtonWithMenu(
			theme.MoreHorizontalIcon(),
			fyne.NewMenu("", u.corporationAssetSearch.MoreItems()...),
		)),
	)

	var corporationNav *xwidget.NavDrawer
	corpWalletsItem := xwidget.NewNavPage(
		"Wallets",
		theme.NewThemedResource(icons.CashSvg),
		newContentPage("Wallets", u.corporationWallets),
	)

	corpContractsItem := xwidget.NewNavPage(
		"Contracts",
		theme.NewThemedResource(icons.FileSignSvg),
		newContentPage("Contracts", u.corporationContracts),
	)
	u.corporationContracts.OnUpdate = func(count int) {
		var badge string
		if count > 0 {
			badge = ihumanize.Comma(count)
		}
		corporationNav.SetItemBadge(corpContractsItem, badge)
	}

	corpIndustryItem := xwidget.NewNavPage(
		"Industry",
		theme.NewThemedResource(icons.FactorySvg),
		newContentPage("Industry", u.corporationIndyJobs),
	)
	u.corporationIndyJobs.OnUpdate = func(count int) {
		var badge string
		if count > 0 {
			badge = ihumanize.Comma(count)
		}
		corporationNav.SetItemBadge(corpIndustryItem, badge)
	}

	corpStructuresItem := xwidget.NewNavPage(
		"Structures",
		theme.NewThemedResource(icons.OfficeBuildingSvg),
		newContentPage("Structures", u.corporationStructures),
	)
	u.corporationStructures.OnUpdate = func(count int) {
		var badge string
		if count > 0 {
			badge = ihumanize.Comma(count)
		}
		corporationNav.SetItemBadge(corpStructuresItem, badge)
	}

	corpSheetItem := xwidget.NewNavPage(
		"Corporation",
		theme.NewThemedResource(icons.StarCircleOutlineSvg),
		newContentPage("Corporation", container.NewAppTabs(
			ui.NewTabItem("Corporation", u.corporationSheet),
			ui.NewTabItem("Members", u.corporationMember),
		)),
	)

	corpWealthItem := xwidget.NewNavPage(
		"Wealth",
		theme.NewThemedResource(icons.GoldSvg),
		newContentPage("Wealth", u.corporationWealth),
	)

	corporationNav = xwidget.NewNavDrawer(
		corpSheetItem,
		corpAssetsItem,
		corpContractsItem,
		corpIndustryItem,
		corpStructuresItem,
		corpWalletsItem,
		corpWealthItem,
	)
	corporationNav.MinWidth = navDrawerMinWidth

	// Hides the total while the wallets are not permitted, since the two update independently.
	var corpWalletTotal optional.Optional[float64]
	var corpWalletEnabled bool
	refreshCorpWalletBadge := func() {
		var s string
		if corpWalletEnabled {
			s = formatISKValueShort(corpWalletTotal)
		}
		corporationNav.SetItemBadge(corpWalletsItem, s)
	}
	u.corporationWallets.OnBalanceUpdate = func(total optional.Optional[float64]) {
		corpWalletTotal = total
		refreshCorpWalletBadge()
	}

	// Make overall UI
	makeTabContent := func(header *PageHeader, content fyne.CanvasObject) fyne.CanvasObject {
		return container.NewBorder(
			container.NewVBox(container.NewHBox(header), widget.NewSeparator()),
			nil,
			nil,
			nil,
			content,
		)
	}

	homeItem := xwidget.NewNavRailItem(
		theme.HomeIcon(),
		"Home",
		makeTabContent(NewPageHeader("Home", nil), homeNav),
	)

	characterHeader := NewPageHeader("Characters", icons.Characterplaceholder64Jpeg)
	characterHeader.SetToolTip("Switch character")
	characterItem := xwidget.NewNavRailItem(
		theme.AccountIcon(),
		"Characters",
		makeTabContent(characterHeader, characterNav),
	)

	corporationHeader := NewPageHeader("Corporations", icons.Corporationplaceholder64Png)
	corporationHeader.SetToolTip("Switch corporation")
	corporationItem := xwidget.NewNavRailItem(
		icons.StarCircleOutlineSvg,
		"Corporations",
		makeTabContent(corporationHeader, corporationNav),
	)
	searchItem := xwidget.NewNavRailActionItem(theme.SearchIcon(), "Search New Eden", u.showSearchWindow)
	rail := xwidget.NewNavRail(
		[]*xwidget.NavRailItem{homeItem, characterItem, corporationItem},
		searchItem,
		xwidget.NewNavRailMenuItem(theme.SettingsIcon(), "Manage", makeMainMenu(u)),
	)

	statusBar := newStatusBar(u)
	mainContent := container.NewBorder(
		nil,
		statusBar,
		nil,
		nil,
		rail,
	)

	// initial state is disabled
	rail.DisableItem(characterItem)
	rail.DisableItem(corporationItem)
	homeNav.Disable()
	rail.DisableItem(searchItem)

	w := u.MainWindow()
	w.SetContent(fynetooltip.AddWindowToolTipLayer(mainContent, w.Canvas()))

	u.snackbar.BottomMargin = statusBar.MinSize().Height

	// Without the tray, closing the main window quits the app.
	u.MainWindow().SetOnClosed(u.signals.BeginShutdown)

	// system tray menu
	if u.settings.SysTrayEnabled() {
		name := ui.Name()
		item := fyne.NewMenuItem(name, nil)
		item.Disabled = true
		quitItem := fyne.NewMenuItem("Quit", u.quit) // replaces Fyne's default, which skips u.quit
		quitItem.IsQuit = true
		m := fyne.NewMenu(
			"MyApp",
			item,
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem(fmt.Sprintf("Open %s", name), func() {
				u.MainWindow().Show()
			}),
			fyne.NewMenuItemSeparator(),
			quitItem,
		)
		deskApp.SetSystemTrayMenu(m)
		deskApp.SetSystemTrayWindow(u.MainWindow())
		u.MainWindow().SetCloseIntercept(func() {
			u.MainWindow().Hide()
		})
	}
	u.hideMailIndicator() // init system tray icon

	u.onSetCharacter = func(c *app.Character) {
		fyne.Do(func() {
			characterHeader.SetTitle(c.EveCharacter.Name)
		})
		go func() {
			u.SetCharacterAvatarAsync(c.ID, func(r fyne.Resource) {
				fyne.Do(func() {
					characterHeader.SetIcon(r)
				})
			})
		}()
	}
	u.onShowCharacter = func() {
		fyne.Do(func() {
			rail.Select(characterItem)
		})
	}
	u.onShowCorporation = func() {
		fyne.Do(func() {
			rail.Select(corporationItem)
		})
	}

	togglePermittedSections := func() {
		sections, err := u.Corporation().PermittedSections(context.Background(), u.CurrentCorporation().IDOrZero())
		if err != nil {
			slog.Error("Failed to identify permitted sections", "error", err)
			sections.Clear()
		}
		fyne.Do(func() {
			var hasDisabled bool

			if sections.Contains(app.SectionCorporationAssets) {
				corpAssetsItem.Enable()
			} else {
				corpAssetsItem.Disable()
				hasDisabled = true
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

			if sections.Contains(app.SectionCorporationContracts) {
				corpContractsItem.Enable()
			} else {
				corpContractsItem.Disable()
				hasDisabled = true
			}

			corpWalletEnabled = sections.Contains(app.SectionCorporationWalletBalances)
			if corpWalletEnabled {
				corpWalletsItem.Enable()
			} else {
				corpWalletsItem.Disable()
				hasDisabled = true
			}
			refreshCorpWalletBadge()

			if sections.Contains(app.SectionCorporationAssets) &&
				sections.Contains(app.SectionCorporationContracts) &&
				sections.Contains(app.SectionCorporationWalletBalances) {
				corpWealthItem.Enable()
			} else {
				corpWealthItem.Disable()
				hasDisabled = true
			}

			if hasDisabled {
				corporationNav.Select(corpSheetItem)
			}
		})
	}

	u.onSetCorporation = func(c *app.Corporation) {
		fyne.Do(func() {
			corporationHeader.SetTitle(c.EveCorporation.Name)
		})
		go func() {
			u.setCorporationAvatarAsync(c.ID, func(r fyne.Resource) {
				fyne.Do(func() {
					corporationHeader.SetIcon(r)
				})
			})
		}()
		go togglePermittedSections()
	}

	u.Signals().CurrentCharacterExchanged.AddListener(func(_ context.Context, c *app.Character) {
		if c == nil {
			fyne.Do(func() {
				rail.DisableItem(characterItem)
				homeNav.Disable()
				rail.DisableItem(searchItem)
				characterNav.SelectIndex(0)
			})
			return
		}
		fyne.Do(func() {
			rail.EnableItem(characterItem)
			homeNav.Enable()
			rail.EnableItem(searchItem)
		})
	})

	u.onShowAndRun = func() {
		u.MainWindow().Resize(u.settings.WindowSize())
	}
	u.onAppFirstStarted = func() {
		xdesktop.EnableShortcuts(u.MainWindow())
		go u.UpdateMailIndicator(context.Background())
		go statusBar.start()
	}
	u.onAppStopped = func() {
		u.saveAppState()
	}
	u.onUpdateStatus = func(ctx context.Context) {
		go togglePermittedSections()
		var wg sync.WaitGroup
		wg.Go(func() {
			u.setCharacterSwitchMenu(
				ctx,
				func(items []*fyne.MenuItem) {
					characterHeader.SetMenu(items)
				},
				func() {
					characterHeader.Refresh()
				},
			)
		})
		wg.Go(func() {
			u.setCorporationSwitchMenu(
				ctx,
				func(items []*fyne.MenuItem) {
					corporationHeader.SetMenu(items)
				},
				func() {
					corporationHeader.Refresh()
				},
			)
		})
		wg.Go(func() {
			cc, err := u.ListCorporationsForSelection(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				slog.Error("Failed to fetch corporations", "error", err)
				return
			}
			if len(cc) == 0 {
				fyne.Do(func() {
					corporationNav.Select(corpSheetItem)
					rail.DisableItem(corporationItem)
				})
				return
			}
			fyne.Do(func() {
				rail.EnableItem(corporationItem)
			})
		})
		wg.Wait()
	}
	return u
}

// quit stops signals before Fyne closes its queue, so listeners can't call fyne.Do into a closed channel.
func (u *DesktopUI) quit() {
	u.signals.BeginShutdown()
	u.app.Quit()
}

func (u *DesktopUI) saveAppState() {
	if u.MainWindow() == nil || u.app == nil {
		slog.Warn("Failed to save app state")
	}
	u.settings.SetWindowSize(u.MainWindow().Canvas().Size())
	slog.Debug("Saved app state")
}

func (u *DesktopUI) showSearchWindow() {
	w, created, onClosed := u.GetOrCreateWindowWithOnClosed("new-eden-search", "Search New Eden")
	if !created {
		w.Show()
		return
	}
	sb := kxwidget.NewSnackbar(w.Canvas())
	w.SetOnClosed(func() {
		if onClosed != nil {
			onClosed()
		}
		sb.Stop()
	})
	w.Resize(fyne.Size{Width: 700, Height: 400})
	w.SetContent(u.gameSearch)
	w.Show()
	u.gameSearch.SetWindow(w, sb.Display)
	u.gameSearch.Focus()
}

func (u *DesktopUI) defineShortcuts() {
	m := map[string]shortcutDef{
		"snackbar": {
			&desktop.CustomShortcut{
				KeyName:  fyne.KeyS,
				Modifier: fyne.KeyModifierAlt + fyne.KeyModifierControl,
			},
			func(fyne.Shortcut) {
				u.DisplaySnackbar(fmt.Sprintf(
					"%s. This is a test snack bar at %s",
					fake.Paragraph(),
					time.Now().Format("15:04:05.999999999"),
				))
				u.DisplaySnackbar(fmt.Sprintf(
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
				c := u.character.Load()
				if c == nil {
					u.DisplaySnackbar("ERROR: No character selected")
					return
				}
				u.InfoViewer().Show(c.EveCharacter.ToEveEntity())
			}},
		"currentLocation": {
			&desktop.CustomShortcut{
				KeyName:  fyne.KeyL,
				Modifier: fyne.KeyModifierAlt + fyne.KeyModifierShift,
			},
			func(fyne.Shortcut) {
				c := u.character.Load()
				if c == nil {
					u.DisplaySnackbar("ERROR: No character selected")
					return
				}
				id, ok := c.LocationID.Value()
				if !ok {
					u.DisplaySnackbar("ERROR: Missing location for current character.")
					return
				}
				u.InfoViewer().ShowLocation(id)
			}},
		"currentShip": {
			&desktop.CustomShortcut{
				KeyName:  fyne.KeyS,
				Modifier: fyne.KeyModifierAlt + fyne.KeyModifierShift,
			},
			func(fyne.Shortcut) {
				c := u.character.Load()
				if c == nil {
					u.DisplaySnackbar("ERROR: No character selected")
					return
				}
				shipTypeID, ok := c.ShipTypeID.Value()
				if !ok {
					u.DisplaySnackbar("ERROR: Missing ship for current character.")
					return
				}
				u.InfoViewer().ShowType(shipTypeID, 0)
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
				settings.Show(u)
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
				updatestatus.Show(u)
			}},
		"quit": {
			&desktop.CustomShortcut{
				KeyName:  fyne.KeyQ,
				Modifier: fyne.KeyModifierControl,
			},
			func(fyne.Shortcut) {
				u.quit()
			}},
	}
	for name, def := range m {
		xdesktop.AddShortcut(name, xdesktop.ShortcutWithHandler{
			Shortcut: def.shortcut,
			Handler:  def.handler,
		}, u.MainWindow())
	}
}

func (u *DesktopUI) showAboutDialog() {
	d := dialog.NewCustom("About", "Close", makeAboutPage(u.baseUI), u.MainWindow())
	xdesktop.DisableShortcutsForDialog(d, u.MainWindow())
	d.Show()
}

func (u *DesktopUI) showUserDataDialog() {
	f := widget.NewForm()
	for name, path := range u.dataPaths.All() {
		p := filepath.Clean(path)
		c := container.NewHBox(
			widget.NewLabel(p),
			layout.NewSpacer(),
			widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
				u.app.Clipboard().SetContent(p)
			}),
		)
		f.Append(name, c)
	}
	d := dialog.NewCustom("User data", "Close", f, u.MainWindow())
	xdesktop.DisableShortcutsForDialog(d, u.MainWindow())
	d.Show()
}

func formatISKValueShort(value optional.Optional[float64]) string {
	v, ok := value.Value()
	if !ok {
		return "?"
	}
	return ihumanize.NumberF(v, 1)
}

// contentPage is a widget that is used produce a consistent appearance for each page.
// It always has a title.
// It can optionally have trailing items, i.e. icon buttons.
type contentPage struct {
	widget.BaseWidget

	content  fyne.CanvasObject
	title    *widget.Label
	trailing []fyne.CanvasObject
}

func newContentPage(title string, content fyne.CanvasObject, trailing ...fyne.CanvasObject) *contentPage {
	l := widget.NewLabel(title)
	l.SizeName = theme.SizeNameSubHeadingText
	w := &contentPage{
		content:  content,
		title:    l,
		trailing: trailing,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *contentPage) CreateRenderer() fyne.WidgetRenderer {
	top := container.NewHBox(w.title)
	if len(w.trailing) > 0 {
		top.Add(layout.NewSpacer())
		for _, x := range w.trailing {
			top.Add(x)
		}
	}
	c := container.NewBorder(
		top,
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
