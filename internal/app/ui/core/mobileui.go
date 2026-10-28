package core

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/mobile"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"
	fynetooltip "github.com/dweymouth/fyne-tooltip"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/charactermanager"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/settings"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/updatestatus"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/ximage"
	"github.com/ErikKalkoken/evebuddy/internal/xsync"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// Destinations of the mobile navigation bar.
const (
	homeDestination = iota
	characterDestination
	corporationDestination
	searchDestination
	moreDestination
)

// MobileUI creates the UI for mobile.
type MobileUI struct {
	*baseUI

	navBar *xwidget.NavBar

	homeNav        *xwidget.Navigator
	characterNav   *xwidget.Navigator
	corporationNav *xwidget.Navigator
	searchNav      *xwidget.Navigator
	moreNav        *xwidget.Navigator

	characterPage       *xwidget.AppBar
	characterSelector   *xwidget.ContextMenuButton
	corporationPage     *xwidget.AppBar
	corporationSelector *xwidget.ContextMenuButton

	characterMailReadingPage *xwidget.AppBar
	unifiedMailReadingPage   *xwidget.AppBar
	homeUpdateStatusItem     *StatusBarItem

	homeCharactersItem     *xwidget.NavListItem
	homeColoniesItem       *xwidget.NavListItem
	homeCommunicationsItem *xwidget.NavListItem
	homeContractsItem      *xwidget.NavListItem
	homeCorporationsItem   *xwidget.NavListItem
	homeIndustryItem       *xwidget.NavListItem
	homeMailItem           *xwidget.NavListItem
	homeSkillsItem         *xwidget.NavListItem
	homeStructuresItem     *xwidget.NavListItem
	homeWealthItem         *xwidget.NavListItem

	characterCommunicationsItem *xwidget.NavListItem
	characterMailItem           *xwidget.NavListItem
	characterSkillsItem         *xwidget.NavListItem
	characterWalletItem         *xwidget.NavListItem

	corporationAssetBrowserItem *xwidget.NavListItem
	corporationAssetSearchItem  *xwidget.NavListItem
	corporationContractsItem    *xwidget.NavListItem
	corporationIndustryItem     *xwidget.NavListItem
	corporationSheetItem        *xwidget.NavListItem
	corporationStructuresItem   *xwidget.NavListItem
	corporationWalletsItem      *xwidget.NavListItem
	corporationWealthItem       *xwidget.NavListItem

	moreAboutItem            *xwidget.NavListItem
	moreList                 *xwidget.NavList
	moreManageCharactersItem *xwidget.NavListItem
	moreUpdateStatusItem     *xwidget.NavListItem

	// Hide the totals while not permitted, since totals and permissions update independently.
	corporationWalletEnabled bool
	corporationWalletTotal   optional.Optional[float64]
	corporationWealthEnabled bool
	corporationWealthTotal   optional.Optional[float64]

	hasScopeError  atomic.Bool
	hasUpdate      atomic.Bool
	hasUpdateError atomic.Bool
}

