package ui

import (
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

func makeMenu(u *ui) *fyne.MainMenu {
	fileMenu := fyne.NewMenu("File")

	settingsItem := fyne.NewMenuItem("Settings...", func() {
		u.showSettingsWindow()
	})
	settingsShortcut := &desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: fyne.KeyModifierAlt}
	settingsItem.Shortcut = settingsShortcut
	u.window.Canvas().AddShortcut(settingsShortcut, func(shortcut fyne.Shortcut) {
		u.showSettingsWindow()
	})

	charactersItem := fyne.NewMenuItem("Manage characters...", func() {
		u.showAccountDialog()
	})
	charactersShortcut := &desktop.CustomShortcut{KeyName: fyne.KeyC, Modifier: fyne.KeyModifierAlt}
	charactersItem.Shortcut = charactersShortcut
	u.window.Canvas().AddShortcut(charactersShortcut, func(shortcut fyne.Shortcut) {
		u.showAccountDialog()
	})

	statusItem := fyne.NewMenuItem("Update status...", func() {
		u.showStatusWindow()
	})
	statusShortcut := &desktop.CustomShortcut{KeyName: fyne.KeyU, Modifier: fyne.KeyModifierAlt}
	statusItem.Shortcut = statusShortcut
	u.window.Canvas().AddShortcut(statusShortcut, func(shortcut fyne.Shortcut) {
		u.showStatusWindow()
	})

	toolsMenu := fyne.NewMenu("Tools",
		charactersItem,
		fyne.NewMenuItemSeparator(),
		statusItem,
		fyne.NewMenuItemSeparator(),
		settingsItem,
	)
	helpMenu := fyne.NewMenu("Help",
		fyne.NewMenuItem("Website", func() {
			url, _ := url.Parse("https://github.com/ErikKalkoken/evebuddy")
			_ = u.fyneApp.OpenURL(url)
		}),
		fyne.NewMenuItem("Report a bug", func() {
			url, _ := url.Parse("https://github.com/ErikKalkoken/evebuddy/issues")
			_ = u.fyneApp.OpenURL(url)
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("About...", func() {
			u.showAboutDialog()
		}),
	)
	main := fyne.NewMainMenu(fileMenu, toolsMenu, helpMenu)
	return main
}
