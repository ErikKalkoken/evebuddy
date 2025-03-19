// Package mobile contains the code for rendering the mobile UI.
package mobile

import (
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/fynetools"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type MobileUI struct {
	*ui.BaseUI

	navItemUpdateStatus *iwidget.ListItem
}

// NewUI build the UI and returns it.
func NewMobileUI(bui *ui.BaseUI) *MobileUI {
	u := &MobileUI{BaseUI: bui}

	var navBar *iwidget.NavBar

	// character destination
	fallbackAvatar, _ := fynetools.MakeAvatar(icons.Characterplaceholder64Jpeg)
	characterSelector := iwidget.NewIconButtonWithMenu(fallbackAvatar, fyne.NewMenu(""))
	newCharacterAppBar := func(title string, body fyne.CanvasObject, items ...*iwidget.IconButton) *iwidget.AppBar {
		items = append(items, characterSelector)
		return iwidget.NewAppBar(title, body, items...)
	}

	var characterNav *iwidget.Navigator
	mailMenu := fyne.NewMenu("")
	communicationsMenu := fyne.NewMenu("")
	u.CharacterMail.OnSendMessage = func(character *app.Character, mode ui.SendMailMode, mail *app.CharacterMail) {
		page, sendIcon, sendAction := ui.MakeSendMailPage(bui, character, mode, mail, u.Window)
		if mode != ui.SendMailNew {
			characterNav.Pop() // FIXME: Workaround to avoid pushing upon page w/o navbar
		}
		characterNav.PushHideNavBar(
			newCharacterAppBar(
				"",
				page,
				iwidget.NewIconButton(sendIcon, func() {
					if sendAction() {
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
			u.CharacterMail.OnSelected = func() {
				characterNav.PushHideNavBar(
					newCharacterAppBar(
						"Mail",
						u.CharacterMail.Detail,
						iwidget.NewIconButton(u.CharacterMail.MakeReplyAction()),
						iwidget.NewIconButton(u.CharacterMail.MakeReplyAllAction()),
						iwidget.NewIconButton(u.CharacterMail.MakeForwardAction()),
						iwidget.NewIconButton(u.CharacterMail.MakeDeleteAction(func() {
							characterNav.Pop()
						})),
					),
				)
			}
			characterNav.Push(
				newCharacterAppBar(
					"Mail",
					u.CharacterMail.Headers,
					iwidget.NewIconButtonWithMenu(theme.FolderIcon(), mailMenu),
					iwidget.NewIconButton(u.CharacterMail.MakeComposeMessageAction()),
				))
		},
	)
	navItemCommunications := iwidget.NewListItemWithIcon(
		"Communications",
		theme.NewThemedResource(icons.MessageSvg),
		func() {
			u.CharacterCommunications.OnSelected = func() {
				characterNav.PushHideNavBar(
					newCharacterAppBar("Communications", u.CharacterCommunications.Detail),
				)
			}
			characterNav.Push(
				newCharacterAppBar(
					"Communications",
					u.CharacterCommunications.Notifications,
					iwidget.NewIconButtonWithMenu(theme.FolderIcon(), communicationsMenu),
				),
			)
		},
	)
	navItemAssets := iwidget.NewListItemWithIcon(
		"Assets",
		theme.NewThemedResource(icons.Inventory2Svg),
		func() {
			u.CharacterAssets.OnSelected = func() {
				characterNav.Push(newCharacterAppBar("Assets", u.CharacterAssets.LocationAssets))
			}
			characterNav.Push(newCharacterAppBar("Assets", container.NewHScroll(u.CharacterAssets.Locations)))
		},
	)
	navItemColonies1 := iwidget.NewListItemWithIcon(
		"Colonies",
		theme.NewThemedResource(icons.EarthSvg),
		func() {
			characterNav.Push(newCharacterAppBar("Colonies", u.CharacterPlanets))
		},
	)
	navItemSkills := iwidget.NewListItemWithIcon(
		"Skills",
		theme.NewThemedResource(icons.SchoolSvg),
		func() {
			characterNav.Push(
				newCharacterAppBar(
					"Skills",
					iwidget.NewNavList(
						iwidget.NewListItemWithNavigator(
							characterNav,
							newCharacterAppBar("Training Queue", u.CharacterSkillQueue),
						),
						iwidget.NewListItemWithNavigator(
							characterNav,
							newCharacterAppBar("Skill Catalogue", u.CharacterSkillCatalogue),
						),
						iwidget.NewListItemWithNavigator(
							characterNav,
							newCharacterAppBar("Ships", u.CharacterShips),
						),
						iwidget.NewListItemWithNavigator(
							characterNav,
							newCharacterAppBar("Attributes", u.CharacterAttributes),
						),
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
						container.NewTabItem("Transactions", u.CharacterWalletJournal),
						container.NewTabItem("Market Transactions", u.CharacterWalletTransaction),
					),
				))
		},
	)

	navItemClones := iwidget.NewListItemWithIcon(
		"Clones",
		theme.NewThemedResource(icons.HeadSnowflakeSvg),
		func() {
			characterNav.Push(
				newCharacterAppBar(
					"Clones",
					container.NewAppTabs(
						container.NewTabItem("Current Clone", u.CharacterImplants),
						container.NewTabItem("Jump Clones", u.CharacterJumpClones),
					),
				))
		},
	)
	characterList := iwidget.NewNavList(
		navItemAssets,
		navItemColonies1,
		navItemMail,
		navItemCommunications,
		navItemClones,
		iwidget.NewListItemWithIcon(
			"Contracts",
			theme.NewThemedResource(icons.FileSignSvg),
			func() {
				characterNav.Push(newCharacterAppBar("Contracts", u.CharacterContracts))
			},
		),
		navItemSkills,
		navItemWallet,
	)

	u.CharacterAssets.OnRedraw = func(s string) {
		navItemAssets.Supporting = s
		characterList.Refresh()
	}

	u.CharacterJumpClones.OnReDraw = func(clonesCount int) {
		navItemClones.Supporting = fmt.Sprintf("%d jump clones", clonesCount)
		characterList.Refresh()
	}

	u.CharacterPlanets.OnUpdate = func(total, expired int) {
		var s string
		if total > 0 {
			s = fmt.Sprintf("%d colonies", total)
			if expired > 0 {
				s += fmt.Sprintf(" • %d expired", expired)
			}
		}
		navItemColonies1.Supporting = s
		characterList.Refresh()
	}

	u.CharacterMail.OnUpdate = func(count int) {
		s := ""
		if count > 0 {
			s = fmt.Sprintf("%s unread", humanize.Comma(int64(count)))
		}
		navItemMail.Supporting = s
		characterList.Refresh()
	}

	u.CharacterCommunications.OnUpdate = func(count int) {
		s := ""
		if count > 0 {
			s = fmt.Sprintf("%s unread", humanize.Comma(int64(count)))
		}
		navItemCommunications.Supporting = s
		characterList.Refresh()
	}

	u.CharacterSkillQueue.OnUpdate = func(_, status string) {
		navItemSkills.Supporting = status
		characterList.Refresh()
	}

	u.CharacterWalletJournal.OnUpdate = func(b string) {
		navItemWallet.Supporting = "Balance: " + b
		characterList.Refresh()
	}

	characterPage := newCharacterAppBar("Character", characterList)
	characterNav = iwidget.NewNavigator(characterPage)

	// characters cross destination
	var crossNav *iwidget.Navigator
	navItemWealth := iwidget.NewListItemWithIcon(
		"Wealth",
		theme.NewThemedResource(icons.GoldSvg),
		func() {
			crossNav.Push(iwidget.NewAppBar("Wealth", u.WealthOverview))
		},
	)
	navItemColonies2 := iwidget.NewListItemWithIcon(
		"Colonies",
		theme.NewThemedResource(icons.EarthSvg),
		func() {
			crossNav.Push(iwidget.NewAppBar("Colonies", u.ColonyOverview))
		},
	)
	crossList := iwidget.NewNavList(
		iwidget.NewListItemWithIcon(
			"Overview",
			theme.NewThemedResource(icons.AccountMultipleSvg),
			func() {
				crossNav.Push(iwidget.NewAppBar("Overview", u.CharacterOverview))
			},
		),
		iwidget.NewListItemWithIcon(
			"Asset Search",
			theme.NewThemedResource(icons.Inventory2Svg),
			func() {
				crossNav.Push(iwidget.NewAppBar("Asset Search", u.AllAssetSearch))
				u.AllAssetSearch.Focus()
			},
		),
		iwidget.NewListItemWithIcon(
			"Locations",
			theme.NewThemedResource(icons.MapMarkerSvg),
			func() {
				crossNav.Push(iwidget.NewAppBar("Locations", u.LocationOverview))
			},
		),
		iwidget.NewListItemWithIcon(
			"Training",
			theme.NewThemedResource(icons.SchoolSvg),
			func() {
				crossNav.Push(iwidget.NewAppBar("Training", u.TrainingOverview))
			},
		),
		navItemColonies2,
		navItemWealth,
	)
	crossNav = iwidget.NewNavigator(iwidget.NewAppBar("Characters", crossList))
	u.ColonyOverview.OnUpdate = func(top string) {
		navItemColonies2.Supporting = top
		crossList.Refresh()
	}
	u.WealthOverview.OnUpdate = func(total string) {
		navItemWealth.Supporting = total
		crossList.Refresh()
	}

	// info destination
	searchNav := iwidget.NewNavigator(
		newCharacterAppBar("Search", u.GameSearch),
	)

	// more destination
	var moreNav *iwidget.Navigator
	makeSettingsMenu := func(actions []ui.SettingAction) (fyne.Resource, *fyne.Menu) {
		items := make([]*fyne.MenuItem, 0)
		for _, a := range actions {
			items = append(items, fyne.NewMenuItem(a.Label, a.Action))
		}
		return theme.MoreVerticalIcon(), fyne.NewMenu("", items...)
	}
	u.navItemUpdateStatus = iwidget.NewListItemWithIcon(
		"Update status",
		theme.NewThemedResource(icons.UpdateSvg),
		func() {
			u.ShowUpdateStatusWindow()
		},
	)
	navItemManageCharacters := iwidget.NewListItemWithIcon(
		"Manage characters",
		theme.NewThemedResource(icons.ManageaccountsSvg),
		func() {
			moreNav.Push(iwidget.NewAppBar(
				"Manage characters",
				u.ManagerCharacters,
				iwidget.NewIconButton(
					theme.NewPrimaryThemedResource(theme.ContentAddIcon()),
					u.ManagerCharacters.ShowAddCharacterDialog,
				),
			))
		},
	)
	navItemGeneralSettings := iwidget.NewListItem(
		"General",
		func() {
			moreNav.Push(iwidget.NewAppBar(
				"General",
				u.Settings.GeneralContent,
				iwidget.NewIconButtonWithMenu(makeSettingsMenu(u.Settings.GeneralActions)),
			))
		},
	)
	navItemNotificationSettings := iwidget.NewListItem(
		"Notifications",
		func() {
			u.Settings.OnCommunicationGroupSelected = func(
				title string, content fyne.CanvasObject, actions []ui.SettingAction,
			) {
				moreNav.Push(iwidget.NewAppBar(
					title,
					content,
					iwidget.NewIconButtonWithMenu(makeSettingsMenu(actions)),
				))
			}
			moreNav.Push(iwidget.NewAppBar(
				"Notifications",
				u.Settings.NotificationSettings,
				iwidget.NewIconButtonWithMenu(makeSettingsMenu(u.Settings.NotificationActions)),
			))
		},
	)

	navItemAbout := iwidget.NewListItemWithIcon(
		"About",
		theme.InfoIcon(),
		func() {
			moreNav.Push(iwidget.NewAppBar("About", u.MakeAboutPage()))
		},
	)
	toolsList := iwidget.NewNavList(
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
		u.navItemUpdateStatus,
		navItemAbout,
	)
	u.ManagerCharacters.OnUpdate = func(characterCount int) {
		navItemManageCharacters.Supporting = fmt.Sprintf("%d characters", characterCount)
	}
	moreNav = iwidget.NewNavigator(iwidget.NewAppBar("More", toolsList))

	// navigation bar
	characterDest := iwidget.NewDestinationDef("Character", theme.NewThemedResource(icons.AccountSvg), characterNav)
	characterDest.OnSelectedAgain = func() {
		characterNav.PopAll()
	}

	crossDest := iwidget.NewDestinationDef("Characters", theme.NewThemedResource(icons.AccountMultipleSvg), crossNav)
	crossDest.OnSelectedAgain = func() {
		crossNav.PopAll()
	}

	searchDest := iwidget.NewDestinationDef("Search", theme.SearchIcon(), searchNav)
	searchDest.OnSelected = func() {
		u.GameSearch.Focus()
	}
	searchDest.OnSelectedAgain = func() {
		u.GameSearch.Reset()
	}

	moreDest := iwidget.NewDestinationDef("More", theme.MenuIcon(), moreNav)
	moreDest.OnSelectedAgain = func() {
		moreNav.PopAll()
	}

	navBar = iwidget.NewNavBar(characterDest, crossDest, searchDest, moreDest)
	characterNav.NavBar = navBar

	u.OnUpdateStatus = func() {
		go func() {
			characterSelector.SetMenuItems(u.MakeCharacterSwitchMenu(characterSelector.Refresh))
		}()
	}
	u.OnUpdateCharacter = func(c *app.Character) {
		mailMenu.Items = u.makeMailMenu()
		mailMenu.Refresh()
		communicationsMenu.Items = u.makeCommunicationsMenu()
		communicationsMenu.Refresh()
		if c == nil {
			navBar.Disable(0)
			navBar.Disable(1)
			navBar.Select(2)
		} else {
			navBar.Enable(0)
			navBar.Enable(1)
		}
	}
	u.OnSetCharacter = func(id int32) {
		go u.UpdateAvatar(id, func(r fyne.Resource) {
			characterSelector.SetIcon(r)
		})
		u.CharacterMail.ResetFolders()
		u.CharacterCommunications.ResetGroups()
		characterNav.PopAll()
		navBar.Select(0)
	}

	u.OnAppFirstStarted = func() {
		tickerUpdateStatus := time.NewTicker(5 * time.Second)
		go func() {
			for {
				x := u.StatusCacheService().Summary()
				u.navItemUpdateStatus.Supporting = x.Display()
				toolsList.Refresh()
				<-tickerUpdateStatus.C
			}
		}()
		tickerNewVersion := time.NewTicker(3600 * time.Second)
		go func() {
			for {
				v, err := u.AvailableUpdate()
				if err != nil {
					slog.Error("fetch github version for menu info", "error", err)
				} else {
					if v.IsRemoteNewer {
						navBar.SetBadge(2, true)
						navItemAbout.Supporting = "Update available"
						navItemAbout.Trailing = theme.NewPrimaryThemedResource(icons.Numeric1CircleSvg)
					} else {
						navBar.SetBadge(2, false)
						navItemAbout.Supporting = ""
						navItemAbout.Trailing = nil
					}
				}
				crossList.Refresh()
				<-tickerNewVersion.C
			}
		}()
	}

	u.Window.SetContent(navBar)
	return u
}

func (u *MobileUI) makeMailMenu() []*fyne.MenuItem {
	// current := u.MailArea.CurrentFolder.ValueOrZero()
	items1 := make([]*fyne.MenuItem, 0)
	for _, f := range u.CharacterMail.Folders() {
		s := f.Name
		if f.UnreadCount > 0 {
			s += fmt.Sprintf(" (%d)", f.UnreadCount)
		}
		it := fyne.NewMenuItem(s, func() {
			u.CharacterMail.SetFolder(f)
		})
		// if f == current {
		// 	it.Disabled = true
		// }
		items1 = append(items1, it)
	}
	return items1
}

func (u *MobileUI) makeCommunicationsMenu() []*fyne.MenuItem {
	items2 := make([]*fyne.MenuItem, 0)
	for _, f := range u.CharacterCommunications.Groups {
		s := f.Name
		if f.UnreadCount > 0 {
			s += fmt.Sprintf(" (%d)", f.UnreadCount)
		}
		it := fyne.NewMenuItem(s, func() {
			u.CharacterCommunications.SetGroup(f.Group)
		})
		items2 = append(items2, it)
	}
	return items2
}
