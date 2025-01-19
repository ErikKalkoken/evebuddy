// Package mobile contains the code for rendering the mobile UI.
package mobile

import (
	"context"
	"errors"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/sync/singleflight"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/character"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverse"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
)

// Mobile UI constants
const (
	defaultIconSize = 64
	myFloatFormat   = "#,###.##"
)

type MobileUI struct {
	CacheService       app.CacheService
	CharacterService   *character.CharacterService
	ESIStatusService   app.ESIStatusService
	EveImageService    app.EveImageService
	EveUniverseService *eveuniverse.EveUniverseService
	StatusCacheService app.StatusCacheService
	// Run the app in offline mode
	IsOffline bool
	// Whether to disable update tickers (useful for debugging)
	IsUpdateTickerDisabled bool

	navBar       *container.AppTabs
	characterTab *container.TabItem
	character    *app.Character
	fyneApp      fyne.App
	sfg          *singleflight.Group
	window       fyne.Window
}

var _ ui.UI = (*MobileUI)(nil)

// NewUI build the UI and returns it.
func NewMobileUI(fyneApp fyne.App) *MobileUI {
	u := &MobileUI{
		fyneApp: fyneApp,
		sfg:     new(singleflight.Group),
	}
	u.window = fyneApp.NewWindow(u.appName())

	var main *Navigator
	menu := NewNavList(
		NewNavListItem(
			theme.AccountIcon(),
			"Character Sheet",
			func() {
				main.Push("Character Sheet",
					container.NewVBox(
						widget.NewLabel("Character Sheet"),
						NewNavList(
							NewNavListItem(
								nil,
								"Attributes",
								func() {
									main.Push("Attributes", widget.NewLabel("PLACEHOLDER"))
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
					))
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
		container.NewTabItemWithIcon("", theme.MenuIcon(), widget.NewLabel("More")),
	)
	u.navBar.SetTabLocation(container.TabLocationBottom)
	u.window.SetContent(u.navBar)
	return u
}

func (u *MobileUI) Init() {
	var c *app.Character
	var err error
	ctx := context.Background()
	if cID := u.fyneApp.Preferences().Int(ui.SettingLastCharacterID); cID != 0 {
		c, err = u.CharacterService.GetCharacter(ctx, int32(cID))
		if err != nil {
			if !errors.Is(err, character.ErrNotFound) {
				slog.Error("Failed to load character", "error", err)
			}
		}
	}
	if c == nil {
		c, err = u.CharacterService.GetAnyCharacter(ctx)
		if err != nil {
			if !errors.Is(err, character.ErrNotFound) {
				slog.Error("Failed to load character", "error", err)
			}
		}
	}
	if c == nil {
		return
	}
	u.character = c
}

func (u *MobileUI) ShowAndRun() {
	u.fyneApp.Lifecycle().SetOnStarted(func() {
		slog.Info("App started")
		if u.IsOffline {
			slog.Info("Started in offline mode")
		}
		if u.IsUpdateTickerDisabled {
			slog.Info("Update ticker disabled")
		}
		go func() {
			// u.refreshCrossPages()
			if u.hasCharacter() {
				u.setCharacter(u.character)
			} else {
				u.resetCharacter()
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
	u.fyneApp.Lifecycle().SetOnStopped(func() {
		slog.Info("App shut down complete")
	})
	u.window.ShowAndRun()
}

func (u *MobileUI) appName() string {
	info := u.fyneApp.Metadata()
	name := info.Name
	if name == "" {
		return "EVE Buddy"
	}
	return name
}

func (u *MobileUI) hasCharacter() bool {
	return u.character != nil
}

func (u *MobileUI) setCharacter(c *app.Character) {
	u.character = c
	u.refreshCharacter()
	u.fyneApp.Preferences().SetInt(ui.SettingLastCharacterID, int(c.ID))
}

func (u *MobileUI) resetCharacter() {
	u.character = nil
	u.fyneApp.Preferences().SetInt(ui.SettingLastCharacterID, 0)
	u.refreshCharacter()
}

func (u *MobileUI) refreshCharacter() {
	if u.character != nil {
		characterID := u.character.ID
		r, err := u.EveImageService.CharacterPortrait(characterID, defaultIconSize)
		if err != nil {
			slog.Error("Failed to fetch character portrait", "characterID", characterID, "err", err)
			r = ui.IconCharacterplaceholder32Jpeg
		}
		u.characterTab.Icon = r
		u.navBar.Refresh()
	}
}
