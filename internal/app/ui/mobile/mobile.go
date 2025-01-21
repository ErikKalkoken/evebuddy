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
		NewNavListItem(
			theme.AccountIcon(),
			"Character Sheet",
			func() {
				characterNav.Push(
					NewAppBar(
						"Character Sheet",
						NewNavList(
							NewNavListItem(
								nil,
								"Attributes",
								func() {
									characterNav.Push(NewAppBar("Attributes", u.AttributesArea.Content))
								},
							),
							NewNavListItem(
								nil,
								"Implants",
								func() {
									characterNav.Push(NewAppBar("Implants", widget.NewLabel("PLACEHOLDER")))
								},
							),
						),
					))
			},
		),
		NewNavListItem(
			theme.NewThemedResource(ui.IconInventory2Svg),
			"Assets",
			func() {
				characterNav.Push(NewAppBar("Assets", widget.NewLabel("PLACEHOLDER")))
			},
		),
		NewNavListItem(
			theme.NewThemedResource(ui.IconEarthSvg),
			"Colonies",
			func() {
				characterNav.Push(NewAppBar("Colonies", widget.NewLabel("PLACEHOLDER")))
			},
		),
		NewNavListItem(
			theme.MailComposeIcon(),
			"Mail",
			func() {
				characterNav.Push(NewAppBar("Mail", widget.NewLabel("PLACEHOLDER")))
			},
		),
		NewNavListItem(
			theme.MailComposeIcon(),
			"Communications",
			func() {
				characterNav.Push(NewAppBar("Communications", widget.NewLabel("PLACEHOLDER")))
			},
		),
		NewNavListItem(
			theme.NewThemedResource(ui.IconFileSignSvg),
			"Contracts",
			func() {
				characterNav.Push(NewAppBar("Contracts", widget.NewLabel("PLACEHOLDER")))
			},
		))
	characterNav = NewNavigator(NewAppBar("Character", homeList))

	var crossNav *Navigator
	crossList := NewNavList(
		NewNavListItem(
			nil,
			"Overview",
			func() {
				crossNav.Push(NewAppBar("Overview", widget.NewLabel("PLACEHOLDER")))
			},
		),
		NewNavListItem(
			nil,
			"Asset Search",
			func() {
				crossNav.Push(NewAppBar("Asset Search", widget.NewLabel("PLACEHOLDER")))
			},
		),
		NewNavListItem(
			nil,
			"Wealth",
			func() {
				crossNav.Push(NewAppBar("Wealth", widget.NewLabel("PLACEHOLDER")))
			},
		),
	)
	crossNav = NewNavigator(NewAppBar("Characters", crossList))

	var settingsNav *Navigator
	makePage := func(c fyne.CanvasObject) fyne.CanvasObject {
		return container.NewScroll(c)
	}
	settingsList := NewNavList(
		NewNavListItem(
			nil,
			"General",
			func() {
				c, f := u.MakeGeneralSettingsPage(nil)
				settingsNav.Push(
					NewAppBar("General", makePage(c), NewMenuToolbarAction(
						fyne.NewMenuItem(
							"Reset", f,
						))))
			},
		),
		NewNavListItem(
			nil,
			"Eve Online",
			func() {
				c, f := u.MakeEVEOnlinePage()
				settingsNav.Push(
					NewAppBar("Eve Online", makePage(c), NewMenuToolbarAction(
						fyne.NewMenuItem(
							"Reset", f,
						))))
			},
		),
		NewNavListItem(
			nil,
			"Notifications",
			func() {
				c, f := u.MakeNotificationPage(nil)
				settingsNav.Push(
					NewAppBar("Notifications", makePage(c), NewMenuToolbarAction(
						fyne.NewMenuItem(
							"Reset", f,
						),
					)))
			},
		),
		NewNavListItem(
			nil,
			"Manage characters",
			func() {
				settingsNav.Push(
					NewAppBar("Manage characters", u.AccountArea.Content))
			},
		),
	)
	settingsNav = NewNavigator(NewAppBar("Settings", settingsList))
	characterTab := container.NewTabItemWithIcon("", ui.IconCharacterplaceholder32Jpeg, characterNav)
	navBar := container.NewAppTabs(
		characterTab,
		container.NewTabItemWithIcon("", theme.NewThemedResource(ui.IconGroupSvg), crossNav),
		container.NewTabItemWithIcon("", theme.SettingsIcon(), settingsNav),
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
