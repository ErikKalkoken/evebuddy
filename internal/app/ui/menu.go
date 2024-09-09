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
		fyne.NewMenuItem("User data...", func() {
			u.showUserDataDialog()
		}), fyne.NewMenuItem("About...", func() {
			u.showAboutDialog()
		}),
	)
	main := fyne.NewMainMenu(fileMenu, toolsMenu, helpMenu)
	return main
}

func (u *ui) showAboutDialog() {
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

func (u *ui) showUserDataDialog() {
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