// NewMobileUI builds the UI and returns it.
func NewMobileUI(params UIParams) *MobileUI {
	u := &MobileUI{baseUI: newBaseUI(params)}

	// character destination
	fallbackAvatar, _ := ximage.MakeAvatar(icons.Characterplaceholder64Jpeg)
	u.characterSelector = xwidget.NewIconButtonWithMenu(fallbackAvatar, fyne.NewMenu(""))
	newCharacterAppBar := func(title string, body fyne.CanvasObject, items ...fyne.CanvasObject) *xwidget.AppBar {
		items = append(items, u.characterSelector)
		return xwidget.NewAppBar(title, body, items...)
	}

	const assetsTitle = "Asset Browser"
	characterAssetBrowserItem := xwidget.NewNavListItem(
		assetsTitle,
		theme.NewThemedResource(icons.Inventory2Svg),
		func() {
			u.characterAssetBrowser.Navigation.OnSelected = func() {
				u.characterNav.PushAndHideNavBar(newCharacterAppBar(assetsTitle, u.characterAssetBrowser.Selected))
			}
			u.characterNav.Push(newCharacterAppBar(assetsTitle, container.NewHScroll(u.characterAssetBrowser.Navigation)))
		},
	)

	characterCommunicationsMenu := fyne.NewMenu("")
	u.characterCommunicationsItem = xwidget.NewNavListItem(
		"Communications",
		theme.NewThemedResource(icons.MessageSvg),
		func() {
			u.characterCommunications.MessagePane.OnSelected = func() {
				u.characterNav.PushAndHideNavBar(
					newCharacterAppBar("Communications", u.characterCommunications.ReadingPane),
				)
			}
			u.characterNav.Push(
				newCharacterAppBar(
					"Communications",
					u.characterCommunications.MessagePane,
					xwidget.NewIconButtonWithMenu(theme.FolderIcon(), characterCommunicationsMenu),
				),
			)
		},
	)

	mailMenu := fyne.NewMenu("")
	u.characterMailItem = xwidget.NewNavListItem(
		"Mail",
		theme.MailComposeIcon(),
		func() {
			u.characterMails.MessagePane.OnSelected = func() {
				u.characterMailReadingPage = newCharacterAppBar(
					"Mail",
					u.characterMails.ReadingPane,
					xwidget.NewIconButton(u.characterMails.ReadingPane.MakeReplyAction()),
					xwidget.NewIconButton(u.characterMails.ReadingPane.MakeReplyAllAction()),
					xwidget.NewIconButton(u.characterMails.ReadingPane.MakeForwardAction()),
					xwidget.NewIconButton(u.characterMails.ReadingPane.MakeDeleteAction()),
				)
				u.characterNav.PushAndHideNavBar(u.characterMailReadingPage)
			}
			u.characterNav.Push(newCharacterAppBar(
				"Mail",
				u.characterMails.MessagePane,
				xwidget.NewIconButtonWithMenu(theme.FolderIcon(), mailMenu),
				xwidget.NewIconButton(u.characterMails.MakeComposeMessageAction()),
			))
		},
	)

	u.characterSkillsItem = xwidget.NewNavListItem(
		"Skills",
		theme.NewThemedResource(icons.SchoolSvg),
		func() {
			u.characterNav.Push(
				newCharacterAppBar(
					"Skills",
					container.NewAppTabs(
						ui.NewTabItem("Catalogue", u.characterSkillCatalogue),
						ui.NewTabItem("Training", u.characterSkillQueue),
						ui.NewTabItem("Ships", u.characterShips),
					),
				))
		},
	)

	u.characterWalletItem = xwidget.NewNavListItem(
		"Wallet",
		theme.NewThemedResource(icons.AttachmoneySvg),
		func() {
			u.characterNav.Push(
				newCharacterAppBar("Wallet", u.characterWallet))
		},
	)

	characterList := xwidget.NewNavList(
		xwidget.NewNavListItem(
			"Character",
			theme.NewThemedResource(icons.PortraitSvg),
			func() {
				u.characterNav.Push(
					newCharacterAppBar(
						"Character",
						container.NewAppTabs(
							ui.NewTabItem("Character", u.characterSheet),
							ui.NewTabItem("Corporation", u.characterCorporation),
							ui.NewTabItem("Augmentations", u.characterAugmentations),
							ui.NewTabItem("Clones", u.characterJumpClones),
							ui.NewTabItem("Attributes", u.characterAttributes),
							ui.NewTabItem("Bio", u.characterBiography),
						),
					))
			},
		),
		characterAssetBrowserItem,
		xwidget.NewNavListItem(
			"Contacts",
			theme.NewThemedResource(icons.AccountSearchSvg),
			func() {
				u.characterNav.Push(
					newCharacterAppBar("Contacts", u.characterContacts))
			},
		),
		u.characterCommunicationsItem,
		u.characterMailItem,
		u.characterSkillsItem,
		u.characterWalletItem,
	)

	u.characterMails.OnUpdate = func(unread optional.Optional[int], missing int) {
		var s []string
		if x := formatMailUnread(unread); x != "" {
			s = append(s, x)
		}
		if missing > 0 {
			s = append(s, fmt.Sprintf("%d%% downloaded", 100-missing))
		}
		u.characterMailItem.Supporting = strings.Join(s, " • ")
		u.characterMailItem.Refresh()

		mailMenu.Items = u.characterMails.NavigationPane.MakeFolderMenu()
		mailMenu.Refresh()
	}
	u.characterMails.ReadingPane.OnCleared = func() {
		if u.characterMailReadingPage != nil && u.characterNav.Current() == u.characterMailReadingPage {
			u.characterNav.Pop()
		}
	}

	u.characterCommunications.OnUpdate = func(count optional.Optional[int]) {
		u.characterCommunicationsItem.Supporting = formatCount(count, formatUnread)
		u.characterCommunicationsItem.Refresh()

		characterCommunicationsMenu.Items = u.characterCommunications.NavigationPane.MakeFolderMenu()
		characterCommunicationsMenu.Refresh()
	}

	u.characterSkillQueue.OnUpdate = func(_, status string) {
		u.characterSkillsItem.Supporting = status
		u.characterSkillsItem.Refresh()
	}

	u.characterWallet.OnBalanceUpdate = func(balance optional.Optional[float64]) {
		u.characterWalletItem.Supporting = formatISKValue(balance, ui.FloatFormatISKRounded)
		u.characterWalletItem.Refresh()
	}

	u.characterPage = newCharacterAppBar("Characters", characterList)
	u.characterNav = xwidget.NewNavigator(u.characterPage)

	// corporation destination
	fallbackAvatar2, _ := ximage.MakeAvatar(icons.Corporationplaceholder64Png)
	u.corporationSelector = xwidget.NewIconButtonWithMenu(fallbackAvatar2, fyne.NewMenu(""))
	newCorpAppBar := func(title string, body fyne.CanvasObject, items ...fyne.CanvasObject) *xwidget.AppBar {
		items = append(items, u.corporationSelector)
		return xwidget.NewAppBar(title, body, items...)
	}

	const corpAssetBrowserTitle = "Asset Browser"
	u.corporationAssetBrowserItem = xwidget.NewNavListItem(
		corpAssetBrowserTitle,
		theme.NewThemedResource(icons.Inventory2Svg),
		func() {
			u.corporationAssetBrowser.Navigation.OnSelected = func() {
				u.corporationNav.PushAndHideNavBar(newCorpAppBar(corpAssetBrowserTitle, u.corporationAssetBrowser.Selected))
			}
			u.corporationNav.Push(newCorpAppBar(corpAssetBrowserTitle, container.NewHScroll(u.corporationAssetBrowser.Navigation)))
		},
	)

	const corpAssetSearchTitle = "Asset Search"
	u.corporationAssetSearchItem = xwidget.NewNavListItem(
		corpAssetSearchTitle,
		theme.NewThemedResource(icons.Inventory2Svg),
		func() {
			u.corporationNav.Push(xwidget.NewAppBar(corpAssetSearchTitle, u.corporationAssetSearch))
			u.corporationAssetSearch.Focus()
		},
	)

	u.corporationWalletsItem = xwidget.NewNavListItem(
		"Wallets",
		theme.NewThemedResource(icons.CashSvg),
		func() {
			u.corporationNav.Push(newCorpAppBar("Wallets", u.corporationWallets))
		},
	)

	u.corporationContractsItem = xwidget.NewNavListItem(
		"Contracts",
		theme.NewThemedResource(icons.FileSignSvg),
		func() {
			u.corporationNav.Push(newCorpAppBar("Contracts", u.corporationContracts))
		},
	)

	u.corporationIndustryItem = xwidget.NewNavListItem(
		"Industry",
		theme.NewThemedResource(icons.FactorySvg),
		func() {
			u.corporationNav.Push(newCorpAppBar("Industry", u.corporationIndyJobs))
		},
	)

	u.corporationStructuresItem = xwidget.NewNavListItem(
		"Structures",
		theme.NewThemedResource(icons.OfficeBuildingSvg),
		func() {
			u.corporationNav.Push(newCorpAppBar("Structures", u.corporationStructures))
		},
	)

	u.corporationSheetItem = xwidget.NewNavListItem(
		"Corporation",
		theme.NewThemedResource(icons.PortraitSvg),
		func() {
			u.corporationNav.Push(
				newCorpAppBar(
					"Corporation",
					container.NewAppTabs(
						ui.NewTabItem("Corporation", u.corporationSheet),
						ui.NewTabItem("Members", u.corporationMember),
					),
				))
		},
	)

	u.corporationWealthItem = xwidget.NewNavListItem(
		"Wealth",
		theme.NewThemedResource(icons.GoldSvg),
		func() {
			u.corporationNav.Push(newCorpAppBar("Wealth", u.corporationWealth))
		},
	)

	corpList := xwidget.NewNavList(
		slices.Concat([]*xwidget.NavListItem{
			u.corporationSheetItem,
			u.corporationAssetBrowserItem,
			u.corporationAssetSearchItem,
			u.corporationContractsItem,
			u.corporationIndustryItem,
			u.corporationStructuresItem,
			u.corporationWalletsItem,
			u.corporationWealthItem,
		})...,
	)
	u.corporationContracts.OnUpdate = func(count optional.Optional[int]) {
		u.corporationContractsItem.Supporting = formatCount(count, func(v int) string {
			return ihumanize.Plural(v, "contract", "contracts") + " outstanding"
		})
		u.corporationContractsItem.Refresh()
	}
	u.corporationIndyJobs.OnUpdate = func(count optional.Optional[int]) {
		u.corporationIndustryItem.Supporting = formatCount(count, func(v int) string {
			return ihumanize.Plural(v, "job", "jobs") + " ready"
		})
		u.corporationIndustryItem.Refresh()
	}
	u.corporationStructures.OnUpdate = func(count optional.Optional[int]) {
		u.corporationStructuresItem.Supporting = formatCount(count, func(v int) string {
			return ihumanize.Plural(v, "structure", "structures") + " reinforced"
		})
		u.corporationStructuresItem.Refresh()
	}
	u.corporationWallets.OnBalanceUpdate = func(total optional.Optional[float64]) {
		u.corporationWalletTotal = total
		u.refreshCorporationWalletSupporting()
	}
	u.corporationWealth.OnUpdate = func(total optional.Optional[float64]) {
		u.corporationWealthTotal = total
		u.refreshCorporationWealthSupporting()
	}

	u.corporationPage = newCorpAppBar("Corporations", corpList)
	u.corporationNav = xwidget.NewNavigator(u.corporationPage)

	// other

	u.makeHomeNav()

	u.searchNav = xwidget.NewNavigator(
		newCharacterAppBar("Search", u.gameSearch),
	)

	// more destination
	u.moreUpdateStatusItem = xwidget.NewNavListItem(
		"Update status",
		theme.NewThemedResource(icons.UpdateSvg),
		func() {
			updatestatus.Show(u)
		},
	)
	u.moreManageCharactersItem = xwidget.NewNavListItem(
		"Manage characters",
		theme.NewThemedResource(icons.ManageaccountsSvg),
		func() {
			charactermanager.Show(u)
		},
	)

	u.moreAboutItem = xwidget.NewNavListItem(
		"About",
		theme.InfoIcon(),
		func() {
			u.moreNav.Push(xwidget.NewAppBar("About", newAboutPage(u.baseUI)))
		},
	)
	u.moreList = xwidget.NewNavList(
		xwidget.NewNavListItem(
			"Settings",
			theme.NewThemedResource(icons.TuneVariantSvg),
			func() {
				settings.Show(u)
			},
		),
		u.moreManageCharactersItem,
		u.moreUpdateStatusItem,
		u.moreAboutItem,
	)
	u.moreNav = xwidget.NewNavigator(xwidget.NewAppBar("Manage", u.moreList))

	// navigation bar
	characterDest := xwidget.NewDestinationDef("Characters", theme.NewThemedResource(icons.AccountSvg), u.characterNav)
	characterDest.OnSelectedAgain = func() {
		u.characterNav.PopAll()
	}

	corpDest := xwidget.NewDestinationDef("Corporations", theme.NewThemedResource(icons.StarCircleOutlineSvg), u.corporationNav)
	corpDest.OnSelectedAgain = func() {
		u.corporationNav.PopAll()
	}

	homeDest := xwidget.NewDestinationDef("Home", theme.NewThemedResource(theme.HomeIcon()), u.homeNav)
	homeDest.OnSelectedAgain = func() {
		u.homeNav.PopAll()
	}

	searchDest := xwidget.NewDestinationDef("Search", theme.SearchIcon(), u.searchNav)
	searchDest.OnSelected = func() {
		u.gameSearch.Focus()
	}
	searchDest.OnSelectedAgain = func() {
		u.gameSearch.Reset()
	}

	moreDest := xwidget.NewDestinationDef("Manage", theme.SettingsIcon(), u.moreNav)
	moreDest.OnSelectedAgain = func() {
		u.moreNav.PopAll()
	}

	u.navBar = xwidget.NewNavBar(homeDest, characterDest, corpDest, searchDest, moreDest)
	u.homeNav.NavBar = u.navBar
	u.characterNav.NavBar = u.navBar
	u.corporationNav.NavBar = u.navBar
	u.searchNav.NavBar = u.navBar

	u.snackbar.BottomMargin = theme.Padding() * 17

	w := u.MainWindow()
	w.Canvas().SetOnTypedKey(u.handleTypedKey)

	// initial state
	u.navBar.Disable(homeDestination)
	u.navBar.Disable(characterDestination)
	u.navBar.Disable(corporationDestination)
	u.navBar.Disable(searchDestination)
	u.navBar.Select(moreDestination)

	u.onUpdateStatus = func(ctx context.Context) {
		u.Go(u.updatePermittedSections)
		var wg sync.WaitGroup
		wg.Go(func() {
			u.setCharacterSwitchMenu(
				ctx,
				func(items []*fyne.MenuItem) {
					u.characterSelector.SetMenuItems(items)
				},
				u.characterSelector.Refresh,
			)
		})
		wg.Go(func() {
			u.setCorporationSwitchMenu(
				ctx,
				func(items []*fyne.MenuItem) {
					u.corporationSelector.SetMenuItems(items)
				},
				u.corporationSelector.Refresh,
			)
		})
		wg.Go(func() {
			u.updateCorporationDestination(ctx)
		})
		wg.Wait()
	}

	u.Signals().CurrentCharacterExchanged.AddListener(u.handleCurrentCharacterExchanged)
	u.onSetCharacter = func(c *app.Character) {
		fyne.Do(func() {
			u.characterPage.SetTitle(c.EveCharacter.Name)
			u.characterNav.PopAll()
		})
		runAsync(func() {
			u.SetCharacterAvatarAsync(c.ID, func(r fyne.Resource) {
				fyne.Do(func() {
					u.characterSelector.SetIcon(r)
				})
			})
		})
		runAsync(func() {
			fyne.Do(func() {
				u.characterCommunications.MessagePane.ResetHeaders()
			})
		})
	}
	u.onShowCharacter = func() {
		fyne.Do(func() {
			u.navBar.Select(characterDestination)
		})
	}
	u.onShowCorporation = func() {
		fyne.Do(func() {
			u.navBar.Select(corporationDestination)
		})
	}

	u.onSetCorporation = func(c *app.Corporation) {
		fyne.Do(func() {
			u.corporationPage.SetTitle(c.EveCorporation.Name)
			u.corporationNav.PopAll()
		})
		runAsync(func() {
			u.setCorporationAvatarAsync(c.ID, func(r fyne.Resource) {
				fyne.Do(func() {
					u.corporationSelector.SetIcon(r)
				})
			})
		})
		// permitted sections are updated by the status update that follows the exchange
	}

	u.onShowAndRun = func() {
		if u.isFakeMobile {
			u.MainWindow().Resize(fyne.NewSize(340, 700))
			u.MainWindow().SetFixedSize(true)
		}
	}

	u.onAppFirstStarted = func(startupCtx context.Context) {
		// signals
		u.Signals().CharacterAdded.AddListener(func(ctx context.Context, _ *app.Character) {
			u.updateCharacterCount(ctx)
			u.updateUpdateStatus(ctx)
		})
		u.Signals().CharacterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
			u.updateCharacterCount(ctx)
			u.updateUpdateStatus(ctx)
		})
		u.Signals().CharacterSectionUpdated.AddListener(func(ctx context.Context, _ app.CharacterSectionUpdated) {
			u.updateUpdateStatus(ctx)
		})
		u.Signals().CorporationSectionUpdated.AddListener(func(ctx context.Context, _ app.CorporationSectionUpdated) {
			u.updateUpdateStatus(ctx)
		})
		u.Signals().EveUniverseSectionUpdated.AddListener(func(ctx context.Context, _ app.EveUniverseSectionUpdated) {
			u.updateUpdateStatus(ctx)
		})

		u.updateCharacterCount(startupCtx)
		u.updateUpdateStatus(startupCtx)

		if !u.isOfflineMode {
			u.tasks.Go(func(ctx context.Context) {
				xsync.RunEvery(ctx, versionTicker, u.updateVersionHint)
			})
		}
	}

	u.onUpdateMissingScope = func(characterCount int) {
		var icon fyne.Resource
		if characterCount > 0 {
			icon = theme.NewWarningThemedResource(theme.WarningIcon())
			u.hasScopeError.Store(true)
		} else {
			icon = nil
			u.hasScopeError.Store(false)
		}
		fyne.Do(func() {
			u.moreManageCharactersItem.Trailing = icon
			u.moreList.Refresh()
			u.refreshMoreBadge()
		})
	}

	w.SetContent(fynetooltip.AddWindowToolTipLayer(u.navBar, w.Canvas()))
	return u
}

