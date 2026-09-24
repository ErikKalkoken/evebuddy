package core

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"

	"github.com/ErikKalkoken/evebuddy/internal/xdesktop"
)

// makeMainMenu returns the app's main menu.
func makeMainMenu(u *DesktopUI) *fyne.Menu {
	makeMenuItem := func(title string, sc xdesktop.ShortcutWithHandler) *fyne.MenuItem {
		it := fyne.NewMenuItem(title, func() {
			sc.Handler(sc.Shortcut)
		})
		it.Shortcut = sc.Shortcut
		return it
	}
	it := fyne.NewMenuItem("Close", func() {
		u.MainWindow().Hide()
	})
	it.Shortcut = &desktop.CustomShortcut{
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
		it,
		makeMenuItem("Quit", quit),
	)
	return menu
}
