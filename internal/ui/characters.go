package ui

import (
	"example/esiapp/internal/core"
	"example/esiapp/internal/esi"
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

type characters struct {
	container *fyne.Container
	esiApp    *esiApp
}

func (c *characters) update(charID int32) {
	buttonAdd := c.makeManageButton()
	image, name := makeCharacter(charID)
	c.container.RemoveAll()
	c.container.Add(image)
	c.container.Add(name)
	c.container.Add(layout.NewSpacer())
	c.container.Add(buttonAdd)
	c.container.Refresh()
}

func (c *characters) makeManageButton() *contextMenuButton {
	shareItem := c.makeShareItem()
	buttonAdd := newContextMenuButton(
		"Manage Characters", fyne.NewMenu("",
			fyne.NewMenuItem("Add Character", func() {
				info := dialog.NewInformation(
					"Add Character",
					"Please follow instructions in your browser to add a new character.",
					c.esiApp.main,
				)
				info.Show()
				t, err := core.AddCharacter()
				if err != nil {
					log.Printf("Failed to add a new character: %v", err)
				} else {
					c.update(t.CharacterID)
				}
			}),
			shareItem,
		))
	return buttonAdd
}

func (c *characters) makeShareItem() *fyne.MenuItem {
	chars, err := storage.FetchAllCharacters()
	if err != nil {
		log.Fatal(err)
	}
	shareItem := fyne.NewMenuItem("Switch character", nil)

	var items []*fyne.MenuItem
	for _, char := range chars {
		item := fyne.NewMenuItem(char.Name, func() {
			c.update(char.ID)
		})
		items = append(items, item)
	}
	shareItem.ChildMenu = fyne.NewMenu("", items...)
	return shareItem
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

func (e *esiApp) newCharacters() *characters {
	c := characters{esiApp: e}
	c.container = container.NewHBox()
	return &c
}