// handleTypedKey navigates back in the selected destination when the back key is pressed.
func (u *MobileUI) handleTypedKey(ev *fyne.KeyEvent) {
	if ev.Name != mobile.KeyBack {
		return
	}
	id, ok := u.navBar.Selected()
	if !ok {
		return
	}
	switch id {
	case homeDestination:
		u.homeNav.Pop()
	case characterDestination:
		u.characterNav.Pop()
	case corporationDestination:
		u.corporationNav.Pop()
	case searchDestination:
		u.searchNav.Pop()
	case moreDestination:
		u.moreNav.Pop()
	}
}

func (u *MobileUI) handleCurrentCharacterExchanged(_ context.Context, c *app.Character) {
	fyne.Do(func() {
		if c == nil {
			u.navBar.Disable(homeDestination)
			u.navBar.Disable(characterDestination)
			u.navBar.Disable(searchDestination)
			u.navBar.Select(moreDestination)
		} else {
			wasDisabled := !u.navBar.Enabled(homeDestination)
			u.navBar.Enable(homeDestination)
			u.navBar.Enable(characterDestination)
			u.navBar.Enable(searchDestination)
			if wasDisabled {
				u.navBar.Select(homeDestination)
			}
		}
	})
}

// updateCorporationDestination enables the corporation destination when there are corporations.
func (u *MobileUI) updateCorporationDestination(ctx context.Context) {
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
			u.navBar.Disable(corporationDestination)
			id, ok := u.navBar.Selected()
			if ok && id == corporationDestination {
				u.navBar.Select(homeDestination)
			}
		})
		return
	}
	fyne.Do(func() {
		u.navBar.Enable(corporationDestination)
	})
}

