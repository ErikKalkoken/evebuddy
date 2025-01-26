// Package mobile contains the code for rendering the mobile UI.
package mobile

import (
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/dustin/go-humanize"
)

type MobileUI struct {
	*ui.BaseUI
}

// NewUI build the UI and returns it.
func NewMobileUI(fyneApp fyne.App) *MobileUI {
	u := &MobileUI{}
	u.BaseUI = ui.NewBaseUI(fyneApp)

	u.MailArea.SendMessage = func(_ ui.SendMessageMode, _ *app.CharacterMail) {
		d := dialog.NewInformation("Send Message", "PLACEHOLDER", u.Window)
		d.Show()
	}

	// character
	characterSelector := widget.NewToolbarAction(ui.IconCharacterplaceholder32Jpeg, nil)
	characterSelector.OnActivated = func() {
		o := characterSelector.ToolbarObject()
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
		ShowContextMenu(o, fyne.NewMenu("", items...))
	}

	newCharacterAppBar := func(title string, body fyne.CanvasObject, items ...widget.ToolbarItem) *AppBar {
		items = append(items, characterSelector)
		return NewAppBar(title, body, items...)
	}

	var characterNav *Navigator
	mailMenu := fyne.NewMenu("")
	communicationsMenu := fyne.NewMenu("")

	navListMail := NewNavListItemWithIcon(
		theme.MailComposeIcon(),
		"Mail",
		func() {
			deleteAction := u.MailArea.MakeDeleteAction(func() {
				characterNav.Pop()
			})
			u.MailArea.OnSelected = func() {
				characterNav.Push(
					NewAppBar("", u.MailArea.Detail, deleteAction),
				)
			}
			characterNav.Push(
				newCharacterAppBar(
					"Mail",
					u.MailArea.Headers,
					NewToolbarActionMenu(theme.FolderIcon(), mailMenu),
				))
		},
	)

	navListCommunications := NewNavListItemWithIcon(
		theme.NewThemedResource(ui.IconMessageSvg),
		"Communications",
		func() {
			u.NotificationsArea.OnSelected = func() {
				characterNav.Push(
					NewAppBar("", u.NotificationsArea.Detail),
				)
			}
			characterNav.Push(
				newCharacterAppBar(
					"Communications",
					u.NotificationsArea.Notifications,
					NewToolbarActionMenu(theme.FolderIcon(), communicationsMenu),
				),
			)
		},
	)
	navListColonies := NewNavListItemWithIcon(
		theme.NewThemedResource(ui.IconEarthSvg),
		"Colonies",
		func() {
			characterNav.Push(newCharacterAppBar("Colonies", u.PlanetArea.Content))
		},
	)
	navListSkills := NewNavListItemWithIcon(
		theme.NewThemedResource(ui.IconSchoolSvg),
		"Skills",
		func() {
			characterNav.Push(
				newCharacterAppBar(
					"Skills",
					NewNavList(
						NewNavListItemWithNavigator(
							characterNav,
							newCharacterAppBar("Training Queue", u.SkillqueueArea.Content),
						),
						NewNavListItemWithNavigator(
							characterNav,
							newCharacterAppBar("Skill Catalogue", u.SkillCatalogueArea.Content),
						),
						NewNavListItemWithNavigator(
							characterNav,
							newCharacterAppBar("Ships", u.ShipsArea.Content),
						),
					),
				))
		},
	)
	navListWallet := NewNavListItemWithIcon(
		theme.NewThemedResource(ui.IconAttachmoneySvg),
		"Wallet",
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
	characterList := NewNavList(
		NewNavListItemWithIcon(
			theme.NewThemedResource(ui.IconInventory2Svg),
			"Assets",
			func() {
				u.AssetsArea.OnSelected = func() {
					characterNav.Push(newCharacterAppBar("", u.AssetsArea.LocationAssets))
				}
				characterNav.Push(newCharacterAppBar("Assets", container.NewHScroll(u.AssetsArea.Locations)))
			},
		),
		navListColonies,
		navListMail,
		navListCommunications,
		NewNavListItemWithIcon(
			theme.NewThemedResource(ui.IconHeadSnowflakeSvg),
			"Clones",
			func() {
				characterNav.Push(
					newCharacterAppBar(
						"Clones",
						container.NewAppTabs(
							container.NewTabItem("Augmentations", u.ImplantsArea.Content),
							container.NewTabItem("Jump Clones", u.JumpClonesArea.Content),
						),
					))
			},
		),
		NewNavListItemWithIcon(
			theme.NewThemedResource(ui.IconFileSignSvg),
			"Contracts",
			func() {
				characterNav.Push(newCharacterAppBar("Contracts", u.ContractsArea.Content))
			},
		),
		navListSkills,
		navListWallet,
	)

	u.PlanetArea.OnCountRefresh = func(count int) {
		s := ""
		if count > 0 {
			s = humanize.Comma(int64(count))
		}
		navListColonies.Suffix = s
		characterList.Refresh()
	}

	u.MailArea.OnUnreadRefresh = func(count int) {
		s := ""
		if count > 0 {
			s = humanize.Comma(int64(count))
		}
		navListMail.Suffix = s
		characterList.Refresh()
	}

	u.NotificationsArea.OnUnreadRefresh = func(count int) {
		s := ""
		if count > 0 {
			s = humanize.Comma(int64(count))
		}
		navListCommunications.Suffix = s
		characterList.Refresh()
	}

	u.SkillqueueArea.OnStatusRefresh = func(status string) {
		navListSkills.Suffix = status
		characterList.Refresh()
	}

	u.WalletJournalArea.OnBalanceRefresh = func(b string) {
		navListWallet.Suffix = b
		characterList.Refresh()
	}

	characterPage := newCharacterAppBar("Character", characterList)
	characterNav = NewNavigator(characterPage)

	// characters cross
	var crossNav *Navigator
	navListWealth := NewNavListItemWithIcon(
		theme.NewThemedResource(ui.IconGoldSvg),
		"Wealth",
		func() {
			crossNav.Push(NewAppBar("Wealth", u.WealthArea.Content))
		},
	)
	crossList := NewNavList(
		NewNavListItemWithIcon(
			theme.NewThemedResource(ui.IconAccountMultipleSvg),
			"Overview",
			func() {
				crossNav.Push(NewAppBar("Overview", u.OverviewArea.Content))
			},
		),
		NewNavListItemWithIcon(
			theme.NewThemedResource(ui.IconInventory2Svg),
			"Asset Search",
			func() {
				crossNav.Push(NewAppBar("Asset Search", u.AssetSearchArea.Content))
			},
		),
		NewNavListItemWithIcon(
			theme.NewThemedResource(ui.IconMapMarkerSvg),
			"Locations",
			func() {
				crossNav.Push(NewAppBar("Locations", u.LocationsArea.Content))
			},
		),
		NewNavListItemWithIcon(
			theme.NewThemedResource(ui.IconSchoolSvg),
			"Training",
			func() {
				crossNav.Push(NewAppBar("Training", u.TrainingArea.Content))
			},
		),
		NewNavListItemWithIcon(
			theme.NewThemedResource(ui.IconEarthSvg),
			"Colonies",
			func() {
				crossNav.Push(NewAppBar("Colonies", u.ColoniesArea.Content))
			},
		),
		navListWealth,
	)
	crossNav = NewNavigator(NewAppBar("Characters", crossList))
	u.WealthArea.OnTotalRefresh = func(total string) {
		navListWealth.Suffix = total
		crossList.Refresh()
	}

	// tools
	var moreNav *Navigator
	makePage := func(c fyne.CanvasObject) fyne.CanvasObject {
		return container.NewScroll(c)
	}
	makeMenu := func(items ...*fyne.MenuItem) (fyne.Resource, *fyne.Menu) {
		return theme.MenuExpandIcon(), fyne.NewMenu("", items...)
	}
	toolsList := NewNavList(
		NewNavListItemWithIcon(
			theme.NewThemedResource(ui.IconCogSvg),
			"Settings",
			func() {
				moreNav.Push(
					NewAppBar(
						"Settings",
						NewNavList(
							NewNavListItem(
								"General",
								func() {
									c, f := u.MakeGeneralSettingsPage(nil)
									moreNav.Push(
										NewAppBar("General", makePage(c), NewToolbarActionMenu(
											makeMenu(fyne.NewMenuItem(
												"Reset", f,
											)))),
									)
								},
							),
							NewNavListItem(
								"Eve Online",
								func() {
									c, f := u.MakeEVEOnlinePage()
									moreNav.Push(
										NewAppBar("Eve Online", makePage(c), NewToolbarActionMenu(
											makeMenu(fyne.NewMenuItem(
												"Reset", f,
											)))),
									)
								},
							),
							NewNavListItem(
								"Notifications",
								func() {
									c, f := u.MakeNotificationGeneralPage(nil)
									moreNav.Push(
										NewAppBar("Notification - General", makePage(c), NewToolbarActionMenu(
											makeMenu(fyne.NewMenuItem(
												"Reset", f,
											)))),
									)
								},
							),
							NewNavListItem(
								"Notification - Types",
								func() {
									c, f := u.MakeNotificationTypesPage()
									moreNav.Push(
										NewAppBar("Notification - Types", makePage(c), NewToolbarActionMenu(
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
		NewNavListItemWithIcon(
			theme.NewThemedResource(ui.IconManageaccountsSvg),
			"Manage characters",
			func() {
				moreNav.Push(NewAppBar("Manage characters", u.AccountArea.Content))
			},
		),
		NewNavListItemWithIcon(
			theme.InfoIcon(),
			"About",
			func() {
				moreNav.Push(NewAppBar("About", u.MakeAboutPage()))
			},
		),
	)
	moreNav = NewNavigator(NewAppBar("More", toolsList))

	// navigation bar
	characterDest := NewNavBarItem("Character", theme.NewThemedResource(ui.IconAccountSvg), characterNav)
	characterDest.OnSelectedAgain = func() {
		characterNav.PopAll()
	}
	crossDest := NewNavBarItem("Characters", theme.NewThemedResource(ui.IconAccountMultipleSvg), crossNav)
	crossDest.OnSelectedAgain = func() {
		crossNav.PopAll()
	}
	moreDest := NewNavBarItem("More", theme.MenuIcon(), moreNav)
	moreDest.OnSelectedAgain = func() {
		moreNav.PopAll()
	}
	navBar := NewNavBar(characterDest, crossDest, moreDest)

	u.OnSetCharacter = func(id int32) {
		// update character selector
		go func() {
			r, err := u.EveImageService.CharacterPortrait(id, ui.DefaultIconPixelSize)
			if err != nil {
				slog.Error("Failed to fetch character portrait", "characterID", id, "err", err)
				r = ui.IconCharacterplaceholder32Jpeg
			}
			characterSelector.SetIcon(r)
		}()

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
