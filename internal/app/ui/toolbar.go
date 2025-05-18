package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

const (
	searchPlaceholderEnabled  = "Search New Eden"
	searchPlaceholderDisabled = "(Needs a character)"
)

type toolbar struct {
	widget.BaseWidget

	searchbar *widget.Entry
	hamburger *iwidget.IconButton
	u         *DesktopUI
}

func newToolbar(u *DesktopUI) *toolbar {
	searchbar := widget.NewEntry()
	searchbar.PlaceHolder = searchPlaceholderEnabled
	searchbar.Scroll = container.ScrollNone
	searchbar.Wrapping = fyne.TextWrapOff
	searchbar.OnSubmitted = func(s string) {
		u.PerformSearch(s)
	}
	searchbar.ActionItem = iwidget.NewIconButton(theme.CancelIcon(), func() {
		searchbar.SetText("")
	})
	makeMenuItem := func(title string, sc shortcutDef) *fyne.MenuItem {
		it := fyne.NewMenuItem(title, func() {
			sc.handler(sc.shortcut)
		})
		it.Shortcut = sc.shortcut
		return it
	}
	quit := fyne.NewMenuItem("Quit", func() {
		u.App().Quit()
	})
	quit.Shortcut = &desktop.CustomShortcut{
		KeyName:  fyne.KeyF4,
		Modifier: fyne.KeyModifierAlt,
	}
	menu := fyne.NewMenu(
		"",
		makeMenuItem("Settings", u.shortcuts["settings"]),
		makeMenuItem("Manage Characters", u.shortcuts["manageCharacters"]),
		makeMenuItem("Update Status", u.shortcuts["updateStatus"]),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("User Data", u.showUserDataDialog),
		fyne.NewMenuItem("About", u.ShowAboutDialog),
		fyne.NewMenuItemSeparator(),
		makeMenuItem("Quit", u.shortcuts["quit"]),
	)
	a := &toolbar{
		u:         u,
		searchbar: searchbar,
		hamburger: iwidget.NewIconButtonWithMenu(theme.MenuIcon(), menu),
	}
	a.ExtendBaseWidget(a)
	return a
}

func (a *toolbar) ToogleSearchBar(enabled bool) {
	if enabled {
		a.searchbar.PlaceHolder = searchPlaceholderEnabled
		a.searchbar.Enable()
	} else {
		a.searchbar.PlaceHolder = searchPlaceholderDisabled
		a.searchbar.Disable()
	}
}

func (a *toolbar) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	x := container.NewGridWithColumns(
		3,
		container.NewHBox(),
		container.NewBorder(nil, nil, nil, iwidget.NewIconButton(theme.SearchIcon(), func() {
			a.u.showAdvancedSearch()
		}),
			a.searchbar,
		),
		container.New(layout.NewCustomPaddedHBoxLayout(2*p), layout.NewSpacer(), a.hamburger),
	)
	c := container.NewVBox(x, widget.NewSeparator())
	return widget.NewSimpleRenderer(c)
}