// updatePermittedSections enables the corporation pages the current corporation permits.
func (u *MobileUI) updatePermittedSections(ctx context.Context) {
	corporationID := u.CurrentCorporation().IDOrZero()
	sections, err := u.Corporation().PermittedSections(ctx, corporationID)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		slog.Error("Failed to identify permitted sections", "error", err)
		sections.Clear()
	}
	setEnabled := func(it *xwidget.NavListItem, enabled bool) {
		if enabled {
			it.Enable()
		} else {
			it.Disable()
		}
	}
	fyne.Do(func() {
		if u.CurrentCorporation().IDOrZero() != corporationID {
			return // a newer run follows for the current corporation
		}
		setEnabled(u.corporationAssetBrowserItem, sections.Contains(app.SectionCorporationAssets))
		setEnabled(u.corporationAssetSearchItem, sections.Contains(app.SectionCorporationAssets))
		setEnabled(u.corporationIndustryItem, sections.Contains(app.SectionCorporationIndustryJobs))
		setEnabled(u.corporationStructuresItem, sections.Contains(app.SectionCorporationStructures))
		setEnabled(u.corporationContractsItem, sections.Contains(app.SectionCorporationContracts))
		u.corporationWalletEnabled = sections.Contains(app.SectionCorporationWalletBalances)
		setEnabled(u.corporationWalletsItem, u.corporationWalletEnabled)
		u.refreshCorporationWalletSupporting()
		u.corporationWealthEnabled = sections.Contains(app.SectionCorporationAssets) &&
			sections.Contains(app.SectionCorporationContracts) &&
			sections.Contains(app.SectionCorporationWalletBalances)
		setEnabled(u.corporationWealthItem, u.corporationWealthEnabled)
		u.refreshCorporationWealthSupporting()
	})
}

