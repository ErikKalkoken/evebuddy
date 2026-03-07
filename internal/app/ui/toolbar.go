package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/xdesktop"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

const (
	searchPlaceholderEnabled  = "Search New Eden"
	searchPlaceholderDisabled = "(Needs a character)"
)

type toolbar struct {
	widget.BaseWidget

	searchBar  *widget.Entry
	searchIcon *xwidget.TappableIcon
	hamburger  *xwidget.TappableIcon
	u          *DesktopUI
}

func newToolbar(u *DesktopUI) *toolbar {
	searchBar := widget.NewEntry()
	searchBar.PlaceHolder = searchPlaceholderEnabled
	searchBar.Scroll = container.ScrollNone
	searchBar.Wrapping = fyne.TextWrapOff
	searchBar.OnSubmitted = func(s string) {
		u.PerformSearch(s)
	}
	clearSearch := xwidget.NewTappableIcon(theme.CancelIcon(), func() {
		searchBar.SetText("")
	})
	clearSearch.SetToolTip("Clear search bar")
	searchBar.ActionItem = container.NewPadded(clearSearch)

	searchIcon := xwidget.NewTappableIcon(theme.SearchIcon(), func() {
		u.showAdvancedSearch(searchBar.Text)
	})
	searchIcon.SetToolTip("Advanced search")

	makeMenuItem := func(title string, sc xdesktop.ShortcutWithHandler) *fyne.MenuItem {
		it := fyne.NewMenuItem(title, func() {
			sc.Handler(sc.Shortcut)
		})
		it.Shortcut = sc.Shortcut
		return it
	}
	close := fyne.NewMenuItem("Close", func() {
		u.MainWindow().Hide()
	})
	close.Shortcut = &desktop.CustomShortcut{
		KeyName:  fyne.KeyF4,
		Modifier: fyne.KeyModifierAlt,
	}
	w := u.MainWindow()
	settings, _ := xdesktop.Shortcut("settings", w)
	characters, _ := xdesktop.Shortcut("manageCharacters", w)
	status, _ := xdesktop.Shortcut("updateStatus", w)
	quit, _ := xdesktop.Shortcut("quit", w)
	menu := fyne.NewMenu(
		"",
		makeMenuItem("Settings", settings),
		makeMenuItem("Manage Characters", characters),
		makeMenuItem("Update Status", status),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("User Data", u.showUserDataDialog),
		fyne.NewMenuItem("About", u.showAboutDialog),
		fyne.NewMenuItemSeparator(),
		close,
		makeMenuItem("Quit", quit),
	)
	hamburger := xwidget.NewTappableIconWithMenu(theme.MenuIcon(), menu)
	hamburger.SetToolTip("Main menu")
	a := &toolbar{
		hamburger:  hamburger,
		searchBar:  searchBar,
		searchIcon: searchIcon,
		u:          u,
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
	x := container.NewGridWithColumns(
		3,
		container.NewHBox(),
		container.NewBorder(nil, nil, nil, a.searchIcon,
			a.searchBar,
		),
		container.New(layout.NewCustomPaddedHBoxLayout(2*p), layout.NewSpacer(), a.hamburger),
	)
	c := container.NewVBox(x, widget.NewSeparator())
	return widget.NewSimpleRenderer(c)
}
