package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type Toolbar struct {
	widget.BaseWidget

	search    *widget.Entry
	hamburger *iwidget.IconButton
	u         *UIDesktop
}

func NewToolbar(u *UIDesktop) *Toolbar {
	search := widget.NewEntry()
	search.PlaceHolder = "Search New Eden"
	search.Scroll = container.ScrollNone
	search.Wrapping = fyne.TextWrapOff
	search.OnSubmitted = func(s string) {
		u.PerformSearch(s)
	}
	search.ActionItem = iwidget.NewIconButton(theme.CancelIcon(), func() {
		search.SetText("")
	})
	makeMenuItem := func(title string, sc shortcutDef) *fyne.MenuItem {
		it := fyne.NewMenuItem(title, func() {
			sc.handler(sc.shortcut)
		})
		it.Shortcut = sc.shortcut
		return it
	}
	menu := fyne.NewMenu(
		"",
		makeMenuItem("Settings", u.shortcuts["settings"]),
		makeMenuItem("Manage Characters", u.shortcuts["manageCharacters"]),
		makeMenuItem("Update Status", u.shortcuts["updateStatus"]),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("User Data", u.showUserDataDialog),
		fyne.NewMenuItem("About", u.ShowAboutDialog),
	)
	a := &Toolbar{
		u:         u,
		search:    search,
		hamburger: iwidget.NewIconButtonWithMenu(theme.MenuIcon(), menu),
	}
	a.ExtendBaseWidget(a)
	return a
}

func (a *Toolbar) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	x := container.NewGridWithColumns(
		3,
		container.NewHBox(),
		container.NewBorder(nil, nil, nil, iwidget.NewIconButton(theme.SearchIcon(), func() {
			a.u.ShowAdvancedSearch()
		}),
			a.search,
		),
		container.New(layout.NewCustomPaddedHBoxLayout(2*p), layout.NewSpacer(), a.hamburger),
	)
	c := container.NewVBox(x, widget.NewSeparator())
	return widget.NewSimpleRenderer(c)
}
