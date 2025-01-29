package desktop

import (
	"log/slog"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxdialog "github.com/ErikKalkoken/fyne-kx/dialog"
)

func makeMenu(u *DesktopUI) *fyne.MainMenu {
	// File menu
	fileMenu := fyne.NewMenu("File")

	// Tools menu
	settingsItem := fyne.NewMenuItem("Settings...", u.showSettingsWindow)
	settingsItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyComma, Modifier: fyne.KeyModifierControl}
	u.menuItemsWithShortcut = append(u.menuItemsWithShortcut, settingsItem)

	charactersItem := fyne.NewMenuItem("Manage characters...", u.showAccountWindow)
	charactersItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyC, Modifier: fyne.KeyModifierAlt}
	u.menuItemsWithShortcut = append(u.menuItemsWithShortcut, charactersItem)

	statusItem := fyne.NewMenuItem("Update status...", u.ShowUpdateStatusWindow)
	statusItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyU, Modifier: fyne.KeyModifierAlt}
	u.menuItemsWithShortcut = append(u.menuItemsWithShortcut, statusItem)

	u.enableMenuShortcuts()

	// Help menu
	toolsMenu := fyne.NewMenu("Tools",
		charactersItem,
		fyne.NewMenuItemSeparator(),
		statusItem,
		fyne.NewMenuItemSeparator(),
		settingsItem,
	)
	website := fyne.NewMenuItem("Website", func() {
		if err := u.FyneApp.OpenURL(u.WebsiteRootURL()); err != nil {
			slog.Error("open main website", "error", err)
		}
	})
	report := fyne.NewMenuItem("Report a bug", func() {
		url := u.WebsiteRootURL().JoinPath("issues")
		if err := u.FyneApp.OpenURL(url); err != nil {
			slog.Error("open issue website", "error", err)
		}
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

// enableMenuShortcuts enables all registered menu shortcuts.
func (u *DesktopUI) enableMenuShortcuts() {
	addShortcutFromMenuItem := func(item *fyne.MenuItem) (fyne.Shortcut, func(fyne.Shortcut)) {
		return item.Shortcut, func(s fyne.Shortcut) {
			item.Action()
		}
	}
	for _, mi := range u.menuItemsWithShortcut {
		u.Window.Canvas().AddShortcut(addShortcutFromMenuItem(mi))
	}
}

// disableMenuShortcuts disabled all registered menu shortcuts.
func (u *DesktopUI) disableMenuShortcuts() {
	for _, mi := range u.menuItemsWithShortcut {
		u.Window.Canvas().RemoveShortcut(mi.Shortcut)
	}
}

func (u *DesktopUI) showAboutDialog() {
	d := dialog.NewCustom("About", "Close", u.MakeAboutPage(), u.Window)
	kxdialog.AddDialogKeyHandler(d, u.Window)
	u.disableMenuShortcuts()
	d.SetOnClosed(func() {
		u.enableMenuShortcuts()
	})
	d.Show()
}

func (u *DesktopUI) showUserDataDialog() {
	f := widget.NewForm(
		widget.NewFormItem("DB", makePathEntry(u.Window.Clipboard(), u.DataPaths["db"])),
		widget.NewFormItem("Log", makePathEntry(u.Window.Clipboard(), u.DataPaths["log"])),
		widget.NewFormItem("Settings", makePathEntry(u.Window.Clipboard(), u.FyneApp.Storage().RootURI().Path())),
	)
	d := dialog.NewCustom("User data", "Close", f, u.Window)
	kxdialog.AddDialogKeyHandler(d, u.Window)
	u.disableMenuShortcuts()
	d.SetOnClosed(func() {
		u.enableMenuShortcuts()
	})
	d.Show()
}

func makePathEntry(cb fyne.Clipboard, path string) *fyne.Container {
	p := filepath.Clean(path)
	return container.NewHBox(
		widget.NewLabel(p),
		layout.NewSpacer(),
		widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
			cb.SetContent(p)
		}))
}
