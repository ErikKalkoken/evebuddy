package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	// ttwidget "github.com/dweymouth/fyne-tooltip/widget"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

const (
	searchPlaceholderEnabled  = "Search New Eden"
	searchPlaceholderDisabled = "(Needs a character)"
)

type toolbar struct {
	widget.BaseWidget

	searchBar *widget.Entry
	hamburger *kxwidget.IconButton
	u         *DesktopUI
}

func newToolbar(u *DesktopUI) *toolbar {
	searchBar := widget.NewEntry()
	searchBar.PlaceHolder = searchPlaceholderEnabled
	searchBar.Scroll = container.ScrollNone
	searchBar.Wrapping = fyne.TextWrapOff
	searchBar.OnSubmitted = func(s string) {
		u.PerformSearch(s)
	}
	clearSearch := iwidget.NewTappableIcon(theme.CancelIcon(), func() {
		searchBar.SetText("")
	})
	clearSearch.SetToolTip("Clear search bar")
	searchBar.ActionItem = container.NewPadded(clearSearch)
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
		searchBar: searchBar,
		hamburger: kxwidget.NewIconButtonWithMenu(theme.MenuIcon(), menu),
	}
	a.ExtendBaseWidget(a)
	return a
}

func (a *toolbar) ToogleSearchBar(enabled bool) {
	if enabled {
		a.searchBar.PlaceHolder = searchPlaceholderEnabled
		a.searchBar.Enable()
	} else {
		a.searchBar.PlaceHolder = searchPlaceholderDisabled
		a.searchBar.Disable()
	}
}

func (a *toolbar) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	searchIcon := iwidget.NewTappableIcon(theme.SearchIcon(), func() {
		a.u.showAdvancedSearch()
	})
	searchIcon.SetToolTip("Advanced search")
	x := container.NewGridWithColumns(
		3,
		container.NewHBox(),
		container.NewBorder(nil, nil, nil, searchIcon,
			a.searchBar,
		),
		container.New(layout.NewCustomPaddedHBoxLayout(2*p), layout.NewSpacer(), a.hamburger),
	)
	c := container.NewVBox(x, widget.NewSeparator())
	return widget.NewSimpleRenderer(c)
}