func (u *MobileUI) refreshCorporationWalletSupporting() {
	var s string
	if u.corporationWalletEnabled {
		s = formatISKValue(u.corporationWalletTotal, ui.FloatFormatISKRounded)
	}
	u.corporationWalletsItem.Supporting = s
	u.corporationWalletsItem.Refresh()
}

func (u *MobileUI) refreshCorporationWealthSupporting() {
	var s string
	if u.corporationWealthEnabled {
		s = formatISKValue(u.corporationWealthTotal, ui.FloatFormatISKRounded)
	}
	u.corporationWealthItem.Supporting = s
	u.corporationWealthItem.Refresh()
}

// refreshMoreBadge shows a badge on the more destination when something needs attention.
func (u *MobileUI) refreshMoreBadge() {
	if u.hasUpdateError.Load() || u.hasUpdate.Load() || u.hasScopeError.Load() || u.isOffline.Load() {
		var importance widget.Importance
		if u.hasUpdateError.Load() {
			importance = widget.DangerImportance
		} else if u.hasScopeError.Load() || u.isOffline.Load() {
			importance = widget.WarningImportance
		} else if u.hasUpdate.Load() {
			importance = widget.HighImportance
		}
		u.navBar.ShowBadge(moreDestination, importance)
	} else {
		u.navBar.HideBadge(moreDestination)
	}
}

