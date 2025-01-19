// Package mobile contains the code for rendering the mobile UI.
package mobile

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/sync/singleflight"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/character"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverse"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/app/widgets"
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

	fyneApp fyne.App
	sfg     *singleflight.Group
	window  fyne.Window
}

var _ ui.UI = (*MobileUI)(nil)

// NewUI build the UI and returns it.
func NewMobileUI(fyneApp fyne.App) *MobileUI {
	u := &MobileUI{
		fyneApp: fyneApp,
		sfg:     new(singleflight.Group),
	}
	u.window = fyneApp.NewWindow(u.appName())

	var main *widgets.Navigator
	var menu *widget.List
	items := []struct {
		icon   fyne.Resource
		name   string
		action func()
	}{
		{
			theme.AccountIcon(), "Character Sheet", func() {
				main.Push("Character Sheet",
					container.NewVBox(
						widget.NewLabel("Character Sheet"),
						widget.NewButton("Detail >", func() {
							main.Push("Detail", widget.NewLabel("Detail"))
						}),
					))
			},
		},
		{
			theme.NewThemedResource(ui.IconInventory2Svg), "Assets", func() {
				main.Push("Assets", widget.NewLabel("PLACEHOLDER"))
			},
		},
		{
			theme.NewThemedResource(ui.IconEarthSvg), "Colonies", func() {
				main.Push("Colonies", widget.NewLabel("PLACEHOLDER"))
			},
		},
		{
			theme.MailComposeIcon(), "Mail", func() {
				main.Push("Mail", widget.NewLabel("PLACEHOLDER"))
			},
		},
		{
			theme.MailComposeIcon(), "Communications", func() {
				main.Push("Communications", widget.NewLabel("PLACEHOLDER"))
			},
		},
		{
			theme.NewThemedResource(ui.IconFileSignSvg), "Contracts", func() {
				main.Push("Contracts", widget.NewLabel("PLACEHOLDER"))
			},
		},
		{
			theme.NewThemedResource(ui.IconGroupSvg), "Characters", func() {
				main.Push("Characters", widget.NewLabel("PLACEHOLDER"))
			},
		},
	}
	menu = widget.NewList(
		func() int {
			return len(items)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(widget.NewIcon(theme.MediaFastForwardIcon()), widget.NewLabel(""), layout.NewSpacer(), widget.NewLabel(">"))
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			item := items[id]
			hbox := co.(*fyne.Container).Objects
			hbox[0].(*widget.Icon).SetResource(item.icon)
			hbox[1].(*widget.Label).SetText(item.name)
		},
	)
	menu.OnSelected = func(id widget.ListItemID) {
		defer menu.UnselectAll()
		items[id].action()
	}
	main = widgets.NewNavigator("Home", menu)
	master := container.NewAppTabs(
		container.NewTabItemWithIcon("Home", theme.HomeIcon(), main),
		container.NewTabItemWithIcon("Character", theme.AccountIcon(), widget.NewLabel("Character")),
		container.NewTabItemWithIcon("More", theme.MenuIcon(), widget.NewLabel("More")),
	)
	master.SetTabLocation(container.TabLocationBottom)
	u.window.SetContent(master)
	return u
}

func (u *MobileUI) Init() {
}

func (u *MobileUI) ShowAndRun() {
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
