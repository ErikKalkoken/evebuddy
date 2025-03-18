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
	u.MailArea.OnSendMessage = func(character *app.Character, mode ui.SendMailMode, mail *app.CharacterMail) {
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
			u.MailArea.OnSelected = func() {
				characterNav.PushHideNavBar(
					newCharacterAppBar(
						"Mail",
						u.MailArea.Detail,
						iwidget.NewIconButton(u.MailArea.MakeReplyAction()),
						iwidget.NewIconButton(u.MailArea.MakeReplyAllAction()),
						iwidget.NewIconButton(u.MailArea.MakeForwardAction()),
						iwidget.NewIconButton(u.MailArea.MakeDeleteAction(func() {
							characterNav.Pop()
						})),
					),
				)
			}
			characterNav.Push(
				newCharacterAppBar(
					"Mail",
					u.MailArea.Headers,
					iwidget.NewIconButtonWithMenu(theme.FolderIcon(), mailMenu),
					iwidget.NewIconButton(u.MailArea.MakeComposeMessageAction()),
				))
		},
	)
	navItemCommunications := iwidget.NewListItemWithIcon(
		"Communications",
		theme.NewThemedResource(icons.MessageSvg),
		func() {
			u.NotificationsArea.OnSelected = func() {
				characterNav.PushHideNavBar(
					newCharacterAppBar("Communications", u.NotificationsArea.Detail),
				)
			}
			characterNav.Push(
				newCharacterAppBar(
					"Communications",
					u.NotificationsArea.Notifications,
					iwidget.NewIconButtonWithMenu(theme.FolderIcon(), communicationsMenu),
				),
			)
		},
	)
	navItemAssets := iwidget.NewListItemWithIcon(
		"Assets",
		theme.NewThemedResource(icons.Inventory2Svg),
		func() {
			u.AssetsArea.OnSelected = func() {
				characterNav.Push(newCharacterAppBar("Assets", u.AssetsArea.LocationAssets))
			}
			characterNav.Push(newCharacterAppBar("Assets", container.NewHScroll(u.AssetsArea.Locations)))
		},
	)
	navItemColonies1 := iwidget.NewListItemWithIcon(
		"Colonies",
		theme.NewThemedResource(icons.EarthSvg),
		func() {
			characterNav.Push(newCharacterAppBar("Colonies", u.PlanetArea.Content))
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
							newCharacterAppBar("Training Queue", u.SkillqueueArea.Content),
						),
						iwidget.NewListItemWithNavigator(
							characterNav,
							newCharacterAppBar("Skill Catalogue", u.SkillCatalogueArea.Content),
						),
						iwidget.NewListItemWithNavigator(
							characterNav,
							newCharacterAppBar("Ships", u.ShipsArea.Content),
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
						container.NewTabItem("Transactions", u.WalletJournalArea.Content),
						container.NewTabItem("Market Transactions", u.WalletTransactionArea.Content),
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
						container.NewTabItem("Current Clone", u.ImplantsArea.Content),
						container.NewTabItem("Jump Clones", u.JumpClonesArea.Content),
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
				characterNav.Push(newCharacterAppBar("Contracts", u.ContractsArea.Content))
			},
		),
		navItemSkills,
		navItemWallet,
	)

	u.AssetsArea.OnRedraw = func(s string) {
		navItemAssets.Supporting = s
		characterList.Refresh()
	}

	u.JumpClonesArea.OnReDraw = func(clonesCount int) {
		navItemClones.Supporting = fmt.Sprintf("%d jump clones", clonesCount)
		characterList.Refresh()
	}

	u.PlanetArea.OnRefresh = func(total, expired int) {
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

	u.MailArea.OnRefresh = func(count int) {
		s := ""
		if count > 0 {
			s = fmt.Sprintf("%s unread", humanize.Comma(int64(count)))
		}
		navItemMail.Supporting = s
		characterList.Refresh()
	}

	u.NotificationsArea.OnRefresh = func(count int) {
		s := ""
		if count > 0 {
			s = fmt.Sprintf("%s unread", humanize.Comma(int64(count)))
		}
		navItemCommunications.Supporting = s
		characterList.Refresh()
	}

	u.SkillqueueArea.OnRefresh = func(_, status string) {
		navItemSkills.Supporting = status
		characterList.Refresh()
	}

	u.WalletJournalArea.OnRefresh = func(b string) {
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
			crossNav.Push(iwidget.NewAppBar("Wealth", u.WealthArea.Content))
		},
	)
	navItemColonies2 := iwidget.NewListItemWithIcon(
		"Colonies",
		theme.NewThemedResource(icons.EarthSvg),
		func() {
			crossNav.Push(iwidget.NewAppBar("Colonies", u.ColoniesArea.Content))
		},
	)
	crossList := iwidget.NewNavList(
		iwidget.NewListItemWithIcon(
			"Overview",
			theme.NewThemedResource(icons.AccountMultipleSvg),
			func() {
				crossNav.Push(iwidget.NewAppBar("Overview", u.OverviewArea.Content))
			},
		),
		iwidget.NewListItemWithIcon(
			"Asset Search",
			theme.NewThemedResource(icons.Inventory2Svg),
			func() {
				crossNav.Push(iwidget.NewAppBar("Asset Search", u.AssetSearchArea.Content))
				u.AssetSearchArea.Focus()
			},
		),
		iwidget.NewListItemWithIcon(
			"Locations",
			theme.NewThemedResource(icons.MapMarkerSvg),
			func() {
				crossNav.Push(iwidget.NewAppBar("Locations", u.LocationsArea.Content))
			},
		),
		iwidget.NewListItemWithIcon(
			"Training",
			theme.NewThemedResource(icons.SchoolSvg),
			func() {
				crossNav.Push(iwidget.NewAppBar("Training", u.TrainingArea.Content))
			},
		),
		navItemColonies2,
		navItemWealth,
	)
	crossNav = iwidget.NewNavigator(iwidget.NewAppBar("Characters", crossList))
	u.ColoniesArea.OnRefresh = func(top string) {
		navItemColonies2.Supporting = top
		crossList.Refresh()
	}
	u.WealthArea.OnRefresh = func(total string) {
		navItemWealth.Supporting = total
		crossList.Refresh()
	}

	// info destination
	searchNav := iwidget.NewNavigator(
		newCharacterAppBar("Search", u.SearchArea.Content),
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
				u.AccountArea.Content,
				iwidget.NewIconButton(
					theme.NewPrimaryThemedResource(theme.ContentAddIcon()),
					u.AccountArea.ShowAddCharacterDialog,
				),
			))
		},
	)
	navItemGeneralSettings := iwidget.NewListItem(
		"General",
		func() {
			moreNav.Push(iwidget.NewAppBar(
				"General",
				u.SettingsArea.GeneralContent,
				iwidget.NewIconButtonWithMenu(makeSettingsMenu(u.SettingsArea.GeneralActions)),
			))
		},
	)
	navItemNotificationSettings := iwidget.NewListItem(
		"Notifications",
		func() {
			u.SettingsArea.OnCommunicationGroupSelected = func(
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
				u.SettingsArea.NotificationSettings,
				iwidget.NewIconButtonWithMenu(makeSettingsMenu(u.SettingsArea.NotificationActions)),
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
	u.AccountArea.OnRefresh = func(characterCount int) {
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
		u.SearchArea.Focus()
	}
	searchDest.OnSelectedAgain = func() {
		u.SearchArea.Reset()
	}

	moreDest := iwidget.NewDestinationDef("More", theme.MenuIcon(), moreNav)
	moreDest.OnSelectedAgain = func() {
		moreNav.PopAll()
	}

	navBar = iwidget.NewNavBar(characterDest, crossDest, searchDest, moreDest)
	characterNav.NavBar = navBar

	u.OnRefreshStatus = func() {
		go func() {
			characterSelector.SetMenuItems(u.MakeCharacterSwitchMenu(characterSelector.Refresh))
		}()
	}
	u.OnRefreshCharacter = func(c *app.Character) {
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
		u.MailArea.ResetFolders()
		u.NotificationsArea.ResetGroups()
		characterNav.PopAll()
		navBar.Select(0)
	}

	u.OnAppFirstStarted = func() {
		tickerUpdateStatus := time.NewTicker(5 * time.Second)
		go func() {
			for {
				x := u.StatusCacheService.Summary()
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
	for _, f := range u.MailArea.Folders() {
		s := f.Name
		if f.UnreadCount > 0 {
			s += fmt.Sprintf(" (%d)", f.UnreadCount)
		}
		it := fyne.NewMenuItem(s, func() {
			u.MailArea.SetFolder(f)
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
	for _, f := range u.NotificationsArea.Groups {
		s := f.Name
		if f.UnreadCount > 0 {
			s += fmt.Sprintf(" (%d)", f.UnreadCount)
		}
		it := fyne.NewMenuItem(s, func() {
			u.NotificationsArea.SetGroup(f.Group)
		})
		items2 = append(items2, it)
	}
	return items2
}