func (u *MobileUI) updateCharacterCount(ctx context.Context) {
	ids, err := u.cs.ListCharacterIDs(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		slog.Error("updating character count", "error", err)
		return
	}
	fyne.Do(func() {
		u.moreManageCharactersItem.Supporting = ihumanize.Plural(ids.Size(), "character", "characters")
		u.moreManageCharactersItem.Refresh()
	})
}

func (u *MobileUI) updateUpdateStatus(_ context.Context) {
	set := func(full, short string, i widget.Importance, icon fyne.Resource) {
		fyne.Do(func() {
			u.refreshMoreBadge()
			u.moreUpdateStatusItem.Supporting = full
			u.moreUpdateStatusItem.SupportingImportance = i
			u.moreUpdateStatusItem.Trailing = icon
			u.moreUpdateStatusItem.Refresh()
			u.homeUpdateStatusItem.SetTextAndImportance(short, i)
		})
	}

	if u.ess.IsDailyDowntime() {
		u.isOffline.Store(true)
		set(
			fmt.Sprintf("Off during daily downtime: %s", u.ess.DailyDowntime()),
			"OFF",
			widget.WarningImportance,
			theme.NewWarningThemedResource(theme.WarningIcon()),
		)
		return
	}
	u.isOffline.Store(false)
	status := u.scs.Summary()
	var icon fyne.Resource
	if status.Errors > 0 {
		icon = theme.NewErrorThemedResource(theme.WarningIcon())
		u.hasUpdateError.Store(true)
	} else {
		u.hasUpdateError.Store(false)
	}
	set(status.Display(), status.DisplayShort(), status.Status().ToImportance(), icon)
}

// updateVersionHint shows on the about item whether a newer version is available.
func (u *MobileUI) updateVersionHint(ctx context.Context) {
	v, err := u.availableUpdate(ctx)
	if err != nil {
		if !app.IsContextCanceled(ctx, err) {
			slog.Error("fetch github version for menu info", "error", err)
		}
		return
	}
	if v.IsRemoteNewer {
		u.hasUpdate.Store(true)
		fyne.Do(func() {
			u.refreshMoreBadge()
			u.moreAboutItem.Supporting = "Update available"
			u.moreAboutItem.SupportingImportance = widget.HighImportance
			u.moreAboutItem.Trailing = theme.NewPrimaryThemedResource(icons.Numeric1CircleSvg)
			u.moreAboutItem.Refresh()
		})
	} else {
		u.hasUpdate.Store(false)
		fyne.Do(func() {
			u.refreshMoreBadge()
			u.moreAboutItem.Supporting = ""
			u.moreAboutItem.SupportingImportance = widget.MediumImportance
			u.moreAboutItem.Trailing = nil
			u.moreAboutItem.Refresh()
		})
	}
}

