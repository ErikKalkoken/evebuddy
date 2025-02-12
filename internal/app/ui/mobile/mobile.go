// Package mobile contains the code for rendering the mobile UI.
package mobile

import (
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/app/widgets"
	"github.com/dustin/go-humanize"
)

type MobileUI struct {
	*ui.BaseUI

	navItemUpdateStatus *widgets.ListItem
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

	u.MailArea.SendMessage = func(_ ui.SendMessageMode, _ *app.CharacterMail) {
		d := dialog.NewInformation("Send Message", "PLACEHOLDER", u.Window)
		d.Show()
	}

	// character
	fallbackAvatar, _ := ui.MakeAvatar(ui.IconCharacterplaceholder64Jpeg)
	characterSelector := widgets.NewIconButton(fallbackAvatar, nil)
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
		widgets.ShowContextMenu(o, fyne.NewMenu("", items...))
	}

	newCharacterAppBar := func(title string, body fyne.CanvasObject, items ...*widgets.IconButton) *widgets.AppBar {
		items = append(items, characterSelector)
		return widgets.NewAppBar(title, body, items...)
	}

	var characterNav *widgets.Navigator
	mailMenu := fyne.NewMenu("")
	communicationsMenu := fyne.NewMenu("")

	navItemMail := widgets.NewListItemWithIcon(
		"Mail",
		theme.MailComposeIcon(),
		func() {
			deleteAction := widgets.NewIconButton(theme.DeleteIcon(), u.MailArea.MakeDeleteAction(func() {
				characterNav.Pop()
			}))
			u.MailArea.OnSelected = func() {
				characterNav.Push(
					widgets.NewAppBar("", u.MailArea.Detail, deleteAction),
				)
			}
			characterNav.Push(
				newCharacterAppBar(
					"Mail",
					u.MailArea.Headers,
					widgets.NewIconButtonWithMenu(theme.FolderIcon(), mailMenu),
				))
		},
	)

	navItemCommunications := widgets.NewListItemWithIcon(
		"Communications",
		theme.NewThemedResource(ui.IconMessageSvg),
		func() {
			u.NotificationsArea.OnSelected = func() {
				characterNav.Push(
					widgets.NewAppBar("", u.NotificationsArea.Detail),
				)
			}
			characterNav.Push(
				newCharacterAppBar(
					"Communications",
					u.NotificationsArea.Notifications,
					widgets.NewIconButtonWithMenu(theme.FolderIcon(), communicationsMenu),
				),
			)
		},
	)
	navItemAssets := widgets.NewListItemWithIcon(
		"Assets",
		theme.NewThemedResource(ui.IconInventory2Svg),
		func() {
			u.AssetsArea.OnSelected = func() {
				characterNav.Push(newCharacterAppBar("", u.AssetsArea.LocationAssets))
			}
			characterNav.Push(newCharacterAppBar("Assets", container.NewHScroll(u.AssetsArea.Locations)))
		},
	)
	navItemColonies1 := widgets.NewListItemWithIcon(
		"Colonies",
		theme.NewThemedResource(ui.IconEarthSvg),
		func() {
			characterNav.Push(newCharacterAppBar("Colonies", u.PlanetArea.Content))
		},
	)
	navItemSkills := widgets.NewListItemWithIcon(
		"Skills",
		theme.NewThemedResource(ui.IconSchoolSvg),
		func() {
			characterNav.Push(
				newCharacterAppBar(
					"Skills",
					widgets.NewNavList(
						widgets.NewListItemWithNavigator(
							characterNav,
							newCharacterAppBar("Training Queue", u.SkillqueueArea.Content),
						),
						widgets.NewListItemWithNavigator(
							characterNav,
							newCharacterAppBar("Skill Catalogue", u.SkillCatalogueArea.Content),
						),
						widgets.NewListItemWithNavigator(
							characterNav,
							newCharacterAppBar("Ships", u.ShipsArea.Content),
						),
					),
				))
		},
	)
	navItemWallet := widgets.NewListItemWithIcon(
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

	navItemClones := widgets.NewListItemWithIcon(
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
	characterList := widgets.NewNavList(
		navItemAssets,
		navItemColonies1,
		navItemMail,
		navItemCommunications,
		navItemClones,
		widgets.NewListItemWithIcon(
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
	characterNav = widgets.NewNavigator(characterPage)

	// characters cross
	var crossNav *widgets.Navigator
	navItemWealth := widgets.NewListItemWithIcon(
		"Wealth",
		theme.NewThemedResource(ui.IconGoldSvg),
		func() {
			crossNav.Push(widgets.NewAppBar("Wealth", u.WealthArea.Content))
		},
	)
	navItemColonies2 := widgets.NewListItemWithIcon(
		"Colonies",
		theme.NewThemedResource(ui.IconEarthSvg),
		func() {
			crossNav.Push(widgets.NewAppBar("Colonies", u.ColoniesArea.Content))
		},
	)
	crossList := widgets.NewNavList(
		widgets.NewListItemWithIcon(
			"Overview",
			theme.NewThemedResource(ui.IconAccountMultipleSvg),
			func() {
				crossNav.Push(widgets.NewAppBar("Overview", u.OverviewArea.Content))
			},
		),
		widgets.NewListItemWithIcon(
			"Asset Search",
			theme.NewThemedResource(ui.IconInventory2Svg),
			func() {
				crossNav.Push(widgets.NewAppBar("Asset Search", u.AssetSearchArea.Content))
			},
		),
		widgets.NewListItemWithIcon(
			"Locations",
			theme.NewThemedResource(ui.IconMapMarkerSvg),
			func() {
				crossNav.Push(widgets.NewAppBar("Locations", u.LocationsArea.Content))
			},
		),
		widgets.NewListItemWithIcon(
			"Training",
			theme.NewThemedResource(ui.IconSchoolSvg),
			func() {
				crossNav.Push(widgets.NewAppBar("Training", u.TrainingArea.Content))
			},
		),
		navItemColonies2,
		navItemWealth,
	)
	crossNav = widgets.NewNavigator(widgets.NewAppBar("Characters", crossList))
	u.ColoniesArea.OnRefresh = func(top string) {
		navItemColonies2.Supporting = top
		crossList.Refresh()
	}
	u.WealthArea.OnRefresh = func(total string) {
		navItemWealth.Supporting = total
		crossList.Refresh()
	}

	// tools
	var moreNav *widgets.Navigator
	makePage := func(c fyne.CanvasObject) fyne.CanvasObject {
		return container.NewScroll(c)
	}
	makeMenu := func(items ...*fyne.MenuItem) (fyne.Resource, *fyne.Menu) {
		return theme.MenuExpandIcon(), fyne.NewMenu("", items...)
	}
	u.navItemUpdateStatus = widgets.NewListItemWithIcon(
		"Update status",
		theme.NewThemedResource(ui.IconUpdateSvg),
		func() {
			u.ShowUpdateStatusWindow()
		},
	)
	navItemManageCharacters := widgets.NewListItemWithIcon(
		"Manage characters",
		theme.NewThemedResource(ui.IconManageaccountsSvg),
		func() {
			moreNav.Push(widgets.NewAppBar(
				"Manage characters",
				u.AccountArea.Content,
				widgets.NewIconButton(
					theme.NewPrimaryThemedResource(theme.ContentAddIcon()),
					u.AccountArea.ShowAddCharacterDialog,
				),
			))
		},
	)
	toolsList := widgets.NewNavList(
		widgets.NewListItemWithIcon(
			"Settings",
			theme.NewThemedResource(ui.IconCogSvg),
			func() {
				moreNav.Push(
					widgets.NewAppBar(
						"Settings",
						widgets.NewNavList(
							widgets.NewListItem(
								"General",
								func() {
									c, f := u.MakeGeneralSettingsPage(nil)
									moreNav.Push(
										widgets.NewAppBar("General", makePage(c), widgets.NewIconButtonWithMenu(
											makeMenu(fyne.NewMenuItem(
												"Reset", f,
											)))),
									)
								},
							),
							widgets.NewListItem(
								"Eve Online",
								func() {
									c, f := u.MakeEVEOnlinePage()
									moreNav.Push(
										widgets.NewAppBar("Eve Online", makePage(c), widgets.NewIconButtonWithMenu(
											makeMenu(fyne.NewMenuItem(
												"Reset", f,
											)))),
									)
								},
							),
							widgets.NewListItem(
								"Notifications",
								func() {
									c, f := u.MakeNotificationGeneralPage(nil)
									moreNav.Push(
										widgets.NewAppBar("Notification - General", makePage(c), widgets.NewIconButtonWithMenu(
											makeMenu(fyne.NewMenuItem(
												"Reset", f,
											)))),
									)
								},
							),
							widgets.NewListItem(
								"Notification - Types",
								func() {
									c, f := u.MakeNotificationTypesPage()
									moreNav.Push(
										widgets.NewAppBar("Notification - Types", makePage(c), widgets.NewIconButtonWithMenu(
											makeMenu(fyne.NewMenuItem(
												"Reset", f,
											)))),
									)
								},
							),
						),
					))
			},
		),
		navItemManageCharacters,
		widgets.NewListItemWithIcon(
			"About",
			theme.NewThemedResource(ui.IconInformationSvg),
			func() {
				moreNav.Push(widgets.NewAppBar("About", u.MakeAboutPage()))
			},
		),
		u.navItemUpdateStatus,
	)
	u.AccountArea.OnRefresh = func(characterCount int) {
		navItemManageCharacters.Supporting = fmt.Sprintf("%d characters", characterCount)
	}
	moreNav = widgets.NewNavigator(widgets.NewAppBar("More", toolsList))

	// navigation bar
	characterDest := widgets.NewNavBarItem("Character", theme.NewThemedResource(ui.IconAccountSvg), characterNav)
	characterDest.OnSelectedAgain = func() {
		characterNav.PopAll()
	}
	crossDest := widgets.NewNavBarItem("Characters", theme.NewThemedResource(ui.IconAccountMultipleSvg), crossNav)
	crossDest.OnSelectedAgain = func() {
		crossNav.PopAll()
	}
	moreDest := widgets.NewNavBarItem("More", theme.MenuIcon(), moreNav)
	moreDest.OnSelectedAgain = func() {
		moreNav.PopAll()
	}
	navBar := widgets.NewNavBar(characterDest, crossDest, moreDest)

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
		u.NotificationsArea.ResetFolders()
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
	for _, f := range u.NotificationsArea.Folders {
		s := f.Name
		if f.UnreadCount > 0 {
			s += fmt.Sprintf(" (%d)", f.UnreadCount)
		}
		it := fyne.NewMenuItem(s, func() {
			u.NotificationsArea.SetFolder(f.Folder)
		})
		items2 = append(items2, it)
	}
	return items2
}
