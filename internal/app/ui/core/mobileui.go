package core

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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
	"github.com/ErikKalkoken/evebuddy/internal/fynetools"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// MobileUI creates the UI for mobile.
type MobileUI struct {
	*baseUI
}

// NewMobileUI builds the UI and returns it.
func NewMobileUI(params UIParams) *MobileUI {
	u := &MobileUI{baseUI: newBaseUI(params)}

	var navBar *xwidget.NavBar

	// character destination
	fallbackAvatar, _ := fynetools.MakeAvatar(icons.Characterplaceholder64Jpeg)
	characterSelector := xwidget.NewIconButtonWithMenu(fallbackAvatar, fyne.NewMenu(""))
	newCharacterAppBar := func(title string, body fyne.CanvasObject, items ...fyne.CanvasObject) *xwidget.AppBar {
		items = append(items, characterSelector)
		return xwidget.NewAppBar(title, body, items...)
	}
	var characterNav *xwidget.Navigator

	const assetsTitle = "Asset Browser"
	navItemAssetBrowser := xwidget.NewNavListItem(
		assetsTitle,
		theme.NewThemedResource(icons.Inventory2Svg),
		func() {
			u.characterAssetBrowser.Navigation.OnSelected = func() {
				characterNav.PushAndHideNavBar(newCharacterAppBar(assetsTitle, u.characterAssetBrowser.Selected))
			}
			characterNav.Push(newCharacterAppBar(assetsTitle, container.NewHScroll(u.characterAssetBrowser.Navigation)))
		},
	)

	characterCommunicationsMenu := fyne.NewMenu("")
	navItemCommunications := xwidget.NewNavListItem(
		"Communications",
		theme.NewThemedResource(icons.MessageSvg),
		func() {
			u.characterCommunications.MessagePane.OnSelected = func() {
				characterNav.PushAndHideNavBar(
					newCharacterAppBar("Communications", u.characterCommunications.ReadingPane),
				)
			}
			characterNav.Push(
				newCharacterAppBar(
					"Communications",
					u.characterCommunications.MessagePane,
					xwidget.NewIconButtonWithMenu(theme.FolderIcon(), characterCommunicationsMenu),
				),
			)
		},
	)

	mailMenu := fyne.NewMenu("")

	var mailPage *xwidget.AppBar
	navItemMail := xwidget.NewNavListItem(
		"Mail",
		theme.MailComposeIcon(),
		func() {
			u.characterMails.MessagePane.OnSelected = func() {
				characterNav.PushAndHideNavBar(
					newCharacterAppBar(
						"Mail",
						u.characterMails.ReadingPane,
						xwidget.NewIconButton(u.characterMails.ReadingPane.MakeReplyAction()),
						xwidget.NewIconButton(u.characterMails.ReadingPane.MakeReplyAllAction()),
						xwidget.NewIconButton(u.characterMails.ReadingPane.MakeForwardAction()),
						xwidget.NewIconButton(u.characterMails.ReadingPane.MakeDeleteAction(func() {
							fyne.Do(func() {
								characterNav.Pop()
							})
						})),
					),
				)
			}
			mailPage = newCharacterAppBar(
				"Mail",
				u.characterMails.MessagePane,
				xwidget.NewIconButtonWithMenu(theme.FolderIcon(), mailMenu),
				xwidget.NewIconButton(u.characterMails.MakeComposeMessageAction()),
			)
			characterNav.Push(mailPage)
		},
	)

	navItemSkills := xwidget.NewNavListItem(
		"Skills",
		theme.NewThemedResource(icons.SchoolSvg),
		func() {
			characterNav.Push(
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

	navItemWallet := xwidget.NewNavListItem(
		"Wallet",
		theme.NewThemedResource(icons.AttachmoneySvg),
		func() {
			characterNav.Push(
				newCharacterAppBar("Wallet", u.characterWallet))
		},
	)

	characterList := xwidget.NewNavList(
		xwidget.NewNavListItem(
			"Character",
			theme.NewThemedResource(icons.PortraitSvg),
			func() {
				characterNav.Push(
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
		navItemAssetBrowser,
		xwidget.NewNavListItem(
			"Contacts",
			theme.NewThemedResource(icons.AccountSearchSvg),
			func() {
				characterNav.Push(
					newCharacterAppBar("Contacts", u.characterContacts))
			},
		),
		navItemCommunications,
		navItemMail,
		navItemSkills,
		navItemWallet,
	)

	u.characterMails.OnUpdate = func(unread, missing int) {
		var s []string
		if unread > 0 {
			s = append(s, fmt.Sprintf("%s unread", humanize.Comma(int64(unread))))
		}
		if missing > 0 {
			s = append(s, fmt.Sprintf("%d%% downloaded", 100-missing))
		}
		navItemMail.Supporting = strings.Join(s, " • ")
		navItemMail.Refresh()

		mailMenu.Items = u.characterMails.NavigationPane.MakeFolderMenu()
		mailMenu.Refresh()

		for !characterNav.IsRoot() && characterNav.Current() != mailPage {
			characterNav.Pop()
		}
	}

	u.characterCommunications.OnUpdate = func(count optional.Optional[int]) {
		var s string
		if v, ok := count.Value(); ok {
			s = fmt.Sprintf("%s unread", humanize.Comma(int64(v)))
		} else if count.ValueOrZero() > 0 {
			s = "?"
		}
		navItemCommunications.Supporting = s
		navItemCommunications.Refresh()

		characterCommunicationsMenu.Items = u.characterCommunications.NavigationPane.MakeFolderMenu()
		characterCommunicationsMenu.Refresh()
	}

	u.characterSkillQueue.OnUpdate = func(_, status string) {
		navItemSkills.Supporting = status
		navItemSkills.Refresh()
	}

	u.characterWallet.OnTopUpdate = func(b string) {
		navItemWallet.Supporting = b
		navItemWallet.Refresh()
	}

	characterPage := newCharacterAppBar("Characters", characterList)
	characterNav = xwidget.NewNavigator(characterPage)

	// corporation destination
	fallbackAvatar2, _ := fynetools.MakeAvatar(icons.Corporationplaceholder64Png)
	corpSelector := xwidget.NewIconButtonWithMenu(fallbackAvatar2, fyne.NewMenu(""))
	newCorpAppBar := func(title string, body fyne.CanvasObject, items ...fyne.CanvasObject) *xwidget.AppBar {
		items = append(items, corpSelector)
		return xwidget.NewAppBar(title, body, items...)
	}

	var corpNav *xwidget.Navigator

	const corpAssetBrowserTitle = "Asset Browser"
	corpAssetBrowserNav := xwidget.NewNavListItem(
		corpAssetBrowserTitle,
		theme.NewThemedResource(icons.Inventory2Svg),
		func() {
			u.corporationAssetBrowser.Navigation.OnSelected = func() {
				corpNav.PushAndHideNavBar(newCorpAppBar(corpAssetBrowserTitle, u.corporationAssetBrowser.Selected))
			}
			corpNav.Push(newCorpAppBar(corpAssetBrowserTitle, container.NewHScroll(u.corporationAssetBrowser.Navigation)))
		},
	)

	const corpAssetSearchTitle = "Asset Search"
	corpAssetSearchNav := xwidget.NewNavListItem(
		corpAssetSearchTitle,
		theme.NewThemedResource(icons.Inventory2Svg),
		func() {
			corpNav.Push(xwidget.NewAppBar(corpAssetSearchTitle, u.corporationAssetSearch,
				xwidget.NewIconButtonWithMenu(
					theme.MoreHorizontalIcon(),
					fyne.NewMenu("", u.corporationAssetSearch.MoreItems()...),
				),
			))
			u.corporationAssetSearch.Focus()
		},
	)

	var corpWalletItems []*xwidget.NavListItem
	corporationWalletNavs := make(map[app.Division]*xwidget.NavListItem)
	for _, d := range app.Divisions {
		corporationWalletNavs[d] = xwidget.NewNavListItem(
			d.DefaultWalletName(),
			theme.NewThemedResource(icons.CashSvg),
			func() {
				corpNav.Push(
					newCorpAppBar(
						corporationWalletNavs[d].Headline,
						u.corporationWallets[d],
					))
			},
		)
		corpWalletItems = append(corpWalletItems, corporationWalletNavs[d])
	}
	corpWalletList := xwidget.NewNavList(corpWalletItems...)
	corpWalletNav := xwidget.NewNavListItem(
		"Wallets",
		theme.NewThemedResource(icons.CashSvg),
		func() {
			corpNav.Push(
				newCorpAppBar(
					"Wallets",
					corpWalletList,
				))
		},
	)
	for _, d := range app.Divisions {
		u.corporationWallets[d].OnTopUpdate = func(top string) {
			fyne.Do(func() {
				corporationWalletNavs[d].Supporting = top
				corporationWalletNavs[d].Refresh()
			})
		}
		u.corporationWallets[d].NnNameUpdate = func(name string) {
			fyne.Do(func() {
				corporationWalletNavs[d].Headline = name
				corporationWalletNavs[d].Refresh()
			})
		}
	}

	corpContractsNav := xwidget.NewNavListItem(
		"Contracts",
		theme.NewThemedResource(icons.FileSignSvg),
		func() {
			corpNav.Push(newCorpAppBar("Contracts", u.corporationContracts))
		},
	)

	corpIndustryNav := xwidget.NewNavListItem(
		"Industry",
		theme.NewThemedResource(icons.FactorySvg),
		func() {
			corpNav.Push(newCorpAppBar("Industry", u.corporationIndyJobs))
		},
	)

	corpStructuresNav := xwidget.NewNavListItem(
		"Structures",
		theme.NewThemedResource(icons.OfficeBuildingSvg),
		func() {
			corpNav.Push(newCorpAppBar("Structures", u.corporationStructures))
		},
	)

	corpSheetNav := xwidget.NewNavListItem(
		"Corporation",
		theme.NewThemedResource(icons.PortraitSvg),
		func() {
			corpNav.Push(
				newCorpAppBar(
					"Corporation",
					container.NewAppTabs(
						ui.NewTabItem("Corporation", u.corporationSheet),
						ui.NewTabItem("Members", u.corporationMember),
					),
				))
		},
	)

	corpWealthNav := xwidget.NewNavListItem(
		"Wealth",
		theme.NewThemedResource(icons.GoldSvg),
		func() {
			corpNav.Push(newCorpAppBar("Wealth", u.corporationWealth))
		},
	)

	corpList := xwidget.NewNavList(
		slices.Concat([]*xwidget.NavListItem{
			corpSheetNav,
			corpAssetBrowserNav,
			corpAssetSearchNav,
			corpContractsNav,
			corpIndustryNav,
			corpStructuresNav,
			corpWealthNav,
			corpWalletNav,
		})...,
	)
	u.corporationContracts.OnUpdate = func(count int) {
		var badge string
		if count > 0 {
			badge = fmt.Sprintf("%d contracts active", count)
		}
		corpContractsNav.Supporting = badge
		corpContractsNav.Refresh()
	}
	u.corporationIndyJobs.OnUpdate = func(count int) {
		var badge string
		if count > 0 {
			badge = fmt.Sprintf("%s jobs ready", ihumanize.Comma(count))
		}
		corpIndustryNav.Supporting = badge
		corpIndustryNav.Refresh()
	}
	u.corporationStructures.OnUpdate = func(count int) {
		var badge string
		if count > 0 {
			badge = fmt.Sprintf("%s structures reinforced", ihumanize.Comma(count))
		}
		corpStructuresNav.Supporting = badge
		corpStructuresNav.Refresh()
	}
	u.onUpdateCorporationWalletTotals = func(balance optional.Optional[float64]) {
		corpWalletNav.Supporting = formatISKValueLong(balance, ui.FloatFormatISKRounded)
		corpWalletNav.Refresh()
	}

	corpPage := newCorpAppBar("Corporations", corpList)
	corpNav = xwidget.NewNavigator(corpPage)

	// other

	homeNav, updateStatus := makeHomeNav(u)

	searchNav := makeSearchNav(newCharacterAppBar, u)

	// more destination
	var moreNav *xwidget.Navigator
	navItemUpdateStatus := xwidget.NewNavListItem(
		"Update status",
		theme.NewThemedResource(icons.UpdateSvg),
		func() {
			updatestatus.Show(u)
		},
	)
	navItemManageCharacters := xwidget.NewNavListItem(
		"Manage characters",
		theme.NewThemedResource(icons.ManageaccountsSvg),
		func() {
			charactermanager.Show(u)
		},
	)

	navItemAbout := xwidget.NewNavListItem(
		"About",
		theme.InfoIcon(),
		func() {
			moreNav.Push(xwidget.NewAppBar("About", makeAboutPage(u.baseUI)))
		},
	)
	moreList := xwidget.NewNavList(
		xwidget.NewNavListItem(
			"Settings",
			theme.NewThemedResource(icons.TuneVariantSvg),
			func() {
				settings.Show(u)
			},
		),
		navItemManageCharacters,
		navItemUpdateStatus,
		navItemAbout,
	)
	moreNav = xwidget.NewNavigator(xwidget.NewAppBar("Manage", moreList))

	// navigation bar
	characterDest := xwidget.NewDestinationDef("Characters", theme.NewThemedResource(icons.AccountSvg), characterNav)
	characterDest.OnSelectedAgain = func() {
		characterNav.PopAll()
	}

	corpDest := xwidget.NewDestinationDef("Corporations", theme.NewThemedResource(icons.StarCircleOutlineSvg), corpNav)
	corpDest.OnSelectedAgain = func() {
		corpNav.PopAll()
	}

	homeDest := xwidget.NewDestinationDef("Home", theme.NewThemedResource(theme.HomeIcon()), homeNav)
	homeDest.OnSelectedAgain = func() {
		homeNav.PopAll()
	}

	searchDest := xwidget.NewDestinationDef("Search", theme.SearchIcon(), searchNav)
	searchDest.OnSelected = func() {
		u.gameSearch.Focus()
	}
	searchDest.OnSelectedAgain = func() {
		u.gameSearch.Reset()
	}

	moreDest := xwidget.NewDestinationDef("Manage", theme.SettingsIcon(), moreNav)
	moreDest.OnSelectedAgain = func() {
		moreNav.PopAll()
	}

	navBar = xwidget.NewNavBar(homeDest, characterDest, corpDest, searchDest, moreDest)
	homeNav.NavBar = navBar
	characterNav.NavBar = navBar
	corpNav.NavBar = navBar
	searchNav.NavBar = navBar

	u.snackbar.BottomMargin = theme.Padding() * 17

	w := u.MainWindow()
	w.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
		if ev.Name != mobile.KeyBack {
			return
		}
		id, ok := navBar.Selected()
		if !ok {
			return
		}
		switch id {
		case 0:
			homeNav.Pop()
		case 1:
			characterNav.Pop()
		case 2:
			corpNav.Pop()
		case 3:
			searchNav.Pop()
		case 4:
			moreNav.Pop()
		}
	})

	// initial state
	navBar.Disable(0)
	navBar.Disable(1)
	navBar.Disable(2)
	navBar.Disable(3)
	navBar.Select(4)

	togglePermittedSections := func() {
		sections, err := u.Corporation().PermittedSections(context.Background(), u.CurrentCorporation().IDOrZero())
		if err != nil {
			slog.Error("Failed to enable corporation tab", "error", err)
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
			setEnabled(corpAssetBrowserNav, sections.Contains(app.SectionCorporationAssets))
			setEnabled(corpIndustryNav, sections.Contains(app.SectionCorporationIndustryJobs))
			setEnabled(corpContractsNav, sections.Contains(app.SectionCorporationContracts))
			setEnabled(corpWalletNav, sections.Contains(app.SectionCorporationWalletBalances))
			setEnabled(corpWealthNav, sections.Contains(app.SectionCorporationAssets) &&
				sections.Contains(app.SectionCorporationContracts) &&
				sections.Contains(app.SectionCorporationWalletBalances))
		})
	}

	u.onUpdateStatus = func(ctx context.Context) {
		go togglePermittedSections()
		var wg sync.WaitGroup
		wg.Go(func() {
			u.setCharacterSwitchMenu(
				ctx,
				func(items []*fyne.MenuItem) {
					characterSelector.SetMenuItems(items)
				},
				characterSelector.Refresh,
			)
		})
		wg.Go(func() {
			u.setCorporationSwitchMenu(
				ctx,
				func(items []*fyne.MenuItem) {
					corpSelector.SetMenuItems(items)
				},
				corpSelector.Refresh,
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
					navBar.Disable(2)
					id, ok := navBar.Selected()
					if ok && id == 2 {
						navBar.Select(0)
					}
				})
				return
			}
			fyne.Do(func() {
				navBar.Enable(2)
			})
		})
		wg.Wait()
	}

	u.Signals().CurrentCharacterExchanged.AddListener(func(_ context.Context, c *app.Character) {
		fyne.Do(func() {
			if c == nil {
				navBar.Disable(0)
				navBar.Disable(1)
				navBar.Disable(3)
				navBar.Select(4)
			} else {
				wasDisabled := !navBar.Enabled(0)
				navBar.Enable(0)
				navBar.Enable(1)
				navBar.Enable(3)
				if wasDisabled {
					navBar.Select(0)
				}
			}
		})
	})
	u.onSetCharacter = func(c *app.Character) {
		fyne.Do(func() {
			characterPage.SetTitle(c.EveCharacter.Name)
			characterNav.PopAll()
		})
		go u.SetCharacterAvatarAsync(c.ID, func(r fyne.Resource) {
			fyne.Do(func() {
				characterSelector.SetIcon(r)
			})
		})
		go func() {
			fyne.Do(func() {
				u.characterCommunications.MessagePane.ResetHeaders()
			})
		}()
	}
	u.onShowCharacter = func() {
		fyne.Do(func() {
			navBar.Select(1)
		})
	}
	u.onShowCorporation = func() {
		fyne.Do(func() {
			navBar.Select(2)
		})
	}

	u.onSetCorporation = func(c *app.Corporation) {
		fyne.Do(func() {
			corpPage.SetTitle(c.EveCorporation.Name)
			corpNav.PopAll()
		})
		go u.setCorporationAvatarAsync(c.ID, func(r fyne.Resource) {
			fyne.Do(func() {
				corpSelector.SetIcon(r)
			})
		})
		go togglePermittedSections()
	}

	var hasUpdateError, hasUpdate, hasScopeError atomic.Bool
	refreshMoreBadge := func() {
		if hasUpdateError.Load() || hasUpdate.Load() || hasScopeError.Load() || u.isOffline.Load() {
			var importance widget.Importance
			if hasUpdateError.Load() {
				importance = widget.DangerImportance
			} else if hasScopeError.Load() || u.isOffline.Load() {
				importance = widget.WarningImportance
			} else if hasUpdate.Load() {
				importance = widget.HighImportance
			}
			navBar.ShowBadge(4, importance)
		} else {
			navBar.HideBadge(4)
		}
	}

	u.onShowAndRun = func() {
		if u.isFakeMobile {
			u.MainWindow().Resize(fyne.NewSize(340, 700))
			u.MainWindow().SetFixedSize(true)
		}
	}

	updateCharacterCount := func(ctx context.Context) {
		ids, err := u.cs.ListCharacterIDs(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("updating character count", "error", err)
			return
		}
		fyne.Do(func() {
			navItemManageCharacters.Supporting = fmt.Sprintf("%d characters", ids.Size())
			navItemManageCharacters.Refresh()
		})
	}

	updateUpdateStatus := func(_ context.Context) {
		set := func(full, short string, i widget.Importance, icon fyne.Resource) {
			fyne.Do(func() {
				refreshMoreBadge()
				navItemUpdateStatus.Supporting = full
				navItemUpdateStatus.SupportingImportance = i
				navItemUpdateStatus.Trailing = icon
				navItemUpdateStatus.Refresh()
				updateStatus.SetTextAndImportance(short, i)
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
			hasUpdateError.Store(true)
		} else {
			hasUpdateError.Store(false)
		}
		set(status.Display(), status.DisplayShort(), status.Status().ToImportance(), icon)
	}

	u.onAppFirstStarted = func() {
		// signals
		u.Signals().CharacterAdded.AddListener(func(ctx context.Context, _ *app.Character) {
			updateCharacterCount(ctx)
			updateUpdateStatus(ctx)
		})
		u.Signals().CharacterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
			updateCharacterCount(ctx)
			updateUpdateStatus(ctx)
		})
		u.Signals().CharacterSectionUpdated.AddListener(func(ctx context.Context, _ app.CharacterSectionUpdated) {
			updateUpdateStatus(ctx)
		})
		u.Signals().CorporationSectionUpdated.AddListener(func(ctx context.Context, _ app.CorporationSectionUpdated) {
			updateUpdateStatus(ctx)
		})
		u.Signals().EveUniverseSectionUpdated.AddListener(func(ctx context.Context, _ app.EveUniverseSectionUpdated) {
			updateUpdateStatus(ctx)
		})

		ctx := context.Background()
		updateCharacterCount(ctx)
		updateUpdateStatus(ctx)

		if !u.isOfflineMode {
			tickerNewVersion := time.NewTicker(3600 * time.Second)
			go func() {
				for {
					func() {
						v, err := u.availableUpdate(ctx)
						if err != nil {
							slog.Error("fetch github version for menu info", "error", err)
							return
						}
						if v.IsRemoteNewer {
							hasUpdate.Store(true)
							fyne.Do(func() {
								refreshMoreBadge()
								navItemAbout.Supporting = "Update available"
								navItemAbout.SupportingImportance = widget.HighImportance
								navItemAbout.Trailing = theme.NewPrimaryThemedResource(icons.Numeric1CircleSvg)
								navItemAbout.Refresh()
							})
						} else {
							hasUpdate.Store(false)
							fyne.Do(func() {
								refreshMoreBadge()
								navItemAbout.Supporting = ""
								navItemAbout.SupportingImportance = widget.MediumImportance
								navItemAbout.Trailing = nil
								navItemAbout.Refresh()
							})
						}
					}()
					<-tickerNewVersion.C
				}
			}()
		}
	}

	u.onUpdateMissingScope = func(characterCount int) {
		var icon fyne.Resource
		if characterCount > 0 {
			icon = theme.NewWarningThemedResource(theme.WarningIcon())
			hasScopeError.Store(true)
		} else {
			icon = nil
			hasScopeError.Store(false)
		}
		fyne.Do(func() {
			navItemManageCharacters.Trailing = icon
			moreList.Refresh()
			refreshMoreBadge()
		})
	}

	w.SetContent(fynetooltip.AddWindowToolTipLayer(navBar, w.Canvas()))
	return u
}

func makeSearchNav(newCharacterAppBar func(title string, body fyne.CanvasObject, items ...fyne.CanvasObject) *xwidget.AppBar, u *MobileUI) *xwidget.Navigator {
	searchNav := xwidget.NewNavigator(
		newCharacterAppBar("Search", u.gameSearch),
	)
	return searchNav
}

func makeHomeNav(u *MobileUI) (*xwidget.Navigator, *StatusBarItem) {
	var homeNav *xwidget.Navigator
	var homeList *xwidget.NavList

	navItemColonies2 := xwidget.NewNavListItem(
		"Colonies",
		theme.NewThemedResource(icons.EarthSvg),
		func() {
			homeNav.PushAndHideNavBar(xwidget.NewAppBar("Colonies", u.colonies))
		},
	)
	u.colonies.OnUpdate = func(_, notWorking int) {
		setNavItemSupportingWarning(navItemColonies2, notWorking, "not working")
	}

	navItemIndustry := xwidget.NewNavListItem(
		"Industry",
		theme.NewThemedResource(icons.FactorySvg),
		func() {
			homeNav.Push(xwidget.NewAppBar("Industry",
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
	u.industryJobs.OnUpdate = func(count int) {
		var badge string
		if count > 0 {
			badge = fmt.Sprintf("%s jobs ready", ihumanize.Comma(count))
		}
		navItemIndustry.Supporting = badge
		navItemIndustry.Refresh()
	}

	navItemContracts := xwidget.NewNavListItem(
		"Contracts",
		theme.NewThemedResource(icons.FileSignSvg),
		func() {
			homeNav.Push(xwidget.NewAppBar("Contracts",
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
	u.contractList.OnUpdate = func(count int) {
		var badge string
		if count > 0 {
			badge = fmt.Sprintf("%d contracts active", count)
		}
		navItemContracts.Supporting = badge
		navItemContracts.Refresh()
	}

	navItemWealth := xwidget.NewNavListItem(
		"Wealth",
		theme.NewThemedResource(icons.GoldSvg),
		func() {
			homeNav.PushAndHideNavBar(xwidget.NewAppBar("Wealth", u.wealth))
		},
	)
	u.wealth.OnUpdate = func(total optional.Optional[float64]) {
		navItemWealth.Supporting = formatISKValueLong(total, ui.FloatFormatISKRounded)
		navItemWealth.Refresh()
	}

	navItemAssetSearch := xwidget.NewNavListItem(
		"Assets",
		theme.NewThemedResource(icons.Inventory2Svg),
		func() {
			homeNav.Push(xwidget.NewAppBar("Assets", u.assetSearchAll,
				xwidget.NewIconButtonWithMenu(
					theme.MoreHorizontalIcon(),
					fyne.NewMenu("", u.assetSearchAll.MoreItems()...),
				),
			))
			u.assetSearchAll.Focus()
		},
	)

	navItemCharacters := xwidget.NewNavListItem(
		"Characters",
		theme.NewThemedResource(icons.PortraitSvg),
		func() {
			homeNav.Push(xwidget.NewAppBar("Characters", u.characterOverview))
		},
	)
	u.characterOverview.OnUpdate = func(characters int) {
		navItemCharacters.Supporting = fmt.Sprintf("%d characters", characters)
		navItemCharacters.Refresh()
	}

	navItemCorporations := xwidget.NewNavListItem(
		"Corporations",
		theme.NewThemedResource(icons.StarCircleOutlineSvg),
		func() {
			homeNav.Push(xwidget.NewAppBar("Corporations", u.corporationOverview))
		},
	)
	u.corporationOverview.OnUpdate = func(corporations int) {
		navItemCorporations.Supporting = fmt.Sprintf("%d corporations", corporations)
		navItemCorporations.Refresh()
	}

	unifiedCommunicationsMenu := fyne.NewMenu("")
	navItemUnifiedCommunications := xwidget.NewNavListItem(
		"Communications",
		theme.NewThemedResource(icons.MessageSvg),
		func() {
			u.unifiedCommunications.MessagePane.OnSelected = func() {
				homeNav.PushAndHideNavBar(
					xwidget.NewAppBar("Communications", u.unifiedCommunications.ReadingPane),
				)
			}
			homeNav.Push(
				xwidget.NewAppBar(
					"Communications",
					u.unifiedCommunications.MessagePane,
					xwidget.NewIconButtonWithMenu(theme.FolderIcon(), unifiedCommunicationsMenu),
				),
			)
		},
	)
	u.unifiedCommunications.OnUpdate = func(count optional.Optional[int]) {
		var s string
		if v, ok := count.Value(); ok {
			s = fmt.Sprintf("%s unread", humanize.Comma(int64(v)))
		} else if count.ValueOrZero() > 0 {
			s = "?"
		}
		navItemUnifiedCommunications.Supporting = s
		navItemUnifiedCommunications.Refresh()

		unifiedCommunicationsMenu.Items = u.unifiedCommunications.NavigationPane.MakeFolderMenu()
		unifiedCommunicationsMenu.Refresh()
	}

	unifiedMailMenu := fyne.NewMenu("")
	navItemUnifiedMail := xwidget.NewNavListItem(
		"Mail",
		theme.MailComposeIcon(),
		func() {
			u.unifiedMails.MessagePane.OnSelected = func() {
				homeNav.PushAndHideNavBar(
					xwidget.NewAppBar(
						"Mail",
						u.unifiedMails.ReadingPane,
						xwidget.NewIconButton(u.unifiedMails.ReadingPane.MakeReplyAction()),
						xwidget.NewIconButton(u.unifiedMails.ReadingPane.MakeReplyAllAction()),
						xwidget.NewIconButton(u.unifiedMails.ReadingPane.MakeForwardAction()),
						xwidget.NewIconButton(u.unifiedMails.ReadingPane.MakeDeleteAction(func() {
							fyne.Do(func() {
								homeNav.Pop()
							})
						})),
					),
				)
			}
			var compose *xwidget.Button
			compose = xwidget.NewIconButton(theme.DocumentCreateIcon(), func() {
				u.unifiedMails.Compose(compose)
			})
			homeNav.Push(
				xwidget.NewAppBar(
					"Mail",
					u.unifiedMails.MessagePane,
					xwidget.NewIconButtonWithMenu(theme.FolderIcon(), unifiedMailMenu),
					compose,
				),
			)
		},
	)
	u.unifiedMails.OnUpdate = func(unread, missing int) {
		var s []string
		if unread > 0 {
			s = append(s, fmt.Sprintf("%s unread", humanize.Comma(int64(unread))))
		}
		if missing > 0 {
			s = append(s, fmt.Sprintf("%d%% downloaded", 100-missing))
		}
		navItemUnifiedMail.Supporting = strings.Join(s, " • ")
		navItemUnifiedMail.Refresh()

		unifiedMailMenu.Items = u.unifiedMails.NavigationPane.MakeFolderMenu()
		unifiedMailMenu.Refresh()
	}

	navItemUnifiedStructures := xwidget.NewNavListItem(
		"Structures",
		theme.NewThemedResource(icons.OfficeBuildingSvg),
		func() {
			homeNav.Push(xwidget.NewAppBar("Structures", u.unifiedStructures))
		},
	)
	u.unifiedStructures.OnUpdate = func(count int) {
		var badge string
		if count > 0 {
			badge = fmt.Sprintf("%s structures reinforced", ihumanize.Comma(count))
		}
		navItemUnifiedStructures.Supporting = badge
		navItemUnifiedStructures.Refresh()
	}

	navItemSkills := xwidget.NewNavListItem(
		"Skills",
		theme.NewThemedResource(icons.SchoolSvg),
		func() {
			homeNav.Push(xwidget.NewAppBar("Skills", container.NewAppTabs(
				ui.NewTabItem("Training", u.training),
				ui.NewTabItem("Search", u.skillSearch),
			), xwidget.NewIconButtonWithMenu(
				theme.MoreHorizontalIcon(),
				fyne.NewMenu("", u.training.MoreItems()...),
			)))
		},
	)
	u.training.OnUpdate = func(expired int) {
		setNavItemSupportingWarning(navItemSkills, expired, "expired")
	}

	homeList = xwidget.NewNavList(
		navItemCharacters,
		navItemCorporations,
		navItemAssetSearch,
		xwidget.NewNavListItem(
			"Clones",
			theme.NewThemedResource(icons.HeadSnowflakeSvg),
			func() {
				homeNav.Push(xwidget.NewAppBar("Clones", container.NewAppTabs(
					ui.NewTabItem("Augmentations", u.augmentations),
					ui.NewTabItem("Jump Clones", u.clones),
				)))
			},
		),
		navItemUnifiedCommunications,
		navItemContracts,
		navItemColonies2,
		navItemIndustry,
		xwidget.NewNavListItem(
			"Loyalty Points",
			theme.NewThemedResource(icons.HandHeartSvg),
			func() {
				homeNav.Push(xwidget.NewAppBar("Loyalty Points", u.loyaltyPoints))
			},
		),
		navItemUnifiedMail,
		xwidget.NewNavListItem(
			"Market Orders",
			theme.NewThemedResource(icons.ChartAreasplineSvg),
			func() {
				homeNav.Push(xwidget.NewAppBar("Market Orders",
					container.NewAppTabs(
						ui.NewTabItem("Buy", u.marketOrdersBuy),
						ui.NewTabItem("Sell", u.marketOrdersSell),
					),
				))
			},
		),
		navItemSkills,
		navItemUnifiedStructures,
		navItemWealth,
	)
	status := NewStatusBarItem(theme.NewThemedResource(icons.UpdateSvg), "?", func() {
		updatestatus.Show(u)
	})
	homeNav = xwidget.NewNavigator(xwidget.NewAppBar("Home", homeList, status))
	return homeNav, status
}

func setNavItemSupportingWarning(item *xwidget.NavListItem, count int, label string) {
	if count > 0 {
		item.Supporting = fmt.Sprintf("%d %s", count, label)
		item.SupportingImportance = widget.WarningImportance
		item.Trailing = theme.NewWarningThemedResource(theme.WarningIcon())
	} else {
		item.Supporting = ""
		item.SupportingImportance = widget.MediumImportance
		item.Trailing = nil
	}
	item.Refresh()
}

func formatISKValueLong(value optional.Optional[float64], format string) string {
	v, ok := value.Value()
	if !ok {
		return "? ISK"
	}
	return ui.FormatISKAmountLong(v, format)
}
