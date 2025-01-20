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

// Mobile UI constants
const (
	defaultIconSize = 64
	myFloatFormat   = "#,###.##"
)

type MobileUI struct {
	*ui.BaseUI

	navBar         *container.AppTabs
	characterTab   *container.TabItem
	attributesArea *ui.Attributes
}

// NewUI build the UI and returns it.
func NewMobileUI(fyneApp fyne.App) *MobileUI {
	u := &MobileUI{}
	u.BaseUI = ui.NewBaseUI(fyneApp, u.refreshCharacter)

	u.attributesArea = u.NewAttributes()

	var main *Navigator
	menu := NewNavList(
		NewNavListItem(
			theme.AccountIcon(),
			"Character Sheet",
			func() {
				main.Push("Character Sheet",
					NewNavList(
						NewNavListItem(
							nil,
							"Attributes",
							func() {
								main.Push("Attributes", u.attributesArea.Content)
							},
						),
						NewNavListItem(
							nil,
							"Implants",
							func() {
								main.Push("Implants", widget.NewLabel("PLACEHOLDER"))
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
				main.Push("Assets", widget.NewLabel("PLACEHOLDER"))
			},
		),
		NewNavListItem(
			theme.NewThemedResource(ui.IconEarthSvg),
			"Colonies",
			func() {
				main.Push("Colonies", widget.NewLabel("PLACEHOLDER"))
			},
		),
		NewNavListItem(
			theme.MailComposeIcon(),
			"Mail",
			func() {
				main.Push("Mail", widget.NewLabel("PLACEHOLDER"))
			},
		),
		NewNavListItem(
			theme.MailComposeIcon(),
			"Communications",
			func() {
				main.Push("Communications", widget.NewLabel("PLACEHOLDER"))
			},
		),
		NewNavListItem(
			theme.NewThemedResource(ui.IconFileSignSvg),
			"Contracts",
			func() {
				main.Push("Contracts", widget.NewLabel("PLACEHOLDER"))
			},
		),
		NewNavListItem(
			theme.NewThemedResource(ui.IconGroupSvg),
			"Characters",
			func() {
				main.Push("Characters", widget.NewLabel("PLACEHOLDER"))
			},
		))
	main = NewNavigator("Home", menu)
	u.characterTab = container.NewTabItemWithIcon("", theme.AccountIcon(), widget.NewLabel("Character"))
	u.navBar = container.NewAppTabs(
		container.NewTabItemWithIcon("", theme.HomeIcon(), main),
		u.characterTab,
		container.NewTabItemWithIcon("", theme.SettingsIcon(), widget.NewLabel("Settings")),
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
		u.attributesArea.Refresh()
	}
}
