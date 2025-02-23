// Package mobile contains the code for rendering the mobile UI.
package mobile

import (
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icon"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/dustin/go-humanize"
)

type MobileUI struct {
	*ui.BaseUI

	navItemUpdateStatus *iwidget.ListItem
}

// NewUI build the UI and returns it.
func NewMobileUI(bui *ui.BaseUI) *MobileUI {
	u := &MobileUI{BaseUI: bui}
	showItemWindow := func(iw *ui.ItemInfoArea, err error) {
		if err != nil {
			t := "Failed to show item info"
			slog.Error(t, "err", err)
			d := ui.NewErrorDialog(t, err, u.Window)
			d.Show()
			return
		}
		w := u.FyneApp.NewWindow("Information")
		w.SetContent(iw.Content)
		w.Show()
	}
	u.ShowTypeInfoWindow = func(typeID, characterID int32, selectTab ui.TypeWindowTab) {
		showItemWindow(u.NewItemInfoArea(typeID, characterID, 0, selectTab))
	}
	u.ShowLocationInfoWindow = func(locationID int64) {
		showItemWindow(u.NewItemInfoArea(0, 0, locationID, ui.DescriptionTab))
	}

	var navBar *iwidget.NavBar

	// character destination
	fallbackAvatar, _ := iwidget.MakeAvatar(icon.Characterplaceholder64Jpeg)
	characterSelector := iwidget.NewIconButton(fallbackAvatar, nil)
	characterSelector.OnTapped = func() {
		o := characterSelector
		characterID := u.CharacterID()
		cc := u.StatusCacheService.ListCharacters()
		items := make([]*fyne.MenuItem, 0)
		if len(cc) == 0 {
			it := fyne.NewMenuItem("No characters", nil)
			it.Disabled = true
			items = append(items, it)
		} else {
			for _, c := range cc {
				it := fyne.NewMenuItem(c.Name, func() {
					u.LoadCharacter(c.ID)
				})
				if c.ID == characterID {
					it.Disabled = true
				}
				items = append(items, it)
			}
		}
		iwidget.ShowContextMenu(o, fyne.NewMenu("", items...))
	}

	newCharacterAppBar := func(title string, body fyne.CanvasObject, items ...*iwidget.IconButton) *iwidget.AppBar {
		items = append(items, characterSelector)
		return iwidget.NewAppBar(title, body, items...)
	}

	var characterNav *iwidget.Navigator
	mailMenu := fyne.NewMenu("")
	communicationsMenu := fyne.NewMenu("")
	u.MailArea.OnSendMessage = func(character *app.Character, mode ui.SendMailMode, mail *app.CharacterMail) {
		page, sendIcon, sendAction := u.MakeSendMailPage(character, mode, mail, u.Window)
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
		theme.NewThemedResource(icon.MessageSvg),
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
		theme.NewThemedResource(icon.Inventory2Svg),
		func() {
			u.AssetsArea.OnSelected = func() {
				characterNav.Push(newCharacterAppBar("Assets", u.AssetsArea.LocationAssets))
			}
			characterNav.Push(newCharacterAppBar("Assets", container.NewHScroll(u.AssetsArea.Locations)))
		},
	)
	navItemColonies1 := iwidget.NewListItemWithIcon(
		"Colonies",
		theme.NewThemedResource(icon.EarthSvg),
		func() {
			characterNav.Push(newCharacterAppBar("Colonies", u.PlanetArea.Content))
		},
	)
	navItemSkills := iwidget.NewListItemWithIcon(
		"Skills",
		theme.NewThemedResource(icon.SchoolSvg),
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
		theme.NewThemedResource(icon.AttachmoneySvg),
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
		theme.NewThemedResource(icon.HeadSnowflakeSvg),
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
			theme.NewThemedResource(icon.FileSignSvg),
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
		theme.NewThemedResource(icon.GoldSvg),
		func() {
			crossNav.Push(iwidget.NewAppBar("Wealth", u.WealthArea.Content))
		},
	)
	navItemColonies2 := iwidget.NewListItemWithIcon(
		"Colonies",
		theme.NewThemedResource(icon.EarthSvg),
		func() {
			crossNav.Push(iwidget.NewAppBar("Colonies", u.ColoniesArea.Content))
		},
	)
	crossList := iwidget.NewNavList(
		iwidget.NewListItemWithIcon(
			"Overview",
			theme.NewThemedResource(icon.AccountMultipleSvg),
			func() {
				crossNav.Push(iwidget.NewAppBar("Overview", u.OverviewArea.Content))
			},
		),
		iwidget.NewListItemWithIcon(
			"Asset Search",
			theme.NewThemedResource(icon.Inventory2Svg),
			func() {
				crossNav.Push(iwidget.NewAppBar("Asset Search", u.AssetSearchArea.Content))
				u.AssetSearchArea.Focus()
			},
		),
		iwidget.NewListItemWithIcon(
			"Locations",
			theme.NewThemedResource(icon.MapMarkerSvg),
			func() {
				crossNav.Push(iwidget.NewAppBar("Locations", u.LocationsArea.Content))
			},
		),
		iwidget.NewListItemWithIcon(
			"Training",
			theme.NewThemedResource(icon.SchoolSvg),
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
		theme.NewThemedResource(icon.UpdateSvg),
		func() {
			u.ShowUpdateStatusWindow()
		},
	)
	navItemManageCharacters := iwidget.NewListItemWithIcon(
		"Manage characters",
		theme.NewThemedResource(icon.ManageaccountsSvg),
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
				iwidget.NewIconButtonWithMenu(makeSettingsMenu(u.SettingsArea.NotificationActions)),
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

	toolsList := iwidget.NewNavList(
		iwidget.NewListItemWithIcon(
			"Settings",
			theme.NewThemedResource(icon.CogSvg),
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
		iwidget.NewListItemWithIcon(
			"About",
			theme.NewThemedResource(icon.InformationSvg),
			func() {
				moreNav.Push(iwidget.NewAppBar("About", u.MakeAboutPage()))
			},
		),
	)
	u.AccountArea.OnRefresh = func(characterCount int) {
		navItemManageCharacters.Supporting = fmt.Sprintf("%d characters", characterCount)
	}
	moreNav = iwidget.NewNavigator(iwidget.NewAppBar("More", toolsList))

	// navigation bar
	characterDest := iwidget.NewNavBarItem("Character", theme.NewThemedResource(icon.AccountSvg), characterNav)
	characterDest.OnSelectedAgain = func() {
		characterNav.PopAll()
	}
	crossDest := iwidget.NewNavBarItem("Characters", theme.NewThemedResource(icon.AccountMultipleSvg), crossNav)
	crossDest.OnSelectedAgain = func() {
		crossNav.PopAll()
	}
	moreDest := iwidget.NewNavBarItem("More", theme.MenuIcon(), moreNav)
	moreDest.OnSelectedAgain = func() {
		moreNav.PopAll()
	}
	navBar = iwidget.NewNavBar(characterDest, crossDest, moreDest)
	characterNav.NavBar = navBar

	u.OnSetCharacter = func(id int32) {
		// update character selector
		go u.UpdateAvatar(id, func(r fyne.Resource) {
			characterSelector.SetIcon(r)
		})
		// init mail
		u.MailArea.ResetFolders()
		mailMenu.Items = u.makeMailMenu()
		mailMenu.Refresh()

		// init communications
		u.NotificationsArea.ResetGroups()
		communicationsMenu.Items = u.makeCommunicationsMenu()
		communicationsMenu.Refresh()

		characterNav.PopAll()
	}

	u.OnAppStarted = func() {
		ticker := time.NewTicker(2 * time.Second)
		go func() {
			for {
				x := u.StatusCacheService.Summary()
				u.navItemUpdateStatus.Supporting = x.Display()
				toolsList.Refresh()
				<-ticker.C
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
