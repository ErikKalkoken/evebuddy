package ui

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/mobile"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/dustin/go-humanize"

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
		x := make([]fyne.CanvasObject, 0)
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

	const assetsTitle = "Character Assets"
	navItemAssets := iwidget.NewListItemWithIcon(
		assetsTitle,
		theme.NewThemedResource(icons.Inventory2Svg),
		func() {
			u.characterAssets.OnSelected = func() {
				characterNav.PushAndHideNavBar(newCharacterAppBar(assetsTitle, u.characterAssets.LocationAssets))
			}
			characterNav.Push(newCharacterAppBar(assetsTitle, container.NewHScroll(u.characterAssets.Locations)))
		},
	)

	communicationsMenu := fyne.NewMenu("")
	navItemCommunications := iwidget.NewListItemWithIcon(
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
	navItemMail := iwidget.NewListItemWithIcon(
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

	navItemSkills := iwidget.NewListItemWithIcon(
		"Skills",
		theme.NewThemedResource(icons.SchoolSvg),
		func() {
			characterNav.Push(
				newCharacterAppBar(
					"Skills",
					container.NewAppTabs(
						container.NewTabItem("Training", u.characterSkillQueue),
						container.NewTabItem("Catalogue", u.characterSkillCatalogue),
						container.NewTabItem("Ships", u.characterShips),
					),
				))
		},
	)

	navItemWallet := iwidget.NewListItemWithIcon(
		"Wallet",
		theme.NewThemedResource(icons.AttachmoneySvg),
		func() {
			characterNav.Push(
				newCharacterAppBar("Wallet", u.characterWallet))
		},
	)

	characterList := iwidget.NewNavList(
		iwidget.NewListItemWithIcon(
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
		navItemAssets,
		navItemCommunications,
		navItemMail,
		navItemSkills,
		navItemWallet,
	)

	u.characterAssets.OnUpdate = func(s string) {
		fyne.Do(func() {
			navItemAssets.Supporting = "Value: " + s
			characterList.Refresh()
		})
	}

	u.characterMails.onUpdate = func(count int) {
		s := ""
		if count > 0 {
			s = fmt.Sprintf("%s unread", humanize.Comma(int64(count)))
		}
		fyne.Do(func() {
			navItemMail.Supporting = s
			characterList.Refresh()
		})
	}

	u.characterCommunications.OnUpdate = func(count optional.Optional[int]) {
		var s string
		if count.IsEmpty() {
			s = "?"
		} else if count.ValueOrZero() > 0 {
			s = fmt.Sprintf("%s unread", humanize.Comma(int64(count.ValueOrZero())))
		}
		fyne.Do(func() {
			navItemCommunications.Supporting = s
			characterList.Refresh()
		})
	}

	u.characterSkillQueue.OnUpdate = func(_, status string) {
		fyne.Do(func() {
			navItemSkills.Supporting = status
			characterList.Refresh()
		})
	}

	u.characterWallet.onUpdate = func(b string) {
		fyne.Do(func() {
			navItemWallet.Supporting = "Balance: " + b
			characterList.Refresh()
		})
	}

	characterPage := newCharacterAppBar("Characters", characterList)
	characterNav = iwidget.NewNavigatorWithAppBar(characterPage)

	// corporation destination
	fallbackAvatar2, _ := fynetools.MakeAvatar(icons.Corporationplaceholder64Png)
	corpSelector := kxwidget.NewIconButtonWithMenu(fallbackAvatar2, fyne.NewMenu(""))
	newCorpAppBar := func(title string, body fyne.CanvasObject, items ...*kxwidget.IconButton) *iwidget.AppBar {
		items = append(items, corpSelector)
		return iwidget.NewAppBar(title, body, makeAppBarIcons(items...)...)
	}
	var corpNav *iwidget.Navigator
	corpWalletItems := make([]*iwidget.ListItem, 0)
	corporationWalletNavs := make(map[app.Division]*iwidget.ListItem)
	for _, d := range app.Divisions {
		corporationWalletNavs[d] = iwidget.NewListItemWithIcon(
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
	corpWalletNav := iwidget.NewListItemWithIcon(
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
		u.corporationWallets[d].onBalanceUpdate = func(balance string) {
			fyne.Do(func() {
				corporationWalletNavs[d].Supporting = balance
				corpWalletList.Refresh()
			})
		}
		u.corporationWallets[d].onNameUpdate = func(name string) {
			fyne.Do(func() {
				corporationWalletNavs[d].Headline = name
				corpWalletList.Refresh()
			})
		}
	}

	corpContractsNav := iwidget.NewListItemWithIcon(
		"Contracts",
		theme.NewThemedResource(icons.FileSignSvg),
		func() {
			corpNav.Push(newCorpAppBar("Contracts", u.corporationContracts))
		},
	)

	corpIndustryNav := iwidget.NewListItemWithIcon(
		"Industry",
		theme.NewThemedResource(icons.FactorySvg),
		func() {
			corpNav.Push(newCorpAppBar("Industry", u.corporationIndyJobs))
		},
	)

	corpStructuresNav := iwidget.NewListItemWithIcon(
		"Structures",
		theme.NewThemedResource(icons.OfficeBuildingSvg),
		func() {
			corpNav.Push(newCorpAppBar("Structures", u.corporationStructures))
		},
	)

	corpList := iwidget.NewNavList(
		slices.Concat([]*iwidget.ListItem{
			iwidget.NewListItemWithIcon(
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
			),
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
		fyne.Do(func() {
			corpContractsNav.Supporting = badge
			corpList.Refresh()
		})
	}
	u.corporationIndyJobs.OnUpdate = func(count int) {
		var badge string
		if count > 0 {
			badge = fmt.Sprintf("%s jobs ready", ihumanize.Comma(count))
		}
		fyne.Do(func() {
			corpIndustryNav.Supporting = badge
			corpList.Refresh()
		})
	}
	u.corporationStructures.OnUpdate = func(count int) {
		var badge string
		if count > 0 {
			badge = fmt.Sprintf("%s structures reinforced", ihumanize.Comma(count))
		}
		fyne.Do(func() {
			corpStructuresNav.Supporting = badge
			corpList.Refresh()
		})
	}
	u.onUpdateCorporationWalletTotals = func(balance string) {
		fyne.Do(func() {
			corpWalletNav.Supporting = balance
			corpList.Refresh()
		})
	}

	corpPage := newCorpAppBar("Corporations", corpList)
	corpNav = iwidget.NewNavigatorWithAppBar(corpPage)

	// other

	homeNav := makeHomeNav(u)

	searchNav := makeSearchNav(newCharacterAppBar, u)

	// more destination
	var moreNav *iwidget.Navigator
	navItemUpdateStatus := iwidget.NewListItemWithIcon(
		"Update status",
		theme.NewThemedResource(icons.UpdateSvg),
		func() {
			showUpdateStatusWindow(u.baseUI)
		},
	)
	navItemManageCharacters := iwidget.NewListItemWithIcon(
		"Manage characters",
		theme.NewThemedResource(icons.ManageaccountsSvg),
		func() {
			showManageCharactersWindow(u.baseUI)
		},
	)

	navItemAbout := iwidget.NewListItemWithIcon(
		"About",
		theme.InfoIcon(),
		func() {
			moreNav.Push(iwidget.NewAppBar("About", makeAboutPage(u.baseUI)))
		},
	)
	moreList := iwidget.NewNavList(
		iwidget.NewListItemWithIcon(
			"Settings",
			theme.NewThemedResource(icons.CogSvg),
			func() {
				showSettingsWindow(u.baseUI)
			},
		),
		navItemManageCharacters,
		navItemUpdateStatus,
		navItemAbout,
	)
	moreNav = iwidget.NewNavigatorWithAppBar(iwidget.NewAppBar("More", moreList))

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
			if sections.Contains(app.SectionCorporationWalletBalances) {
				corpWalletNav.IsDisabled = false
			} else {
				corpWalletNav.IsDisabled = true
			}
			if sections.Contains(app.SectionCorporationIndustryJobs) {
				corpIndustryNav.IsDisabled = false
			} else {
				corpIndustryNav.IsDisabled = true
			}
			corpList.Refresh()
		})
	}

	u.onUpdateStatus = func() {
		go togglePermittedSections()
		go func() {
			fyne.Do(func() {
				navItemManageCharacters.Supporting = fmt.Sprintf("%d characters", u.scs.ListCharacterIDs().Size())
				moreList.Refresh()
			})
			fyne.Do(func() {
				characterSelector.SetMenuItems(u.makeCharacterSwitchMenu(characterSelector.Refresh))
				corpSelector.SetMenuItems(u.makeCorporationSwitchMenu(corpSelector.Refresh))
			})
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

	u.onUpdateCharacter = func(c *app.Character) {
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
	}
	u.onSetCharacter = func(c *app.Character) {
		go u.updateCharacterAvatar(c.ID, func(r fyne.Resource) {
			fyne.Do(func() {
				characterSelector.SetIcon(r)
			})
		})
		u.characterMails.resetCurrentFolder()
		u.characterCommunications.resetCurrentFolder()
		fyne.Do(func() {
			characterPage.SetTitle(c.EveCharacter.Name)
			characterNav.PopAll()
		})
	}

	u.onSetCorporation = func(c *app.Corporation) {
		go u.updateCorporationAvatar(c.ID, func(r fyne.Resource) {
			fyne.Do(func() {
				corpSelector.SetIcon(r)
			})
		})
		fyne.Do(func() {
			corpPage.SetTitle(c.EveCorporation.Name)
			corpNav.PopAll()
		})
		togglePermittedSections()
	}

	var hasUpdate, hasUpdateError, hasScopeError, isOffline bool
	refreshMoreBadge := func() {
		if hasUpdateError || hasUpdate || hasScopeError || isOffline {
			var importance widget.Importance
			if hasUpdateError {
				importance = widget.DangerImportance
			} else if hasScopeError || isOffline {
				importance = widget.WarningImportance
			} else if hasUpdate {
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

	u.onAppFirstStarted = func() {
		tickerUpdateStatus := time.NewTicker(5 * time.Second)
		go func() {
			for {
				var icon fyne.Resource
				var s string
				if u.ess.IsDailyDowntime() {
					isOffline = true
					icon = theme.NewWarningThemedResource(theme.WarningIcon())
					s = fmt.Sprintf("Off during daily downtime: %s", u.ess.DailyDowntime())
				} else {
					isOffline = false
					status := u.scs.Summary()
					if status.Errors > 0 {
						icon = theme.NewErrorThemedResource(theme.WarningIcon())
						hasUpdateError = true
					} else {
						hasUpdateError = false
					}
					s = status.Display()
				}
				fyne.Do(func() {
					refreshMoreBadge()
					navItemUpdateStatus.Supporting = s
					navItemUpdateStatus.Trailing = icon
					moreList.Refresh()
				})
				<-tickerUpdateStatus.C
			}
		}()
		tickerNewVersion := time.NewTicker(3600 * time.Second)
		go func() {
			for {
				v, err := u.availableUpdate()
				if err != nil {
					slog.Error("fetch github version for menu info", "error", err)
				} else {
					fyne.Do(func() {
						if v.IsRemoteNewer {
							hasUpdate = true
							refreshMoreBadge()
							navItemAbout.Supporting = "Update available"
							navItemAbout.Trailing = theme.NewPrimaryThemedResource(icons.Numeric1CircleSvg)
						} else {
							hasUpdate = false
							refreshMoreBadge()
							navItemAbout.Supporting = ""
							navItemAbout.Trailing = nil
						}
					})
				}
				fyne.Do(func() {
					moreList.Refresh()
				})
				<-tickerNewVersion.C
			}
		}()
	}
	u.onUpdateMissingScope = func(characterCount int) {
		var icon fyne.Resource
		if characterCount > 0 {
			icon = theme.NewWarningThemedResource(theme.WarningIcon())
			hasScopeError = true
		} else {
			icon = nil
			hasScopeError = false
		}
		fyne.Do(func() {
			navItemManageCharacters.Trailing = icon
			moreList.Refresh()
			refreshMoreBadge()
		})
	}

	u.MainWindow().SetContent(navBar)
	return u
}

func makeSearchNav(newCharacterAppBar func(title string, body fyne.CanvasObject, items ...*kxwidget.IconButton) *iwidget.AppBar, u *MobileUI) *iwidget.Navigator {
	searchNav := iwidget.NewNavigatorWithAppBar(
		newCharacterAppBar("Search", u.gameSearch),
	)
	return searchNav
}

func makeHomeNav(u *MobileUI) *iwidget.Navigator {
	var homeNav *iwidget.Navigator
	var homeList *iwidget.List
	navItemColonies2 := iwidget.NewListItemWithIcon(
		"Colonies",
		theme.NewThemedResource(icons.EarthSvg),
		func() {
			homeNav.PushAndHideNavBar(iwidget.NewAppBar("Colonies", u.colonies))
		},
	)
	navItemIndustry := iwidget.NewListItemWithIcon(
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
		fyne.Do(func() {
			navItemIndustry.Supporting = badge
			homeList.Refresh()
		})
	}

	navItemContracts := iwidget.NewListItemWithIcon(
		"Contracts",
		theme.NewThemedResource(icons.FileSignSvg),
		func() {
			homeNav.Push(iwidget.NewAppBar("Contracts", u.contracts))
		},
	)
	navItemWealth := iwidget.NewListItemWithIcon(
		"Wealth",
		theme.NewThemedResource(icons.GoldSvg),
		func() {
			homeNav.Push(iwidget.NewAppBar("Wealth", u.wealth))
		},
	)
	navItemAssets := iwidget.NewListItemWithIcon(
		"Assets",
		theme.NewThemedResource(icons.Inventory2Svg),
		func() {
			homeNav.Push(iwidget.NewAppBar("Assets", u.assets))
			u.assets.focus()
		},
	)
	u.assets.onUpdate = func(total string) {
		fyne.Do(func() {
			navItemAssets.Supporting = fmt.Sprintf("Value: %s", total)
			homeList.Refresh()
		})
	}
	homeList = iwidget.NewNavList(
		iwidget.NewListItemWithIcon(
			"Character Overview",
			theme.NewThemedResource(icons.PortraitSvg),
			func() {
				homeNav.Push(iwidget.NewAppBar("Character Overview", u.characterOverview))
			},
		),
		navItemAssets,
		iwidget.NewListItemWithIcon(
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
		iwidget.NewListItemWithIcon(
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
		iwidget.NewListItemWithIcon(
			"Character Locations",
			theme.NewThemedResource(icons.MapMarkerSvg),
			func() {
				homeNav.Push(iwidget.NewAppBar("Character Locations", u.characterLocations))
			},
		),
		iwidget.NewListItemWithIcon(
			"Training",
			theme.NewThemedResource(icons.SchoolSvg),
			func() {
				homeNav.Push(iwidget.NewAppBar("Training", u.training))
			},
		),
		navItemWealth,
	)
	u.contracts.OnUpdate = func(count int) {
		var badge string
		if count > 0 {
			badge = fmt.Sprintf("%d contracts active", count)
		}
		fyne.Do(func() {
			navItemContracts.Supporting = badge
			homeList.Refresh()
		})
	}

	u.colonies.OnUpdate = func(_, expired int) {
		fyne.Do(func() {
			navItemColonies2.Supporting = fmt.Sprintf("%d expired", expired)
			homeList.Refresh()
		})
	}
	u.wealth.onUpdate = func(wallet, assets float64) {
		fyne.Do(func() {
			navItemWealth.Supporting = fmt.Sprintf(
				"Wallet: %s • Assets: %s",
				ihumanize.Number(wallet, 1),
				ihumanize.Number(assets, 1),
			)
			homeList.Refresh()
		})
	}
	homeNav = iwidget.NewNavigatorWithAppBar(iwidget.NewAppBar("Home", homeList))
	return homeNav
}
