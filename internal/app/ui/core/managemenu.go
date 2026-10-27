package core

import (
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"

	"github.com/ErikKalkoken/evebuddy/internal/xdesktop"
)

// makeManageMenu returns the menu for the nav rail's Manage item.
func makeManageMenu(u *DesktopUI) *fyne.Menu {
	w := u.MainWindow()
	var items []*fyne.MenuItem
	addShortcutItem := func(title, name string) {
		sc, ok := xdesktop.Shortcut(name, w)
		if !ok {
			slog.Error("manage menu: shortcut not found", "name", name)
			return
		}
		it := fyne.NewMenuItem(title, func() {
			sc.Handler(sc.Shortcut)
		})
		it.Shortcut = sc.Shortcut
		items = append(items, it)
	}
	addShortcutItem("Settings", "settings")
	addShortcutItem("Manage Characters", "manageCharacters")
	addShortcutItem("Update Status", "updateStatus")
	items = append(items,
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("User Data", u.showUserDataDialog),
		fyne.NewMenuItem("About", u.showAboutDialog),
		fyne.NewMenuItemSeparator(),
	)
	// without a tray icon a hidden window can not be shown again
	if u.settings.SysTrayEnabled() {
		it := fyne.NewMenuItem("Close", func() {
			w.Hide()
		})
		it.Shortcut = &desktop.CustomShortcut{
			KeyName:  fyne.KeyF4,
			Modifier: fyne.KeyModifierAlt,
		}
		items = append(items, it)
	}
	addShortcutItem("Quit", "quit")
	return fyne.NewMenu("", items...)
}
