package ui

import (
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func makeMenu(u *UI) *fyne.MainMenu {
	// File menu
	fileMenu := fyne.NewMenu("File")

	// Tools menu
	settingsItem := fyne.NewMenuItem("Settings...", u.showSettingsWindow)
	settingsItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyComma, Modifier: fyne.KeyModifierControl}
	u.window.Canvas().AddShortcut(addShortcutFromMenuItem(settingsItem))

	charactersItem := fyne.NewMenuItem("Manage characters...", u.showAccountDialog)
	charactersItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyC, Modifier: fyne.KeyModifierAlt}
	u.window.Canvas().AddShortcut(addShortcutFromMenuItem(charactersItem))

	statusItem := fyne.NewMenuItem("Update status...", u.showStatusWindow)
	statusItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyU, Modifier: fyne.KeyModifierAlt}
	u.window.Canvas().AddShortcut(addShortcutFromMenuItem(statusItem))

	// Help menu
	toolsMenu := fyne.NewMenu("Tools",
		charactersItem,
		fyne.NewMenuItemSeparator(),
		statusItem,
		fyne.NewMenuItemSeparator(),
		settingsItem,
	)
	website := fyne.NewMenuItem("Website", func() {
		url, _ := url.Parse("https://github.com/ErikKalkoken/evebuddy")
		_ = u.fyneApp.OpenURL(url)
	})
	report := fyne.NewMenuItem("Report a bug", func() {
		url, _ := url.Parse("https://github.com/ErikKalkoken/evebuddy/issues")
		_ = u.fyneApp.OpenURL(url)
	})
	if u.IsOffline {
		website.Disabled = true
		report.Disabled = true
	}

	// Help menu
	helpMenu := fyne.NewMenu("Help",
		website,
		report,
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("User data...", func() {
			u.showUserDataDialog()
		}), fyne.NewMenuItem("About...", func() {
			u.showAboutDialog()
		}),
	)
	main := fyne.NewMainMenu(fileMenu, toolsMenu, helpMenu)
	return main
}

func (u *UI) showAboutDialog() {
	c := container.NewVBox()
	info := u.fyneApp.Metadata()
	appData := widget.NewRichTextFromMarkdown(
		"## " + u.appName() + "\n**Version:** " + info.Version)
	c.Add(appData)
	uri, _ := url.Parse("https://github.com/ErikKalkoken/evebuddy")
	c.Add(widget.NewHyperlink("Website", uri))
	c.Add(widget.NewLabel("\"EVE\", \"EVE Online\", \"CCP\", \nand all related logos and images \nare trademarks or registered trademarks of CCP hf."))
	c.Add(widget.NewLabel("(c) 2024 Erik Kalkoken"))
	d := dialog.NewCustom("About", "Close", c, u.window)
	d.Show()
}

func (u *UI) showUserDataDialog() {
	f := widget.NewForm(
		widget.NewFormItem("Cache", makePathEntry(u.window.Clipboard(), u.ad.Cache)),
		widget.NewFormItem("Data", makePathEntry(u.window.Clipboard(), u.ad.Data)),
		widget.NewFormItem("Log", makePathEntry(u.window.Clipboard(), u.ad.Log)),
		widget.NewFormItem("Settings", makePathEntry(u.window.Clipboard(), u.ad.Settings)),
	)
	d := dialog.NewCustom("User data", "Close", f, u.window)
	d.Show()
}

func makePathEntry(cb fyne.Clipboard, p string) *fyne.Container {
	return container.NewHBox(
		widget.NewLabel(p),
		layout.NewSpacer(),
		widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
			cb.SetContent(p)
		}))
}

// addShortcutFromMenuItem is a helper for defining shortcuts.
// It allows to add an already defined shortcut from a menu item to the canvas.
//
// For example:
//
//	window.Canvas().AddShortcut(menuItem)
func addShortcutFromMenuItem(item *fyne.MenuItem) (fyne.Shortcut, func(fyne.Shortcut)) {
	return item.Shortcut, func(s fyne.Shortcut) {
		item.Action()
	}
}
