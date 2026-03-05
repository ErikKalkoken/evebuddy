package ui

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/mobile"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/dustin/go-humanize"
	fynetooltip "github.com/dweymouth/fyne-tooltip"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/fynetools"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

// MobileUI creates the UI for mobile.
type MobileUI struct {
	*baseUI
}

// NewMobileUI builds the UI and returns it.
func NewMobileUI(bu *baseUI) *MobileUI {
	u := &MobileUI{baseUI: bu}

	var navBar *iwidget.NavBar

	makeAppBarIcons := func(items ...*kxwidget.IconButton) []fyne.CanvasObject {
		var x []fyne.CanvasObject
		for _, ib := range items {
			x = append(x, ib)
		}
		return x
	}

	// character destination
	fallbackAvatar, _ := fynetools.MakeAvatar(icons.Characterplaceholder64Jpeg)
	characterSelector := kxwidget.NewIconButtonWithMenu(fallbackAvatar, fyne.NewMenu(""))
	newCharacterAppBar := func(title string, body fyne.CanvasObject, items ...*kxwidget.IconButton) *iwidget.AppBar {
		items = append(items, characterSelector)
		return iwidget.NewAppBar(title, body, makeAppBarIcons(items...)...)
	}
	var characterNav *iwidget.Navigator

	const assetsTitle = "Asset Browser"
	navItemAssetBrowser := iwidget.NewNavListItemWithIcon(
		assetsTitle,
		theme.NewThemedResource(icons.Inventory2Svg),
		func() {
			u.characterAssetBrowser.Navigation.OnSelected = func() {
				characterNav.PushAndHideNavBar(newCharacterAppBar(assetsTitle, u.characterAssetBrowser.Selected))
			}
			characterNav.Push(newCharacterAppBar(assetsTitle, container.NewHScroll(u.characterAssetBrowser.Navigation)))
		},
	)

	communicationsMenu := fyne.NewMenu("")
	navItemCommunications := iwidget.NewNavListItemWithIcon(
		"Communications",
		theme.NewThemedResource(icons.MessageSvg),
		func() {
			u.characterCommunications.OnSelected = func() {
				characterNav.PushAndHideNavBar(
					newCharacterAppBar("Communications", u.characterCommunications.Detail),
				)
			}
			characterNav.Push(
				newCharacterAppBar(
					"Communications",
					u.characterCommunications.Notifications,
					kxwidget.NewIconButtonWithMenu(theme.FolderIcon(), communicationsMenu),
				),
			)
		},
	)

	mailMenu := fyne.NewMenu("")
	u.characterMails.onSendMessage = func(c *app.Character, mode app.SendMailMode, mail *app.CharacterMail) {
		page := newCharacterSendMail(bu, c, mode, mail)
		if mode != app.SendMailNew {
			characterNav.Pop() // FIXME: Workaround to avoid pushing upon page w/o navbar
		}
		characterNav.PushAndHideNavBar(
			iwidget.NewAppBar(
				"Send Mail",
				page,
				kxwidget.NewIconButton(theme.MailSendIcon(), func() {
					if page.SendAction() {
						characterNav.Pop()
					}
				}),
			),
		)
	}
	navItemMail := iwidget.NewNavListItemWithIcon(
		"Mail",
		theme.MailComposeIcon(),
		func() {
			u.characterMails.onSelected = func() {
				characterNav.PushAndHideNavBar(
					newCharacterAppBar(
						"Mail",
						u.characterMails.Detail,
						kxwidget.NewIconButton(u.characterMails.MakeReplyAction()),
						kxwidget.NewIconButton(u.characterMails.MakeReplyAllAction()),
						kxwidget.NewIconButton(u.characterMails.MakeForwardAction()),
						kxwidget.NewIconButton(u.characterMails.MakeDeleteAction(func() {
							characterNav.Pop()
						})),
					),
				)
			}
			characterNav.Push(
				newCharacterAppBar(
					"Mail",
					u.characterMails.Headers,
					kxwidget.NewIconButtonWithMenu(theme.FolderIcon(), mailMenu),
					kxwidget.NewIconButton(u.characterMails.makeComposeMessageAction()),
				))
		},
	)

	navItemSkills := iwidget.NewNavListItemWithIcon(
		"Skills",
		theme.NewThemedResource(icons.SchoolSvg),
		func() {
			characterNav.Push(
				newCharacterAppBar(
					"Skills",
					container.NewAppTabs(
						container.NewTabItem("Catalogue", u.characterSkillCatalogue),
						container.NewTabItem("Training", u.characterSkillQueue),
						container.NewTabItem("Ships", u.characterShips),
					),
				))
		},
	)

	navItemWallet := iwidget.NewNavListItemWithIcon(
		"Wallet",
		theme.NewThemedResource(icons.AttachmoneySvg),
		func() {
			characterNav.Push(
				newCharacterAppBar("Wallet", u.characterWallet))
		},
	)

	characterList := iwidget.NewNavList(
		iwidget.NewNavListItemWithIcon(
			"Character Sheet",
			theme.NewThemedResource(icons.PortraitSvg),
			func() {
				characterNav.Push(
					newCharacterAppBar(
						"Character Sheet",
						container.NewAppTabs(
							container.NewTabItem("Character", u.characterSheet),
							container.NewTabItem("Corporation", u.characterCorporation),
							container.NewTabItem("Augmentations", u.characterAugmentations),
							container.NewTabItem("Clones", u.characterJumpClones),
							container.NewTabItem("Attributes", u.characterAttributes),
							container.NewTabItem("Bio", u.characterBiography),
						),
					))
			},
		),
		navItemAssetBrowser,
		iwidget.NewNavListItemWithIcon(
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

	u.characterMails.onUpdate = func(unread, missing int) {
		var s []string
		if unread > 0 {
			s = append(s, fmt.Sprintf("%s unread", humanize.Comma(int64(unread))))
		}
		if missing > 0 {
			s = append(s, fmt.Sprintf("%d%% downloaded", 100-missing))
		}
		navItemMail.Supporting = strings.Join(s, " • ")
		navItemMail.Refresh()
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
	}

	u.characterSkillQueue.OnUpdate = func(_, status string) {
		navItemSkills.Supporting = status
		navItemSkills.Refresh()
	}

	u.characterWallet.onTopUpdate = func(b string) {
		navItemWallet.Supporting = b
		navItemWallet.Refresh()
	}

	characterPage := newCharacterAppBar("Characters", characterList)
	characterNav = iwidget.NewNavigator(characterPage)

	// corporation destination
	fallbackAvatar2, _ := fynetools.MakeAvatar(icons.Corporationplaceholder64Png)
	corpSelector := kxwidget.NewIconButtonWithMenu(fallbackAvatar2, fyne.NewMenu(""))
	newCorpAppBar := func(title string, body fyne.CanvasObject, items ...*kxwidget.IconButton) *iwidget.AppBar {
		items = append(items, corpSelector)
		return iwidget.NewAppBar(title, body, makeAppBarIcons(items...)...)
	}

	var corpNav *iwidget.Navigator

	const corpAssetBrowserTitle = "Asset Browser"
	corpAssetBrowserNav := iwidget.NewNavListItemWithIcon(
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
	corpAssetSearchNav := iwidget.NewNavListItemWithIcon(
		corpAssetSearchTitle,
		theme.NewThemedResource(icons.Inventory2Svg),
		func() {
			corpNav.Push(iwidget.NewAppBar(corpAssetSearchTitle, u.corporationAssetSearch))
			u.corporationAssetSearch.focus()
		},
	)

	var corpWalletItems []*iwidget.NavListItem
	corporationWalletNavs := make(map[app.Division]*iwidget.NavListItem)
	for _, d := range app.Divisions {
		corporationWalletNavs[d] = iwidget.NewNavListItemWithIcon(
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
	corpWalletList := iwidget.NewNavList(corpWalletItems...)
	corpWalletNav := iwidget.NewNavListItemWithIcon(
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
		u.corporationWallets[d].onTopUpdate = func(top string) {
			fyne.Do(func() {
				corporationWalletNavs[d].Supporting = top
				corporationWalletNavs[d].Refresh()
			})
		}
		u.corporationWallets[d].onNameUpdate = func(name string) {
			fyne.Do(func() {
				corporationWalletNavs[d].Headline = name
				corporationWalletNavs[d].Refresh()
			})
		}
	}

	corpContractsNav := iwidget.NewNavListItemWithIcon(
		"Contracts",
		theme.NewThemedResource(icons.FileSignSvg),
		func() {
			corpNav.Push(newCorpAppBar("Contracts", u.corporationContracts))
		},
	)

	corpIndustryNav := iwidget.NewNavListItemWithIcon(
		"Industry",
		theme.NewThemedResource(icons.FactorySvg),
		func() {
			corpNav.Push(newCorpAppBar("Industry", u.corporationIndyJobs))
		},
	)

	corpStructuresNav := iwidget.NewNavListItemWithIcon(
		"Structures",
		theme.NewThemedResource(icons.OfficeBuildingSvg),
		func() {
			corpNav.Push(newCorpAppBar("Structures", u.corporationStructures))
		},
	)

	corpSheetNav := iwidget.NewNavListItemWithIcon(
		"Corporation Sheet",
		theme.NewThemedResource(icons.PortraitSvg),
		func() {
			corpNav.Push(
				newCorpAppBar(
					"Corporation Sheet",
					container.NewAppTabs(
						container.NewTabItem("Corporation", u.corporationSheet),
						container.NewTabItem("Members", u.corporationMember),
					),
				))
		},
	)

	corpList := iwidget.NewNavList(
		slices.Concat([]*iwidget.NavListItem{
			corpSheetNav,
			corpAssetBrowserNav,
			corpAssetSearchNav,
			corpContractsNav,
			corpIndustryNav,
			corpStructuresNav,
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
	u.onUpdateCorporationWalletTotals = func(balance float64, ok bool) {
		var s string
		if !ok {
			s = ""
		} else {
			s = fmt.Sprintf("%s ISK", humanize.FormatFloat(app.FloatFormat, balance))
			if balance > 1000 {
				s += fmt.Sprintf(" (%s)", ihumanize.NumberF(balance, 1))
			}
		}
		corpWalletNav.Supporting = s
		corpWalletNav.Refresh()
	}

	corpPage := newCorpAppBar("Corporations", corpList)
	corpNav = iwidget.NewNavigator(corpPage)

	// other

	homeNav := makeHomeNav(u)

	searchNav := makeSearchNav(newCharacterAppBar, u)

	// more destination
	var moreNav *iwidget.Navigator
	navItemUpdateStatus := iwidget.NewNavListItemWithIcon(
		"Update status",
		theme.NewThemedResource(icons.UpdateSvg),
		func() {
			showUpdateStatusWindow(u.baseUI)
		},
	)
	navItemManageCharacters := iwidget.NewNavListItemWithIcon(
		"Manage characters",
		theme.NewThemedResource(icons.ManageaccountsSvg),
		func() {
			showManageCharactersWindow(u.baseUI)
		},
	)

	navItemAbout := iwidget.NewNavListItemWithIcon(
		"About",
		theme.InfoIcon(),
		func() {
			moreNav.Push(iwidget.NewAppBar("About", makeAboutPage(u.baseUI)))
		},
	)
	moreList := iwidget.NewNavList(
		iwidget.NewNavListItemWithIcon(
			"Settings",
			theme.NewThemedResource(icons.CogSvg),
			func() {
				ShowSettingsWindow(u.baseUI, u.settings, u.isMobile)
			},
		),
		navItemManageCharacters,
		navItemUpdateStatus,
		navItemAbout,
	)
	moreNav = iwidget.NewNavigator(iwidget.NewAppBar("More", moreList))

	// navigation bar
	characterDest := iwidget.NewDestinationDef("Characters", theme.NewThemedResource(icons.AccountSvg), characterNav)
	characterDest.OnSelectedAgain = func() {
		characterNav.PopAll()
	}

	corpDest := iwidget.NewDestinationDef("Corporations", theme.NewThemedResource(icons.StarCircleOutlineSvg), corpNav)
	corpDest.OnSelectedAgain = func() {
		corpNav.PopAll()
	}

	homeDest := iwidget.NewDestinationDef("Home", theme.NewThemedResource(theme.HomeIcon()), homeNav)
	homeDest.OnSelectedAgain = func() {
		homeNav.PopAll()
	}

	searchDest := iwidget.NewDestinationDef("Search", theme.SearchIcon(), searchNav)
	searchDest.OnSelected = func() {
		u.gameSearch.focus()
	}
	searchDest.OnSelectedAgain = func() {
		u.gameSearch.reset()
	}

	moreDest := iwidget.NewDestinationDef("More", theme.MenuIcon(), moreNav)
	moreDest.OnSelectedAgain = func() {
		moreNav.PopAll()
	}

	navBar = iwidget.NewNavBar(homeDest, characterDest, corpDest, searchDest, moreDest)
	homeNav.NavBar = navBar
	characterNav.NavBar = navBar
	corpNav.NavBar = navBar
	searchNav.NavBar = navBar

	u.snackbar.Bottom = 90

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
		sections, err := u.rs.PermittedSections(context.Background(), u.currentCorporationID())
		if err != nil {
			slog.Error("Failed to enable corporation tab", "error", err)
			sections.Clear()
		}
		fyne.Do(func() {
			if sections.Contains(app.SectionCorporationAssets) {
				corpAssetBrowserNav.IsDisabled = false
			} else {
				corpAssetBrowserNav.IsDisabled = true
			}
			if sections.Contains(app.SectionCorporationIndustryJobs) {
				corpIndustryNav.IsDisabled = false
			} else {
				corpIndustryNav.IsDisabled = true
			}
			if sections.Contains(app.SectionCorporationWalletBalances) {
				corpWalletNav.IsDisabled = false
			} else {
				corpWalletNav.IsDisabled = true
			}
			corpList.Refresh()
		})
	}

	u.onUpdateStatus = func(ctx context.Context) {
		go togglePermittedSections()
		go func() {
			items := u.makeCharacterSwitchMenu(characterSelector.Refresh)
			fyne.Do(func() {
				characterSelector.SetMenuItems(items)
			})
		}()
		go func() {
			items := u.makeCorporationSwitchMenu(corpSelector.Refresh)
			fyne.Do(func() {
				corpSelector.SetMenuItems(items)
			})
		}()
		go func() {
			cc, err := u.ListCorporationsForSelection()
			if err != nil {
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
		}()
	}

	u.currentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
		fyne.Do(func() {
			mailMenu.Items = u.characterMails.makeFolderMenu()
			mailMenu.Refresh()
			communicationsMenu.Items = u.characterCommunications.makeFolderMenu()
			communicationsMenu.Refresh()
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
		go u.setCharacterAvatar(c.ID, func(r fyne.Resource) {
			fyne.Do(func() {
				characterSelector.SetIcon(r)
			})
		})
		ctx := context.Background()
		go u.characterMails.resetCurrentFolder(ctx)
		go u.characterCommunications.resetCurrentFolder(ctx)
	}
	u.onShowCharacter = func() {
		navBar.Select(1)
	}

	u.onSetCorporation = func(c *app.Corporation) {
		fyne.Do(func() {
			corpPage.SetTitle(c.EveCorporation.Name)
			corpNav.PopAll()
		})
		go u.setCorporationAvatar(c.ID, func(r fyne.Resource) {
			fyne.Do(func() {
				corpSelector.SetIcon(r)
			})
		})
		go togglePermittedSections()
	}

	var hasUpdateError, isOffline, hasUpdate, hasScopeError atomic.Bool
	refreshMoreBadge := func() {
		if hasUpdateError.Load() || hasUpdate.Load() || hasScopeError.Load() || isOffline.Load() {
			var importance widget.Importance
			if hasUpdateError.Load() {
				importance = widget.DangerImportance
			} else if hasScopeError.Load() || isOffline.Load() {
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

	updateCharacterCount := func(_ context.Context) {
		s := fmt.Sprintf("%d characters", u.scs.ListCharacterIDs().Size())
		fyne.Do(func() {
			navItemManageCharacters.Supporting = s
			navItemManageCharacters.Refresh()
		})
	}

	updateUpdateStatus := func(_ context.Context) {
		set := func(s string, icon fyne.Resource) {
			fyne.Do(func() {
				refreshMoreBadge()
				navItemUpdateStatus.Supporting = s
				navItemUpdateStatus.Trailing = icon
				navItemUpdateStatus.Refresh()
			})
		}

		if u.ess.IsDailyDowntime() {
			isOffline.Store(true)
			set(
				fmt.Sprintf("Off during daily downtime: %s", u.ess.DailyDowntime()),
				theme.NewWarningThemedResource(theme.WarningIcon()),
			)
			return
		}
		isOffline.Store(false)
		status := u.scs.Summary()
		var icon fyne.Resource
		if status.Errors > 0 {
			icon = theme.NewErrorThemedResource(theme.WarningIcon())
			hasUpdateError.Store(true)
		} else {
			hasUpdateError.Store(false)
		}
		set(status.Display(), icon)
	}

	u.onAppFirstStarted = func() {
		// signals
		u.characterAdded.AddListener(func(ctx context.Context, _ *app.Character) {
			updateCharacterCount(ctx)
			updateUpdateStatus(ctx)
		})
		u.characterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
			updateCharacterCount(ctx)
			updateUpdateStatus(ctx)
		})
		u.characterSectionUpdated.AddListener(func(ctx context.Context, _ characterSectionUpdated) {
			updateUpdateStatus(ctx)
		})
		u.corporationSectionUpdated.AddListener(func(ctx context.Context, _ corporationSectionUpdated) {
			updateUpdateStatus(ctx)
		})
		u.generalSectionUpdated.AddListener(func(ctx context.Context, _ generalSectionUpdated) {
			updateUpdateStatus(ctx)
		})

		ctx := context.Background()
		updateCharacterCount(ctx)
		updateUpdateStatus(ctx)

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
							navItemAbout.Trailing = theme.NewPrimaryThemedResource(icons.Numeric1CircleSvg)
							navItemAbout.Refresh()
						})
					} else {
						hasUpdate.Store(false)
						fyne.Do(func() {
							refreshMoreBadge()
							navItemAbout.Supporting = ""
							navItemAbout.Trailing = nil
							navItemAbout.Refresh()
						})
					}
				}()
				<-tickerNewVersion.C
			}
		}()
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

func makeSearchNav(newCharacterAppBar func(title string, body fyne.CanvasObject, items ...*kxwidget.IconButton) *iwidget.AppBar, u *MobileUI) *iwidget.Navigator {
	searchNav := iwidget.NewNavigator(
		newCharacterAppBar("Search", u.gameSearch),
	)
	return searchNav
}

func makeHomeNav(u *MobileUI) *iwidget.Navigator {
	var homeNav *iwidget.Navigator
	var homeList *iwidget.NavList

	navItemColonies2 := iwidget.NewNavListItemWithIcon(
		"Colonies",
		theme.NewThemedResource(icons.EarthSvg),
		func() {
			homeNav.PushAndHideNavBar(iwidget.NewAppBar("Colonies", u.colonies))
		},
	)
	u.colonies.onUpdate = func(_, expired int) {
		navItemColonies2.Supporting = fmt.Sprintf("%d expired", expired)
		navItemColonies2.Refresh()
	}

	navItemIndustry := iwidget.NewNavListItemWithIcon(
		"Industry",
		theme.NewThemedResource(icons.FactorySvg),
		func() {
			homeNav.Push(iwidget.NewAppBar("Industry",
				container.NewAppTabs(
					container.NewTabItem("Jobs", u.industryJobs),
					container.NewTabItem("Slots", container.NewAppTabs(
						container.NewTabItem("Manufacturing", u.slotsManufacturing),
						container.NewTabItem("Science", u.slotsResearch),
						container.NewTabItem("Reactions", u.slotsReactions),
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

	navItemContracts := iwidget.NewNavListItemWithIcon(
		"Contracts",
		theme.NewThemedResource(icons.FileSignSvg),
		func() {
			homeNav.Push(iwidget.NewAppBar("Contracts", u.contracts))
		},
	)
	u.contracts.OnUpdate = func(count int) {
		var badge string
		if count > 0 {
			badge = fmt.Sprintf("%d contracts active", count)
		}
		navItemContracts.Supporting = badge
		navItemContracts.Refresh()
	}

	navItemWealth := iwidget.NewNavListItemWithIcon(
		"Wealth",
		theme.NewThemedResource(icons.GoldSvg),
		func() {
			w := u.app.NewWindow("Wealth")
			w.SetContent(u.wealth)
			w.Show()
		},
	)
	u.wealth.onUpdate = func(wallet, assets float64) {
		s := fmt.Sprintf("Wallet: %s • Assets: %s", ihumanize.NumberF(wallet, 1), ihumanize.NumberF(assets, 1))
		navItemWealth.Supporting = s
		navItemWealth.Refresh()
	}

	navItemAssets := iwidget.NewNavListItemWithIcon(
		"Assets",
		theme.NewThemedResource(icons.Inventory2Svg),
		func() {
			homeNav.Push(iwidget.NewAppBar("Assets", u.assetSearchAll))
			u.assetSearchAll.focus()
		},
	)

	navItemCharacters := iwidget.NewNavListItemWithIcon(
		"Character Overview",
		theme.NewThemedResource(icons.PortraitSvg),
		func() {
			homeNav.Push(iwidget.NewAppBar("Character Overview", u.characterOverview))
		},
	)
	u.characterOverview.onUpdate = func(characters int) {
		navItemCharacters.Supporting = fmt.Sprintf("%d characters", characters)
		navItemCharacters.Refresh()
	}

	navItemTraining := iwidget.NewNavListItemWithIcon(
		"Training",
		theme.NewThemedResource(icons.SchoolSvg),
		func() {
			homeNav.Push(iwidget.NewAppBar("Training", u.training))
		},
	)
	u.training.onUpdate = func(expired int) {
		navItemTraining.Supporting = fmt.Sprintf("%d expired", expired)
		navItemTraining.Refresh()
	}

	homeList = iwidget.NewNavList(
		navItemCharacters,
		navItemAssets,
		iwidget.NewNavListItemWithIcon(
			"Clones",
			theme.NewThemedResource(icons.HeadSnowflakeSvg),
			func() {
				homeNav.Push(iwidget.NewAppBar("Clones", container.NewAppTabs(
					container.NewTabItem("Augmentations", u.augmentations),
					container.NewTabItem("Jump Clones", u.clones),
				)))
			},
		),
		navItemContracts,
		navItemColonies2,
		navItemIndustry,
		iwidget.NewNavListItemWithIcon(
			"Loyalty Points",
			theme.NewThemedResource(icons.HandHeartSvg),
			func() {
				homeNav.Push(iwidget.NewAppBar("Loyalty Points", u.loyaltyPoints))
			},
		),
		iwidget.NewNavListItemWithIcon(
			"Market Orders",
			theme.NewThemedResource(icons.ChartAreasplineSvg),
			func() {
				homeNav.Push(iwidget.NewAppBar("Market Orders",
					container.NewAppTabs(
						container.NewTabItem("Buy", u.marketOrdersBuy),
						container.NewTabItem("Sell", u.marketOrdersSell),
					),
				))
			},
		),
		navItemTraining,
		navItemWealth,
	)

	homeNav = iwidget.NewNavigator(iwidget.NewAppBar("Home", homeList))
	return homeNav
}
