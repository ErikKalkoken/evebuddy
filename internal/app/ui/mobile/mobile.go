// Package mobile contains the code for rendering the mobile UI.
package mobile

import (
	"context"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
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

	var home *Navigator
	menu := NewNavList(
		NewNavListItem(
			theme.AccountIcon(),
			"Character Sheet",
			func() {
				home.Push("Character Sheet",
					NewNavList(
						NewNavListItem(
							nil,
							"Attributes",
							func() {
								home.Push("Attributes", u.AttributesArea.Content)
							},
						),
						NewNavListItem(
							nil,
							"Implants",
							func() {
								home.Push("Implants", widget.NewLabel("PLACEHOLDER"))
							},
						),
					),
				)
			},
		),
		NewNavListItem(
			theme.NewThemedResource(ui.IconInventory2Svg),
			"Assets",
			func() {
				home.Push("Assets", widget.NewLabel("PLACEHOLDER"))
			},
		),
		NewNavListItem(
			theme.NewThemedResource(ui.IconEarthSvg),
			"Colonies",
			func() {
				home.Push("Colonies", widget.NewLabel("PLACEHOLDER"))
			},
		),
		NewNavListItem(
			theme.MailComposeIcon(),
			"Mail",
			func() {
				home.Push("Mail", widget.NewLabel("PLACEHOLDER"))
			},
		),
		NewNavListItem(
			theme.MailComposeIcon(),
			"Communications",
			func() {
				home.Push("Communications", widget.NewLabel("PLACEHOLDER"))
			},
		),
		NewNavListItem(
			theme.NewThemedResource(ui.IconFileSignSvg),
			"Contracts",
			func() {
				home.Push("Contracts", widget.NewLabel("PLACEHOLDER"))
			},
		),
		NewNavListItem(
			theme.NewThemedResource(ui.IconGroupSvg),
			"Characters",
			func() {
				home.Push("Characters", widget.NewLabel("PLACEHOLDER"))
			},
		))
	home = NewNavigator("Home", menu)

	makePage := func(c fyne.CanvasObject, _ func()) fyne.CanvasObject {
		return container.NewScroll(c)
	}
	var settings *Navigator
	settingsList := NewNavList(
		NewNavListItem(
			nil,
			"General",
			func() {
				settings.Push(
					"General",
					makePage(u.MakeGeneralSettingsPage(nil)),
				)
			},
		),
		NewNavListItem(
			nil,
			"Eve Online",
			func() {
				settings.Push(
					"Eve Online",
					makePage(u.MakeEVEOnlinePage()),
				)
			},
		),
		NewNavListItem(
			nil,
			"Notifications",
			func() {
				settings.Push(
					"Notifications",
					makePage(u.MakeNotificationPage(nil)),
				)
			},
		),
	)
	settings = NewNavigator("Settings", settingsList)

	u.characterTab = container.NewTabItemWithIcon("", theme.AccountIcon(), widget.NewLabel(""))
	u.navBar = container.NewAppTabs(
		container.NewTabItemWithIcon("", theme.HomeIcon(), home),
		u.characterTab,
		container.NewTabItemWithIcon("", theme.SettingsIcon(), settings),
	)
	u.navBar.SetTabLocation(container.TabLocationBottom)
	u.navBar.OnSelected = func(ti *container.TabItem) {
		if ti == u.characterTab {
			a := u.NewAccountArea(u.updateCharacterAndRefreshIfNeeded)
			d := dialog.NewCustom("Manage Characters", "Close", a.Content, u.Window)
			a.OnSelectCharacter = func() {
				d.Hide()
			}
			d.SetOnClosed(func() {
				u.refreshCrossPages()
			})
			d.Resize(fyne.Size{Width: 500, Height: 500})
			if err := a.Refresh(); err != nil {
				d.Hide()
				// return err
			}
			d.Show()
		}
	}
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
	}
}

func (u *MobileUI) refreshCrossPages() {
	// TODO
}

func (u *MobileUI) updateCharacterAndRefreshIfNeeded(ctx context.Context, characterID int32, forceUpdate bool) {
	// TODO
}
