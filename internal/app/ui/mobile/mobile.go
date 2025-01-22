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

	var characterNav *Navigator
	homeList := NewNavList(
		NewNavListItemWithIcon(
			theme.AccountIcon(),
			"Character Sheet",
			func() {
				characterNav.Push(
					NewAppBar(
						"Character Sheet",
						NewNavList(
							NewNavListItemWithNavigator(
								characterNav,
								NewAppBar("Augmentations", u.ImplantsArea.Content),
							),
							NewNavListItemWithNavigator(
								characterNav,
								NewAppBar("Jump Clones", container.NewScroll(u.JumpClonesArea.Content)),
							),
							NewNavListItemWithNavigator(
								characterNav,
								NewAppBar("Attributes", u.AttributesArea.Content),
							),
							NewNavListItemWithNavigator(
								characterNav,
								NewAppBar("Bio", container.NewScroll(u.BiographyArea.Content)),
							),
						),
					))
			},
		),
		NewNavListItemWithIcon(
			theme.NewThemedResource(ui.IconInventory2Svg),
			"Assets",
			func() {
				characterNav.Push(NewAppBar("Bio", container.NewScroll(u.AssetsArea.Content)))
			},
		),
		NewNavListItemWithIcon(
			theme.NewThemedResource(ui.IconEarthSvg),
			"Colonies",
			func() {
				characterNav.Push(NewAppBar("Colonies", u.PlanetArea.Content))
			},
		),
		NewNavListItemWithIcon(
			theme.MailComposeIcon(),
			"Mail",
			func() {
				characterNav.Push(NewAppBar("Mail", widget.NewLabel("PLACEHOLDER")))
			},
		),
		NewNavListItemWithIcon(
			theme.MailComposeIcon(),
			"Communications",
			func() {
				characterNav.Push(NewAppBar("Communications", widget.NewLabel("PLACEHOLDER")))
			},
		),
		NewNavListItemWithIcon(
			theme.NewThemedResource(ui.IconFileSignSvg),
			"Contracts",
			func() {
				characterNav.Push(NewAppBar("Contracts", u.ContractsArea.Content))
			},
		),
		NewNavListItemWithIcon(
			theme.NewThemedResource(ui.IconSchoolSvg),
			"Skills",
			func() {
				characterNav.Push(
					NewAppBar(
						"Skills",
						NewNavList(
							NewNavListItemWithNavigator(
								characterNav,
								NewAppBar("Training Queue", u.SkillqueueArea.Content),
							),
							NewNavListItemWithNavigator(
								characterNav,
								NewAppBar("Skill Catalogue", u.SkillCatalogueArea.Content),
							),
							NewNavListItemWithNavigator(
								characterNav,
								NewAppBar("Ships", u.ShipsArea.Content),
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
					NewAppBar(
						"Wallet",
						NewNavList(
							NewNavListItemWithNavigator(
								characterNav,
								NewAppBar("Transactions", u.WalletJournalArea.Content),
							),
							NewNavListItemWithNavigator(
								characterNav,
								NewAppBar("Market Transactions", u.WalletTransactionArea.Content),
							),
						),
					))
			},
		),
	)
	characterNav = NewNavigator(NewAppBar("Character", homeList))

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
										NewAppBar("General", makePage(c), NewMenuToolbarAction(
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
										NewAppBar("Eve Online", makePage(c), NewMenuToolbarAction(
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
										NewAppBar("Notifications", makePage(c), NewMenuToolbarAction(
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
			theme.NewThemedResource(ui.IconPortraitSvg),
			"Manage characters",
			func() {
				toolsNav.Push(
					NewAppBar("Manage characters", u.AccountArea.Content))
			},
		),
	)
	toolsNav = NewNavigator(NewAppBar("Tools", toolsList))
	characterTab := container.NewTabItemWithIcon("", ui.IconCharacterplaceholder32Jpeg, characterNav)
	navBar := container.NewAppTabs(
		characterTab,
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
			characterTab.Icon = r
			navBar.Refresh()
		}()

	}
	navBar.SetTabLocation(container.TabLocationBottom)
	u.Window.SetContent(navBar)
	return u
}
