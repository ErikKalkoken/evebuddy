package ui

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
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

	makeAppBarIcons := func(items ...*kxwidget.IconButton) fyne.CanvasObject {
		is := theme.IconInlineSize()
		icons := container.New(layout.NewCustomPaddedHBoxLayout(is))
		for _, ib := range items {
			icons.Add(ib)
		}
		return icons
	}

	// character destination
	fallbackAvatar, _ := fynetools.MakeAvatar(icons.Characterplaceholder64Jpeg)
	characterSelector := kxwidget.NewIconButtonWithMenu(fallbackAvatar, fyne.NewMenu(""))
	newCharacterAppBar := func(title string, body fyne.CanvasObject, items ...*kxwidget.IconButton) *iwidget.AppBar {
		items = append(items, characterSelector)
		return iwidget.NewAppBarWithTrailing(title, body, makeAppBarIcons(items...))
	}

	var characterNav *iwidget.Navigator
	mailMenu := fyne.NewMenu("")
	communicationsMenu := fyne.NewMenu("")
	u.characterMail.onSendMessage = func(c *app.Character, mode app.SendMailMode, mail *app.CharacterMail) {
		page := newCharacterSendMail(bu, c, mode, mail)
		if mode != app.SendMailNew {
			characterNav.Pop() // FIXME: Workaround to avoid pushing upon page w/o navbar
		}
		characterNav.PushAndHideNavBar(
			newCharacterAppBar(
				"",
				page,
				kxwidget.NewIconButton(theme.MailSendIcon(), func() {
					if page.SendAction() {
						characterNav.Pop()
					}
				}),
			),
		)
	}
	const assetsTitle = "Character Assets"
	navItemAssets := iwidget.NewListItemWithIcon(
		assetsTitle,
		theme.NewThemedResource(icons.Inventory2Svg),
		func() {
			u.characterAsset.OnSelected = func() {
				characterNav.PushAndHideNavBar(newCharacterAppBar(assetsTitle, u.characterAsset.LocationAssets))
			}
			characterNav.Push(newCharacterAppBar(assetsTitle, container.NewHScroll(u.characterAsset.Locations)))
		},
	)
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
	navItemMail := iwidget.NewListItemWithIcon(
		"Mail",
		theme.MailComposeIcon(),
		func() {
			u.characterMail.onSelected = func() {
				characterNav.PushAndHideNavBar(
					newCharacterAppBar(
						"Mail",
						u.characterMail.Detail,
						kxwidget.NewIconButton(u.characterMail.MakeReplyAction()),
						kxwidget.NewIconButton(u.characterMail.MakeReplyAllAction()),
						kxwidget.NewIconButton(u.characterMail.MakeForwardAction()),
						kxwidget.NewIconButton(u.characterMail.MakeDeleteAction(func() {
							characterNav.Pop()
						})),
					),
				)
			}
			characterNav.Push(
				newCharacterAppBar(
					"Mail",
					u.characterMail.Headers,
					kxwidget.NewIconButtonWithMenu(theme.FolderIcon(), mailMenu),
					kxwidget.NewIconButton(u.characterMail.makeComposeMessageAction()),
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

	u.characterAsset.OnRedraw = func(s string) {
		fyne.Do(func() {
			navItemAssets.Supporting = "Value: " + s
			characterList.Refresh()
		})
	}

	u.characterMail.onUpdate = func(count int) {
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
		return iwidget.NewAppBarWithTrailing(title, body, makeAppBarIcons(items...))
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
			corpWalletNav,
		})...,
	)
	u.onUpdateCorporationWalletTotals = func(balance string) {
		sections, err := u.rs.PermittedSections(context.Background(), u.currentCorporationID())
		if err != nil {
			slog.Error("Failed to enable corporation tab", "error", err)
			sections.Clear()
			balance = ""
		}
		fyne.Do(func() {
			if sections.Contains(app.SectionCorporationWalletBalances) {
				corpWalletNav.IsDisabled = false
			} else {
				corpWalletNav.IsDisabled = true
			}
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
			moreNav.Push(iwidget.NewAppBar("About", u.makeAboutPage()))
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

	// initial state
	navBar.Disable(0)
	navBar.Disable(1)
	navBar.Disable(2)
	navBar.Disable(3)
	navBar.Select(4)

	u.onUpdateStatus = func() {
		go func() {
			fyne.Do(func() {
				navItemManageCharacters.Supporting = fmt.Sprintf("%d characters", u.scs.ListCharacterIDs().Size())
				moreList.Refresh()
			})
			fyne.Do(func() {
				characterSelector.SetMenuItems(u.makeCharacterSwitchMenu(characterSelector.Refresh))
				corpSelector.SetMenuItems(u.makeCorporationSwitchMenu(corpSelector.Refresh))
			})
		}()
	}
	u.onUpdateCharacter = func(c *app.Character) {
		fyne.Do(func() {
			mailMenu.Items = u.characterMail.makeFolderMenu()
			mailMenu.Refresh()
			communicationsMenu.Items = u.characterCommunications.makeFolderMenu()
			communicationsMenu.Refresh()
			if c == nil {
				navBar.Disable(0)
				navBar.Disable(1)
				navBar.Disable(2)
				navBar.Disable(3)
				navBar.Select(4)
			} else {
				navBar.Enable(0)
				navBar.Enable(1)
				navBar.Enable(2)
				navBar.Enable(3)
			}
		})
	}
	u.onUpdateCorporation = func(corporation *app.Corporation) {
		if corporation == nil {
			fyne.Do(func() {
				navBar.Disable(3)
			})
			return
		}
		fyne.Do(func() {
			navBar.Enable(3)
		})
	}
	u.onSetCharacter = func(id int32) {
		go u.updateCharacterAvatar(id, func(r fyne.Resource) {
			fyne.Do(func() {
				characterSelector.SetIcon(r)
			})
		})
		u.characterMail.resetCurrentFolder()
		u.characterCommunications.resetCurrentFolder()
		fyne.Do(func() {
			characterPage.SetTitle(u.scs.CharacterName(id))
			characterNav.PopAll()
		})
	}
	u.onSetCorporation = func(id int32) {
		go u.updateCorporationAvatar(id, func(r fyne.Resource) {
			fyne.Do(func() {
				corpSelector.SetIcon(r)
			})
		})
		name := u.scs.CorporationName(id)
		fyne.Do(func() {
			corpPage.SetTitle(name)
			corpNav.PopAll()
		})
	}

	var hasUpdate bool
	var hasError bool
	refreshMoreBadge := func() {
		navBar.SetBadge(4, hasUpdate || hasError)
	}

	u.onAppFirstStarted = func() {
		tickerUpdateStatus := time.NewTicker(5 * time.Second)
		go func() {
			for {
				var icon fyne.Resource
				status := u.scs.Summary()
				if status.Errors > 0 {
					icon = theme.WarningIcon()
					hasError = true
				} else {
					icon = nil
					hasError = false
				}
				fyne.Do(func() {
					refreshMoreBadge()
					navItemUpdateStatus.Supporting = status.Display()
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
				homeNav.Push(iwidget.NewAppBar("Clones", u.clones))
			},
		),
		navItemContracts,
		navItemColonies2,
		navItemIndustry,
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
		s := "Active"
		if count > 0 {
			s += fmt.Sprintf(" (%d)", count)
		}
		fyne.Do(func() {
			navItemContracts.Supporting = s
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
