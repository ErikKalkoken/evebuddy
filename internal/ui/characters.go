package ui

import (
	"example/esiapp/internal/esi"
	"example/esiapp/internal/sso"
	"example/esiapp/internal/storage"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

const defaultIconSize = 64

var scopes = []string{
	"esi-characters.read_contacts.v1",
	"esi-universe.read_structures.v1",
	"esi-mail.read_mail.v1",
}

type characters struct {
	container *fyne.Container
	folders   *folders
	esiApp    *esiApp
}

func (c *characters) update(charID int32) {
	buttonAdd := c.makeManageButton(charID)
	image, name := makeCharacter(charID)
	c.container.RemoveAll()
	c.container.Add(image)
	c.container.Add(name)
	c.container.Add(layout.NewSpacer())
	c.container.Add(buttonAdd)
	c.container.Refresh()
	c.folders.update(charID)
}

func (c *characters) makeManageButton(charID int32) *contextMenuButton {
	addChar := fyne.NewMenuItem("Add Character", func() {
		info := dialog.NewCustomWithoutButtons(
			"Add Character",
			widget.NewLabel("Please follow instructions in your browser to add a new character."),
			c.esiApp.main,
		)
		info.Show()
		t, err := addCharacter()
		if err != nil {
			log.Printf("Failed to add a new character: %v", err)
		} else {
			c.update(t.CharacterID)
			c.folders.updateMailsWithID(t.CharacterID)
		}
		info.Hide()
	})
	menu := fyne.NewMenu("", addChar)
	switchChar, err := c.makeMenuItem(charID)
	if err != nil {
		log.Printf("Failed to make menu item: %v", err)
	}
	if switchChar != nil {
		menu.Items = append(menu.Items, switchChar)
	}
	buttonAdd := newContextMenuButton("Manage Characters", menu)
	return buttonAdd
}

func (c *characters) makeMenuItem(charID int32) (*fyne.MenuItem, error) {
	chars, err := storage.FetchAllCharacters()
	if err != nil {
		return nil, err
	}
	if len(chars) == 0 {
		return nil, nil
	}
	shareItem := fyne.NewMenuItem("Switch character", nil)

	var items []*fyne.MenuItem
	for _, char := range chars {
		item := fyne.NewMenuItem(char.Name, func() {
			c.update(char.ID)
		})
		if char.ID == charID {
			item.Disabled = true
		}
		items = append(items, item)
	}
	if len(chars) < 2 {
		return nil, nil
	}
	shareItem.ChildMenu = fyne.NewMenu("", items...)
	return shareItem, nil
}

func makeCharacter(charID int32) (*canvas.Image, *widget.Label) {
	char, err := storage.FetchCharacter(charID)
	var label string
	var uri fyne.URI
	if err != nil {
		label = "No characters"
		uri = esi.CharacterPortraitURL(esi.PlaceholderCharacterID, defaultIconSize)
	} else {
		label = char.Name
		uri = char.PortraitURL(defaultIconSize)
	}
	image := canvas.NewImageFromURI(uri)
	image.FillMode = canvas.ImageFillOriginal
	name := widget.NewLabel(label)
	return image, name
}

func (e *esiApp) newCharacters(f *folders) *characters {
	c := characters{esiApp: e, folders: f}
	c.container = container.NewHBox()
	return &c
}

// addCharacter adds a new character via SSO authentication and returns the new token.
func addCharacter() (*storage.Token, error) {
	ssoToken, err := sso.Authenticate(httpClient, scopes)
	if err != nil {
		return nil, err
	}
	character := storage.Character{
		ID:   ssoToken.CharacterID,
		Name: ssoToken.CharacterName,
	}
	if err = character.Save(); err != nil {
		return nil, err
	}
	token := storage.Token{
		AccessToken:  ssoToken.AccessToken,
		Character:    character,
		ExpiresAt:    ssoToken.ExpiresAt,
		RefreshToken: ssoToken.RefreshToken,
		TokenType:    ssoToken.TokenType,
	}
	if err = token.Save(); err != nil {
		return nil, err
	}
	return &token, nil
}
