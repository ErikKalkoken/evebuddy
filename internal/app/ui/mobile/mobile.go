// Package mobile contains the code for rendering the mobile UI.
package mobile

import (
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
)

type MobileUI struct {
	*ui.BaseUI
}

// NewUI build the UI and returns it.
func NewMobileUI(fyneApp fyne.App) *MobileUI {
	u := &MobileUI{}
	u.BaseUI = ui.NewBaseUI(fyneApp)

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
		widget.ShowPopUpMenuAtRelativePosition(
			fyne.NewMenu("", items...),
			fyne.CurrentApp().Driver().CanvasForObject(o),
			fyne.Position{},
			o,
		)
	}

	newCharacterAppBar := func(title string, body fyne.CanvasObject, items ...widget.ToolbarItem) *AppBar {
		items = append(items, characterSelector)
		return NewAppBar(title, body, items...)
	}

	var characterNav *Navigator
	homeList := NewNavList(
		NewNavListItemWithIcon(
			theme.AccountIcon(),
			"Character Sheet",
			func() {
				characterNav.Push(
					newCharacterAppBar(
						"Character Sheet",
						NewNavListWithTitle("Character Sheet",
							NewNavListItemWithNavigator(
								characterNav,
								newCharacterAppBar("Augmentations", u.ImplantsArea.Content),
							),
							NewNavListItemWithNavigator(
								characterNav,
								newCharacterAppBar("Jump Clones", container.NewScroll(u.JumpClonesArea.Content)),
							),
							NewNavListItemWithNavigator(
								characterNav,
								newCharacterAppBar("Attributes", u.AttributesArea.Content),
							),
							NewNavListItemWithNavigator(
								characterNav,
								newCharacterAppBar("Bio", container.NewScroll(u.BiographyArea.Content)),
							),
						),
					))
			},
		),
		NewNavListItemWithIcon(
			theme.NewThemedResource(ui.IconInventory2Svg),
			"Assets",
			func() {
				characterNav.Push(newCharacterAppBar("Assets", container.NewScroll(u.AssetsArea.Content)))
			},
		),
		NewNavListItemWithIcon(
			theme.NewThemedResource(ui.IconEarthSvg),
			"Colonies",
			func() {
				characterNav.Push(newCharacterAppBar("Colonies", u.PlanetArea.Content))
			},
		),
		NewNavListItemWithIcon(
			theme.MailComposeIcon(),
			"Mail",
			func() {
				characterNav.Push(newCharacterAppBar("Mail", widget.NewLabel("PLACEHOLDER")))
			},
		),
		NewNavListItemWithIcon(
			theme.MailComposeIcon(),
			"Communications",
			func() {
				characterNav.Push(
					newCharacterAppBar(
						"Communications",
						u.NotificationsArea.Notifications,
					),
				)
			},
		),
		NewNavListItemWithIcon(
			theme.NewThemedResource(ui.IconFileSignSvg),
			"Contracts",
			func() {
				characterNav.Push(newCharacterAppBar("Contracts", u.ContractsArea.Content))
			},
		),
		NewNavListItemWithIcon(
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
		),
		NewNavListItemWithIcon(
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
		),
	)

	characterPage := newCharacterAppBar("Character", homeList)
	characterNav = NewNavigator(characterPage)

	var crossNav *Navigator
	crossList := NewNavList(
		NewNavListItem(
			"Overview",
			func() {
				crossNav.Push(NewAppBar("Overview", u.OverviewArea.Content))
			},
		),
		NewNavListItem(
			"Asset Search",
			func() {
				crossNav.Push(NewAppBar("Asset Search", u.AssetSearchArea.Content))
			},
		),
		NewNavListItem(
			"Locations",
			func() {
				crossNav.Push(NewAppBar("Locations", u.LocationsArea.Content))
			},
		),
		NewNavListItem(
			"Training",
			func() {
				crossNav.Push(NewAppBar("Training", u.TrainingArea.Content))
			},
		),
		NewNavListItem(
			"Colonies",
			func() {
				crossNav.Push(NewAppBar("Colonies", u.ColoniesArea.Content))
			},
		),
		NewNavListItem(
			"Wealth",
			func() {
				crossNav.Push(NewAppBar("Wealth", u.WealthArea.Content))
			},
		),
	)
	crossNav = NewNavigator(NewAppBar("Characters", crossList))

	var toolsNav *Navigator
	makePage := func(c fyne.CanvasObject) fyne.CanvasObject {
		return container.NewScroll(c)
	}
	toolsList := NewNavList(
		NewNavListItemWithIcon(
			theme.SettingsIcon(),
			"Settings",
			func() {
				toolsNav.Push(
					NewAppBar(
						"Settings",
						NewNavList(
							NewNavListItem(
								"General",
								func() {
									c, f := u.MakeGeneralSettingsPage(nil)
									toolsNav.Push(
										NewAppBar("General", makePage(c), NewToolbarActionMenu(
											fyne.NewMenuItem(
												"Reset", f,
											))))
								},
							),
							NewNavListItem(
								"Eve Online",
								func() {
									c, f := u.MakeEVEOnlinePage()
									toolsNav.Push(
										NewAppBar("Eve Online", makePage(c), NewToolbarActionMenu(
											fyne.NewMenuItem(
												"Reset", f,
											))))
								},
							),
							NewNavListItem(
								"Notifications",
								func() {
									c, f := u.MakeNotificationPage(nil)
									toolsNav.Push(
										NewAppBar("Notifications", makePage(c), NewToolbarActionMenu(
											fyne.NewMenuItem(
												"Reset", f,
											),
										)))
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
				toolsNav.Push(
					NewAppBar("Manage characters", u.AccountArea.Content))
			},
		),
	)
	toolsNav = NewNavigator(NewAppBar("Tools", toolsList))
	navBar := container.NewAppTabs(
		container.NewTabItemWithIcon("", theme.AccountIcon(), characterNav),
		container.NewTabItemWithIcon("", theme.NewThemedResource(ui.IconGroupSvg), crossNav),
		container.NewTabItemWithIcon("", theme.NewThemedResource(ui.IconToolsSvg), toolsNav),
	)
	u.OnSetCharacter = func(id int32) {
		go func() {
			r, err := u.EveImageService.CharacterPortrait(id, ui.DefaultIconSize)
			if err != nil {
				slog.Error("Failed to fetch character portrait", "characterID", id, "err", err)
				r = ui.IconCharacterplaceholder32Jpeg
			}
			characterSelector.SetIcon(r)
		}()
	}
	navBar.SetTabLocation(container.TabLocationBottom)
	u.Window.SetContent(navBar)
	return u
}