func (u *MobileUI) makeHomeNav() {
	u.homeColoniesItem = xwidget.NewNavListItem(
		"Colonies",
		theme.NewThemedResource(icons.EarthSvg),
		func() {
			u.homeNav.PushAndHideNavBar(xwidget.NewAppBar("Colonies", u.colonies))
		},
	)
	u.colonies.OnUpdate = func(notWorking optional.Optional[int]) {
		setNavItemSupportingWarning(u.homeColoniesItem, notWorking, "not working")
	}

	u.homeIndustryItem = xwidget.NewNavListItem(
		"Industry",
		theme.NewThemedResource(icons.FactorySvg),
		func() {
			u.homeNav.Push(xwidget.NewAppBar("Industry",
				container.NewAppTabs(
					ui.NewTabItem("Jobs", u.industryJobs),
					ui.NewTabItem("Slots", container.NewAppTabs(
						ui.NewTabItem("Manufacturing", u.industrySlotsManufacturing),
						ui.NewTabItem("Science", u.industrySlotsResearch),
						ui.NewTabItem("Reactions", u.industrySlotsReactions),
					)),
				),
			))
		},
	)
	u.industryJobs.OnUpdate = func(count optional.Optional[int]) {
		u.homeIndustryItem.Supporting = formatCount(count, func(v int) string {
			return ihumanize.Plural(v, "job", "jobs") + " ready"
		})
		u.homeIndustryItem.Refresh()
	}

	u.homeContractsItem = xwidget.NewNavListItem(
		"Contracts",
		theme.NewThemedResource(icons.FileSignSvg),
		func() {
			u.homeNav.Push(xwidget.NewAppBar("Contracts",
				container.NewAppTabs(
					ui.NewTabItem("Contracts", u.contractList),
					ui.NewTabItem("Slots", container.NewAppTabs(
						ui.NewTabItem("Personal Contracts", u.contractSlotsPersonal),
						ui.NewTabItem("Corporation Contracts", u.contractSlotsCorporation),
					)),
				),
			))
		},
	)
	u.contractList.OnUpdate = func(count optional.Optional[int]) {
		u.homeContractsItem.Supporting = formatCount(count, func(v int) string {
			return ihumanize.Plural(v, "contract", "contracts") + " outstanding"
		})
		u.homeContractsItem.Refresh()
	}

	u.homeWealthItem = xwidget.NewNavListItem(
		"Wealth",
		theme.NewThemedResource(icons.GoldSvg),
		func() {
			u.homeNav.PushAndHideNavBar(xwidget.NewAppBar("Wealth", u.wealth))
		},
	)
	u.wealth.OnUpdate = func(total optional.Optional[float64]) {
		var s string
		if v, ok := total.Value(); !ok || v != 0 {
			s = formatISKValue(total, ui.FloatFormatISKRounded)
		}
		u.homeWealthItem.Supporting = s
		u.homeWealthItem.Refresh()
	}

	homeAssetSearchItem := xwidget.NewNavListItem(
		"Assets",
		theme.NewThemedResource(icons.Inventory2Svg),
		func() {
			u.homeNav.Push(xwidget.NewAppBar("Assets", u.assetSearchAll))
			u.assetSearchAll.Focus()
		},
	)

	u.homeCharactersItem = xwidget.NewNavListItem(
		"Characters",
		theme.NewThemedResource(icons.PortraitSvg),
		func() {
			u.homeNav.Push(xwidget.NewAppBar("Characters", u.characterOverview))
		},
	)
	u.characterOverview.OnUpdate = func(characters int) {
		u.homeCharactersItem.Supporting = ihumanize.Plural(characters, "character", "characters")
		u.homeCharactersItem.Refresh()
	}

	u.homeCorporationsItem = xwidget.NewNavListItem(
		"Corporations",
		theme.NewThemedResource(icons.StarCircleOutlineSvg),
		func() {
			u.homeNav.Push(xwidget.NewAppBar("Corporations", u.corporationOverview))
		},
	)
	u.corporationOverview.OnUpdate = func(corporations int) {
		u.homeCorporationsItem.Supporting = ihumanize.Plural(corporations, "corporation", "corporations")
		u.homeCorporationsItem.Refresh()
	}

	unifiedCommunicationsMenu := fyne.NewMenu("")
	u.homeCommunicationsItem = xwidget.NewNavListItem(
		"Communications",
		theme.NewThemedResource(icons.MessageSvg),
		func() {
			u.unifiedCommunications.MessagePane.OnSelected = func() {
				u.homeNav.PushAndHideNavBar(
					xwidget.NewAppBar("Communications", u.unifiedCommunications.ReadingPane),
				)
			}
			u.homeNav.Push(
				xwidget.NewAppBar(
					"Communications",
					u.unifiedCommunications.MessagePane,
					xwidget.NewIconButtonWithMenu(theme.FolderIcon(), unifiedCommunicationsMenu),
				),
			)
		},
	)
	u.unifiedCommunications.OnUpdate = func(count optional.Optional[int]) {
		u.homeCommunicationsItem.Supporting = formatCount(count, formatUnread)
		u.homeCommunicationsItem.Refresh()

		unifiedCommunicationsMenu.Items = u.unifiedCommunications.NavigationPane.MakeFolderMenu()
		unifiedCommunicationsMenu.Refresh()
	}

	unifiedMailMenu := fyne.NewMenu("")
	u.unifiedMails.ReadingPane.OnCleared = func() {
		if u.unifiedMailReadingPage != nil && u.homeNav.Current() == u.unifiedMailReadingPage {
			u.homeNav.Pop()
		}
	}
	u.homeMailItem = xwidget.NewNavListItem(
		"Mail",
		theme.MailComposeIcon(),
		func() {
			u.unifiedMails.MessagePane.OnSelected = func() {
				u.unifiedMailReadingPage = xwidget.NewAppBar(
					"Mail",
					u.unifiedMails.ReadingPane,
					xwidget.NewIconButton(u.unifiedMails.ReadingPane.MakeReplyAction()),
					xwidget.NewIconButton(u.unifiedMails.ReadingPane.MakeReplyAllAction()),
					xwidget.NewIconButton(u.unifiedMails.ReadingPane.MakeForwardAction()),
					xwidget.NewIconButton(u.unifiedMails.ReadingPane.MakeDeleteAction()),
				)
				u.homeNav.PushAndHideNavBar(u.unifiedMailReadingPage)
			}
			var compose *xwidget.Button
			compose = xwidget.NewIconButton(theme.DocumentCreateIcon(), func() {
				u.unifiedMails.Compose(compose)
			})
			u.homeNav.Push(
				xwidget.NewAppBar(
					"Mail",
					u.unifiedMails.MessagePane,
					xwidget.NewIconButtonWithMenu(theme.FolderIcon(), unifiedMailMenu),
					compose,
				),
			)
		},
	)
	u.unifiedMails.OnUpdate = func(unread optional.Optional[int], missing int) {
		var s []string
		if x := formatMailUnread(unread); x != "" {
			s = append(s, x)
		}
		if loaded, total := u.unifiedMails.CharactersLoaded(); loaded < total {
			s = append(s, fmt.Sprintf("%d / %d characters loaded", loaded, total))
		}
		if missing > 0 {
			s = append(s, fmt.Sprintf("%d%% downloaded", 100-missing))
		}
		u.homeMailItem.Supporting = strings.Join(s, " • ")
		u.homeMailItem.Refresh()

		unifiedMailMenu.Items = u.unifiedMails.NavigationPane.MakeFolderMenu()
		unifiedMailMenu.Refresh()
	}

	u.homeStructuresItem = xwidget.NewNavListItem(
		"Structures",
		theme.NewThemedResource(icons.OfficeBuildingSvg),
		func() {
			u.homeNav.Push(xwidget.NewAppBar("Structures", u.unifiedStructures))
		},
	)
	u.unifiedStructures.OnUpdate = func(count optional.Optional[int]) {
		u.homeStructuresItem.Supporting = formatCount(count, func(v int) string {
			return ihumanize.Plural(v, "structure", "structures") + " reinforced"
		})
		u.homeStructuresItem.Refresh()
	}

	u.homeSkillsItem = xwidget.NewNavListItem(
		"Skills",
		theme.NewThemedResource(icons.SchoolSvg),
		func() {
			u.homeNav.Push(xwidget.NewAppBar("Skills", container.NewAppTabs(
				ui.NewTabItem("Training", u.training),
				ui.NewTabItem("Search", u.skillSearch),
			)))
		},
	)
	u.training.OnUpdate = func(expired optional.Optional[int]) {
		setNavItemSupportingWarning(u.homeSkillsItem, expired, "expired")
	}

	homeList := xwidget.NewNavList(
		u.homeCharactersItem,
		u.homeCorporationsItem,
		homeAssetSearchItem,
		xwidget.NewNavListItem(
			"Clones",
			theme.NewThemedResource(icons.HeadSnowflakeSvg),
			func() {
				u.homeNav.Push(xwidget.NewAppBar("Clones", container.NewAppTabs(
					ui.NewTabItem("Augmentations", u.augmentations),
					ui.NewTabItem("Jump Clones", u.clones),
				)))
			},
		),
		u.homeCommunicationsItem,
		u.homeContractsItem,
		u.homeColoniesItem,
		u.homeIndustryItem,
		xwidget.NewNavListItem(
			"Loyalty Points",
			theme.NewThemedResource(icons.HandHeartSvg),
			func() {
				u.homeNav.Push(xwidget.NewAppBar("Loyalty Points", u.loyaltyPoints))
			},
		),
		u.homeMailItem,
		xwidget.NewNavListItem(
			"Market Orders",
			theme.NewThemedResource(icons.ChartAreasplineSvg),
			func() {
				u.homeNav.Push(xwidget.NewAppBar("Market Orders",
					container.NewAppTabs(
						ui.NewTabItem("Buy", u.marketOrdersBuy),
						ui.NewTabItem("Sell", u.marketOrdersSell),
					),
				))
			},
		),
		u.homeSkillsItem,
		u.homeStructuresItem,
		u.homeWealthItem,
	)
	u.homeUpdateStatusItem = NewStatusBarItem(theme.NewThemedResource(icons.UpdateSvg), "?", func() {
		updatestatus.Show(u)
	})
	u.homeNav = xwidget.NewNavigator(xwidget.NewAppBar("Home", homeList, u.homeUpdateStatusItem))
}

func setNavItemSupportingWarning(item *xwidget.NavListItem, count optional.Optional[int], label string) {
	item.Supporting = formatCount(count, func(v int) string {
		return ihumanize.Comma(v) + " " + label
	})
	if v, ok := count.Value(); ok && v > 0 {
		item.SupportingImportance = widget.WarningImportance
		item.Trailing = theme.NewWarningThemedResource(theme.WarningIcon())
	} else {
		item.SupportingImportance = widget.MediumImportance
		item.Trailing = nil
	}
	item.Refresh()
}

func formatUnread(v int) string {
	return ihumanize.Comma(v) + " unread"
}

// formatMailUnread labels an unknown count, since it is shown next to other mail states.
func formatMailUnread(unread optional.Optional[int]) string {
	if unread.IsEmpty() {
		return "? unread"
	}
	return formatCount(unread, formatUnread)
}

func formatISKValue(value optional.Optional[float64], format string) string {
	v, ok := value.Value()
	if !ok {
		return "? ISK"
	}
	return humanize.FormatFloat(format, v) + " ISK"
}
