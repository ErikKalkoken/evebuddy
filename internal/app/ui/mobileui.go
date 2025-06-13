package ui

import (
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
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

// NewUI build the UI and returns it.
func NewMobileUI(bu *baseUI) *MobileUI {
	u := &MobileUI{baseUI: bu}

	var navBar *iwidget.NavBar

	// character destination
	fallbackAvatar, _ := fynetools.MakeAvatar(icons.Characterplaceholder64Jpeg)
	characterSelector := kxwidget.NewIconButtonWithMenu(fallbackAvatar, fyne.NewMenu(""))
	newCharacterAppBar := func(title string, body fyne.CanvasObject, items ...*kxwidget.IconButton) *iwidget.AppBar {
		items = append(items, characterSelector)
		return iwidget.NewAppBar(title, body, items...)
	}

	var characterNav *iwidget.Navigator
	mailMenu := fyne.NewMenu("")
	communicationsMenu := fyne.NewMenu("")
	u.characterMail.onSendMessage = func(c *app.Character, mode app.SendMailMode, mail *app.CharacterMail) {
		page := newCharacterSendMail(bu, c, mode, mail)
		if mode != app.SendMailNew {
			characterNav.Pop() // FIXME: Workaround to avoid pushing upon page w/o navbar
		}
		characterNav.PushHideNavBar(
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

	navItemAssets := iwidget.NewListItemWithIcon(
		"Assets",
		theme.NewThemedResource(icons.Inventory2Svg),
		func() {
			u.characterAsset.OnSelected = func() {
				characterNav.Push(newCharacterAppBar("Assets", u.characterAsset.LocationAssets))
			}
			characterNav.Push(newCharacterAppBar("Assets", container.NewHScroll(u.characterAsset.Locations)))
		},
	)
	navItemCommunications := iwidget.NewListItemWithIcon(
		"Communications",
		theme.NewThemedResource(icons.MessageSvg),
		func() {
			u.characterCommunications.OnSelected = func() {
				characterNav.PushHideNavBar(
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
				characterNav.PushHideNavBar(
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
				newCharacterAppBar(
					"Wallet",
					container.NewAppTabs(
						container.NewTabItem("Transactions", u.characterWalletJournal),
						container.NewTabItem("Market Transactions", u.characterWalletTransaction),
					),
				))
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
							container.NewTabItem("Augmentations", u.characterImplants),
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
			navItemAssets.Supporting = s
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

	u.characterWalletJournal.OnUpdate = func(b string) {
		fyne.Do(func() {
			navItemWallet.Supporting = "Balance: " + b
			characterList.Refresh()
		})
	}

	characterPage := newCharacterAppBar("Character", characterList)
	characterNav = iwidget.NewNavigatorWithAppBar(characterPage)

	// characters cross destination
	var homeNav *iwidget.Navigator
	var homeList *iwidget.List
	navItemWealth := iwidget.NewListItemWithIcon(
		"Wealth",
		theme.NewThemedResource(icons.GoldSvg),
		func() {
			homeNav.Push(iwidget.NewAppBar("Wealth", u.wealth))
		},
	)
	navItemColonies2 := iwidget.NewListItemWithIcon(
		"Colonies",
		theme.NewThemedResource(icons.EarthSvg),
		func() {
			homeNav.Push(iwidget.NewAppBar("Colonies", u.colonies))
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
	homeList = iwidget.NewNavList(
		iwidget.NewListItemWithIcon(
			"Characters",
			theme.NewThemedResource(icons.PortraitSvg),
			func() {
				homeNav.Push(iwidget.NewAppBar("Characters", u.characters))
			},
		),
		iwidget.NewListItemWithIcon(
			"Assets",
			theme.NewThemedResource(icons.Inventory2Svg),
			func() {
				homeNav.Push(iwidget.NewAppBar("Assets", u.assets))
				u.assets.focus()
			},
		),
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
			"Locations",
			theme.NewThemedResource(icons.MapMarkerSvg),
			func() {
				homeNav.Push(iwidget.NewAppBar("Locations", u.locations))
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
	homeNav = iwidget.NewNavigatorWithAppBar(iwidget.NewAppBar("Home", homeList))
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
	u.wealth.OnUpdate = func(wallet, assets float64) {
		fyne.Do(func() {
			navItemWealth.Supporting = fmt.Sprintf(
				"Wallet: %s • Assets: %s",
				ihumanize.Number(wallet, 1),
				ihumanize.Number(assets, 1),
			)
			homeList.Refresh()
		})
	}

	// info destination
	searchNav := iwidget.NewNavigatorWithAppBar(
		newCharacterAppBar("Search", u.gameSearch),
	)

	// more destination
	var moreNav *iwidget.Navigator
	makeSettingsMenu := func(actions []settingAction) (fyne.Resource, *fyne.Menu) {
		items := make([]*fyne.MenuItem, 0)
		for _, a := range actions {
			items = append(items, fyne.NewMenuItem(a.Label, a.Action))
		}
		return theme.MoreVerticalIcon(), fyne.NewMenu("", items...)
	}
	navItemUpdateStatus := iwidget.NewListItemWithIcon(
		"Update status",
		theme.NewThemedResource(icons.UpdateSvg),
		func() {
			u.showUpdateStatusWindow()
		},
	)
	navItemManageCharacters := iwidget.NewListItemWithIcon(
		"Manage characters",
		theme.NewThemedResource(icons.ManageaccountsSvg),
		func() {
			moreNav.Push(iwidget.NewAppBar(
				"Manage characters",
				u.manageCharacters,
				kxwidget.NewIconButton(
					theme.NewPrimaryThemedResource(theme.ContentAddIcon()),
					u.manageCharacters.ShowAddCharacterDialog,
				),
			))
		},
	)
	navItemGeneralSettings := iwidget.NewListItem(
		"General",
		func() {
			moreNav.Push(iwidget.NewAppBar(
				"General",
				u.userSettings.GeneralContent,
				kxwidget.NewIconButtonWithMenu(makeSettingsMenu(u.userSettings.GeneralActions)),
			))
		},
	)
	navItemNotificationSettings := iwidget.NewListItem(
		"Notifications",
		func() {
			u.userSettings.OnCommunicationGroupSelected = func(
				title string, content fyne.CanvasObject, actions []settingAction,
			) {
				moreNav.Push(iwidget.NewAppBar(
					title,
					content,
					kxwidget.NewIconButtonWithMenu(makeSettingsMenu(actions)),
				))
			}
			moreNav.Push(iwidget.NewAppBar(
				"Notifications",
				u.userSettings.NotificationSettings,
				kxwidget.NewIconButtonWithMenu(makeSettingsMenu(u.userSettings.NotificationActions)),
			))
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
				moreNav.Push(iwidget.NewAppBar(
					"Settings",
					iwidget.NewNavList(
						navItemGeneralSettings,
						navItemNotificationSettings,
					),
				))
			},
		),
		navItemManageCharacters,
		navItemUpdateStatus,
		navItemAbout,
	)
	u.manageCharacters.OnUpdate = func(characterCount int) {
		fyne.Do(func() {
			navItemManageCharacters.Supporting = fmt.Sprintf("%d characters", characterCount)
			moreList.Refresh()
		})
	}
	moreNav = iwidget.NewNavigatorWithAppBar(iwidget.NewAppBar("More", moreList))

	// navigation bar
	characterDest := iwidget.NewDestinationDef("Character", theme.NewThemedResource(icons.AccountSvg), characterNav)
	characterDest.OnSelectedAgain = func() {
		characterNav.PopAll()
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

	navBar = iwidget.NewNavBar(homeDest, characterDest, searchDest, moreDest)
	characterNav.NavBar = navBar

	u.onUpdateStatus = func() {
		go func() {
			fyne.Do(func() {
				characterSelector.SetMenuItems(u.makeCharacterSwitchMenu(characterSelector.Refresh))
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
				navBar.Select(3)
			} else {
				navBar.Enable(0)
				navBar.Enable(1)
				navBar.Enable(2)
			}
		})
	}
	u.onSetCharacter = func(id int32) {
		go u.updateAvatar(id, func(r fyne.Resource) {
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

	var hasUpdate bool
	var hasError bool
	refreshMoreBadge := func() {
		navBar.SetBadge(3, hasUpdate || hasError)
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
