package ui

import (
	"context"
	"log/slog"
	"net/url"

	"fyne.io/fyne/v2"
)

func makeMenu(u *ui) (*fyne.MainMenu, *fyne.Menu) {
	fileMenu := fyne.NewMenu("File",
		fyne.NewMenuItem("Settings...", func() {
			u.showSettingsDialog()
		}),
	)
	characterMenu := fyne.NewMenu("Characters",
		fyne.NewMenuItem("Manage...", func() {
			u.showAccountDialog()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Switch", nil),
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
	main := fyne.NewMainMenu(fileMenu, characterMenu, helpMenu)
	return main, characterMenu
}

func (u *ui) refreshCharacterMenu() error {
	switchItem := u.characterMenu.Items[2]
	switchItem.ChildMenu = fyne.NewMenu("")
	currentCharacterID := u.characterID()
	ctx := context.TODO()
	menuItems := make([]*fyne.MenuItem, 0)
	cc, err := u.CharacterService.ListCharactersShort(ctx)
	if err != nil {
		return err
	}
	for _, myC := range cc {
		if myC.ID == currentCharacterID {
			continue
		}
		item := fyne.NewMenuItem(myC.Name, func() {
			err := u.loadCharacter(ctx, myC.ID)
			if err != nil {
				msg := "Failed to switch to new character"
				slog.Error(msg, "err", err)
				u.showErrorDialog(msg, err)
				return
			}
		})
		item.Icon = resourceCharacterplaceholder32Jpeg
		go func() {
			r, err := u.EveImageService.CharacterPortrait(myC.ID, defaultIconSize)
			if err != nil {
				slog.Error("Failed to fetch character portrait", "characterID", myC.ID, "err", err)
				r = resourceCharacterplaceholder32Jpeg
			}
			item.Icon = r
			u.characterMenu.Refresh()
		}()
		menuItems = append(menuItems, item)
	}
	switchItem.ChildMenu.Items = menuItems
	if len(menuItems) == 0 {
		switchItem.Disabled = true
	} else {
		switchItem.Disabled = false
	}
	u.characterMenu.Refresh()
	return nil
}
