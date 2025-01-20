// Package mobile contains the code for rendering the mobile UI.
package mobile

import (
	"context"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
)

// Mobile UI constants
const (
	defaultIconSize = 64
	myFloatFormat   = "#,###.##"
)

type MobileUI struct {
	*ui.BaseUI

	navBar       *container.AppTabs
	characterTab *container.TabItem
}

// NewUI build the UI and returns it.
func NewMobileUI(fyneApp fyne.App) *MobileUI {
	u := &MobileUI{}
	u.BaseUI = ui.NewBaseUI(fyneApp, u.refreshCharacter, u.refreshCrossPages)
	u.AccountArea = u.NewAccountArea(u.updateCharacterAndRefreshIfNeeded)

	var home *Navigator
	menu := NewNavList(
		NewNavListItem(
			theme.AccountIcon(),
			"Character Sheet",
			func() {
				home.Push(
					NewAppBar(
						"Character Sheet",
						NewNavList(
							NewNavListItem(
								nil,
								"Attributes",
								func() {
									home.Push(NewAppBar("Attributes", u.AttributesArea.Content))
								},
							),
							NewNavListItem(
								nil,
								"Implants",
								func() {
									home.Push(NewAppBar("Implants", widget.NewLabel("PLACEHOLDER")))
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
				home.Push(NewAppBar("Assets", widget.NewLabel("PLACEHOLDER")))
			},
		),
		NewNavListItem(
			theme.NewThemedResource(ui.IconEarthSvg),
			"Colonies",
			func() {
				home.Push(NewAppBar("Colonies", widget.NewLabel("PLACEHOLDER")))
			},
		),
		NewNavListItem(
			theme.MailComposeIcon(),
			"Mail",
			func() {
				home.Push(NewAppBar("Mail", widget.NewLabel("PLACEHOLDER")))
			},
		),
		NewNavListItem(
			theme.MailComposeIcon(),
			"Communications",
			func() {
				home.Push(NewAppBar("Communications", widget.NewLabel("PLACEHOLDER")))
			},
		),
		NewNavListItem(
			theme.NewThemedResource(ui.IconFileSignSvg),
			"Contracts",
			func() {
				home.Push(NewAppBar("Contracts", widget.NewLabel("PLACEHOLDER")))
			},
		),
		NewNavListItem(
			theme.NewThemedResource(ui.IconGroupSvg),
			"Characters",
			func() {
				home.Push(NewAppBar("Characters", widget.NewLabel("PLACEHOLDER")))
			},
		))
	home = NewNavigator(NewAppBar("Home", menu))

	makePage := func(c fyne.CanvasObject) fyne.CanvasObject {
		return container.NewScroll(c)
	}
	var settings *Navigator
	settingsList := NewNavList(
		NewNavListItem(
			nil,
			"General",
			func() {
				c, f := u.MakeGeneralSettingsPage(nil)
				settings.Push(
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
				settings.Push(
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
				settings.Push(
					NewAppBar("Notifications", makePage(c), NewMenuToolbarAction(
						fyne.NewMenuItem(
							"Reset", f,
						),
					)))
			},
		),
	)
	settings = NewNavigator(NewAppBar("Settings", settingsList))

	u.characterTab = container.NewTabItemWithIcon("", theme.AccountIcon(), u.AccountArea.Content)
	u.navBar = container.NewAppTabs(
		container.NewTabItemWithIcon("", theme.HomeIcon(), home),
		u.characterTab,
		container.NewTabItemWithIcon("", theme.SettingsIcon(), settings),
	)
	u.navBar.SetTabLocation(container.TabLocationBottom)
	u.Window.SetContent(u.navBar)
	return u
}

func (u *MobileUI) ShowAndRun() {
	u.FyneApp.Lifecycle().SetOnStarted(func() {
		slog.Info("App started")
		if u.IsOffline {
			slog.Info("Started in offline mode")
		}
		if u.IsUpdateTickerDisabled {
			slog.Info("Update ticker disabled")
		}
		go func() {
			// u.refreshCrossPages()
			if u.HasCharacter() {
				u.SetCharacter(u.Character)
			} else {
				u.ResetCharacter()
			}
		}()
		// if !u.IsOffline && !u.IsUpdateTickerDisabled {
		// 	go func() {
		// 		u.startUpdateTickerGeneralSections()
		// 		u.startUpdateTickerCharacters()
		// 	}()
		// }
		// go u.statusBarArea.StartUpdateTicker()
	})
	u.FyneApp.Lifecycle().SetOnStopped(func() {
		slog.Info("App shut down complete")
	})
	u.Window.ShowAndRun()
}

func (u *MobileUI) refreshCharacter() {
	if u.Character != nil {
		characterID := u.Character.ID
		r, err := u.EveImageService.CharacterPortrait(characterID, defaultIconSize)
		if err != nil {
			slog.Error("Failed to fetch character portrait", "characterID", characterID, "err", err)
			r = ui.IconCharacterplaceholder32Jpeg
		}
		u.characterTab.Icon = r
		u.navBar.Refresh()
		u.AttributesArea.Refresh()
		u.AccountArea.Refresh()
	}
}

func (u *MobileUI) refreshCrossPages() {
	// TODO
}

func (u *MobileUI) updateCharacterAndRefreshIfNeeded(ctx context.Context, characterID int32, forceUpdate bool) {
	// TODO
}
