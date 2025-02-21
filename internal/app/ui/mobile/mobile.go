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
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/dustin/go-humanize"
)

type MobileUI struct {
	*ui.BaseUI

	navItemUpdateStatus *widget.ListItem
}

// NewUI build the UI and returns it.
func NewMobileUI(fyneApp fyne.App) *MobileUI {
	u := &MobileUI{}
	u.BaseUI = ui.NewBaseUI(fyneApp)
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

	var navBar *widget.NavBar

	// character destination
	fallbackAvatar, _ := ui.MakeAvatar(ui.IconCharacterplaceholder64Jpeg)
	characterSelector := widget.NewIconButton(fallbackAvatar, nil)
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
		widget.ShowContextMenu(o, fyne.NewMenu("", items...))
	}

	newCharacterAppBar := func(title string, body fyne.CanvasObject, items ...*widget.IconButton) *widget.AppBar {
		items = append(items, characterSelector)
		return widget.NewAppBar(title, body, items...)
	}

	var characterNav *widget.Navigator
	mailMenu := fyne.NewMenu("")
	communicationsMenu := fyne.NewMenu("")
	u.MailArea.OnSendMessage = func(character *app.Character, mode ui.SendMailMode, mail *app.CharacterMail) {
		page, sendIcon, sendAction := u.MakeSendMailPage(character, mode, mail, u.Window)
		if mode != ui.SendMailNew {
			characterNav.Pop() // FIXME: Workaround to avoid pushing upon page w/o navbar
		}
		characterNav.PushNoNavBar(
			newCharacterAppBar(
				"",
				page,
				widget.NewIconButton(sendIcon, func() {
					if sendAction() {
						characterNav.Pop()
					}
				}),
			),
			navBar,
		)
	}
	navItemMail := widget.NewListItemWithIcon(
		"Mail",
		theme.MailComposeIcon(),
		func() {
			u.MailArea.OnSelected = func() {
				characterNav.PushNoNavBar(
					newCharacterAppBar(
						"Mail",
						u.MailArea.Detail,
						widget.NewIconButton(u.MailArea.MakeReplyAction()),
						widget.NewIconButton(u.MailArea.MakeReplyAllAction()),
						widget.NewIconButton(u.MailArea.MakeForwardAction()),
						widget.NewIconButton(u.MailArea.MakeDeleteAction(func() {
							characterNav.Pop()
						})),
					),
					navBar,
				)
			}
			characterNav.Push(
				newCharacterAppBar(
					"Mail",
					u.MailArea.Headers,
					widget.NewIconButtonWithMenu(theme.FolderIcon(), mailMenu),
					widget.NewIconButton(u.MailArea.MakeComposeMessageAction()),
				))
		},
	)
	navItemCommunications := widget.NewListItemWithIcon(
		"Communications",
		theme.NewThemedResource(ui.IconMessageSvg),
		func() {
			u.NotificationsArea.OnSelected = func() {
				characterNav.PushNoNavBar(
					newCharacterAppBar("Communications", u.NotificationsArea.Detail),
					navBar,
				)
			}
			characterNav.Push(
				newCharacterAppBar(
					"Communications",
					u.NotificationsArea.Notifications,
					widget.NewIconButtonWithMenu(theme.FolderIcon(), communicationsMenu),
				),
			)
		},
	)
	navItemAssets := widget.NewListItemWithIcon(
		"Assets",
		theme.NewThemedResource(ui.IconInventory2Svg),
		func() {
			u.AssetsArea.OnSelected = func() {
				characterNav.Push(newCharacterAppBar("Assets", u.AssetsArea.LocationAssets))
			}
			characterNav.Push(newCharacterAppBar("Assets", container.NewHScroll(u.AssetsArea.Locations)))
		},
	)
	navItemColonies1 := widget.NewListItemWithIcon(
		"Colonies",
		theme.NewThemedResource(ui.IconEarthSvg),
		func() {
			characterNav.Push(newCharacterAppBar("Colonies", u.PlanetArea.Content))
		},
	)
	navItemSkills := widget.NewListItemWithIcon(
		"Skills",
		theme.NewThemedResource(ui.IconSchoolSvg),
		func() {
			characterNav.Push(
				newCharacterAppBar(
					"Skills",
					widget.NewNavList(
						widget.NewListItemWithNavigator(
							characterNav,
							newCharacterAppBar("Training Queue", u.SkillqueueArea.Content),
						),
						widget.NewListItemWithNavigator(
							characterNav,
							newCharacterAppBar("Skill Catalogue", u.SkillCatalogueArea.Content),
						),
						widget.NewListItemWithNavigator(
							characterNav,
							newCharacterAppBar("Ships", u.ShipsArea.Content),
						),
					),
				))
		},
	)
	navItemWallet := widget.NewListItemWithIcon(
		"Wallet",
		theme.NewThemedResource(ui.IconAttachmoneySvg),
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

	navItemClones := widget.NewListItemWithIcon(
		"Clones",
		theme.NewThemedResource(ui.IconHeadSnowflakeSvg),
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
	characterList := widget.NewNavList(
		navItemAssets,
		navItemColonies1,
		navItemMail,
		navItemCommunications,
		navItemClones,
		widget.NewListItemWithIcon(
			"Contracts",
			theme.NewThemedResource(ui.IconFileSignSvg),
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
	characterNav = widget.NewNavigator(characterPage)

	// characters cross destination
	var crossNav *widget.Navigator
	navItemWealth := widget.NewListItemWithIcon(
		"Wealth",
		theme.NewThemedResource(ui.IconGoldSvg),
		func() {
			crossNav.Push(widget.NewAppBar("Wealth", u.WealthArea.Content))
		},
	)
	navItemColonies2 := widget.NewListItemWithIcon(
		"Colonies",
		theme.NewThemedResource(ui.IconEarthSvg),
		func() {
			crossNav.Push(widget.NewAppBar("Colonies", u.ColoniesArea.Content))
		},
	)
	crossList := widget.NewNavList(
		widget.NewListItemWithIcon(
			"Overview",
			theme.NewThemedResource(ui.IconAccountMultipleSvg),
			func() {
				crossNav.Push(widget.NewAppBar("Overview", u.OverviewArea.Content))
			},
		),
		widget.NewListItemWithIcon(
			"Asset Search",
			theme.NewThemedResource(ui.IconInventory2Svg),
			func() {
				crossNav.Push(widget.NewAppBar("Asset Search", u.AssetSearchArea.Content))
				u.AssetSearchArea.Focus()
			},
		),
		widget.NewListItemWithIcon(
			"Locations",
			theme.NewThemedResource(ui.IconMapMarkerSvg),
			func() {
				crossNav.Push(widget.NewAppBar("Locations", u.LocationsArea.Content))
			},
		),
		widget.NewListItemWithIcon(
			"Training",
			theme.NewThemedResource(ui.IconSchoolSvg),
			func() {
				crossNav.Push(widget.NewAppBar("Training", u.TrainingArea.Content))
			},
		),
		navItemColonies2,
		navItemWealth,
	)
	crossNav = widget.NewNavigator(widget.NewAppBar("Characters", crossList))
	u.ColoniesArea.OnRefresh = func(top string) {
		navItemColonies2.Supporting = top
		crossList.Refresh()
	}
	u.WealthArea.OnRefresh = func(total string) {
		navItemWealth.Supporting = total
		crossList.Refresh()
	}

	// more destination
	var moreNav *widget.Navigator
	makeSettingsMenu := func(actions []ui.SettingAction) (fyne.Resource, *fyne.Menu) {
		items := make([]*fyne.MenuItem, 0)
		for _, a := range actions {
			items = append(items, fyne.NewMenuItem(a.Label, a.Action))
		}
		return theme.MoreVerticalIcon(), fyne.NewMenu("", items...)
	}
	u.navItemUpdateStatus = widget.NewListItemWithIcon(
		"Update status",
		theme.NewThemedResource(ui.IconUpdateSvg),
		func() {
			u.ShowUpdateStatusWindow()
		},
	)
	navItemManageCharacters := widget.NewListItemWithIcon(
		"Manage characters",
		theme.NewThemedResource(ui.IconManageaccountsSvg),
		func() {
			moreNav.Push(widget.NewAppBar(
				"Manage characters",
				u.AccountArea.Content,
				widget.NewIconButton(
					theme.NewPrimaryThemedResource(theme.ContentAddIcon()),
					u.AccountArea.ShowAddCharacterDialog,
				),
			))
		},
	)
	navItemGeneralSettings := widget.NewListItem(
		"General",
		func() {
			moreNav.Push(widget.NewAppBar(
				"General",
				u.SettingsArea.GeneralContent,
				widget.NewIconButtonWithMenu(makeSettingsMenu(u.SettingsArea.NotificationActions)),
			))
		},
	)
	navItemNotificationSettings := widget.NewListItem(
		"Notifications",
		func() {
			u.SettingsArea.OnCommunicationGroupSelected = func(
				title string, content fyne.CanvasObject, actions []ui.SettingAction,
			) {
				moreNav.Push(widget.NewAppBar(
					title,
					content,
					widget.NewIconButtonWithMenu(makeSettingsMenu(actions)),
				))
			}
			moreNav.Push(widget.NewAppBar(
				"Notifications",
				u.SettingsArea.NotificationSettings,
				widget.NewIconButtonWithMenu(makeSettingsMenu(u.SettingsArea.NotificationActions)),
			))
		},
	)

	toolsList := widget.NewNavList(
		widget.NewListItemWithIcon(
			"Settings",
			theme.NewThemedResource(ui.IconCogSvg),
			func() {
				moreNav.Push(widget.NewAppBar(
					"Settings",
					widget.NewNavList(
						navItemGeneralSettings,
						navItemNotificationSettings,
					),
				))
			},
		),
		navItemManageCharacters,
		widget.NewListItemWithIcon(
			"About",
			theme.NewThemedResource(ui.IconInformationSvg),
			func() {
				moreNav.Push(widget.NewAppBar("About", u.MakeAboutPage()))
			},
		),
		u.navItemUpdateStatus,
	)
	u.AccountArea.OnRefresh = func(characterCount int) {
		navItemManageCharacters.Supporting = fmt.Sprintf("%d characters", characterCount)
	}
	moreNav = widget.NewNavigator(widget.NewAppBar("More", toolsList))

	// navigation bar
	characterDest := widget.NewNavBarItem("Character", theme.NewThemedResource(ui.IconAccountSvg), characterNav)
	characterDest.OnSelectedAgain = func() {
		characterNav.PopAll()
	}
	crossDest := widget.NewNavBarItem("Characters", theme.NewThemedResource(ui.IconAccountMultipleSvg), crossNav)
	crossDest.OnSelectedAgain = func() {
		crossNav.PopAll()
	}
	moreDest := widget.NewNavBarItem("More", theme.MenuIcon(), moreNav)
	moreDest.OnSelectedAgain = func() {
		moreNav.PopAll()
	}
	navBar = widget.NewNavBar(characterDest, crossDest, moreDest)

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
