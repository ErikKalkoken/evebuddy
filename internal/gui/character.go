package gui

import (
	"example/esiapp/internal/api/images"
	"example/esiapp/internal/model"
	"example/esiapp/internal/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

const defaultIconSize = 64

type characters struct {
	container *fyne.Container
	folders   *folders
	esiApp    *esiApp
}

func (c *characters) update(charID int32) {
	btnSwitch, err := c.makeSwitchButton(charID)
	if err != nil {
		panic(err)
	}
	image, name := makeCharacter(charID)
	c.container.RemoveAll()
	c.container.Add(image)
	c.container.Add(name)
	c.container.Add(layout.NewSpacer())
	c.container.Add(btnSwitch)
	c.container.Refresh()
	c.folders.update(charID)
	c.folders.updateMails()
}

func (c *characters) makeSwitchButton(charID int32) (*widgets.ContextMenuButton, error) {
	menu, ok, err := c.makeSwitchMenu(charID)
	if err != nil {
		return nil, err
	}
	b := widgets.NewContextMenuButton("Switch Character", menu)
	if !ok {
		b.Disable()
	}
	return b, nil
}

func (c *characters) makeSwitchMenu(charID int32) (*fyne.Menu, bool, error) {
	menu := fyne.NewMenu("")
	chars, err := model.FetchAllCharacters()
	if err != nil {
		return nil, false, err
	}
	if len(chars) == 0 {
		return menu, false, nil
	}
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
		return menu, false, nil
	}
	menu.Items = items
	return menu, true, nil
}

func makeCharacter(charID int32) (*canvas.Image, *widget.Label) {
	char, err := model.FetchCharacter(charID)
	var label string
	var uri fyne.URI
	if err != nil {
		label = "No characters"
		uri, _ = images.CharacterPortraitURL(images.PlaceholderCharacterID, defaultIconSize)
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
