package gui

import (
	"example/esiapp/internal/api/images"
	"example/esiapp/internal/model"
	"example/esiapp/internal/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
)

const defaultIconSize = 64

type characters struct {
	content       *fyne.Container
	folders       *folders
	esiApp        *eveApp
	currentCharID int32
}

func (c *characters) update(charID int32) {
	btnSwitch, err := c.makeSwitchButton(charID)
	if err != nil {
		panic(err)
	}
	character := makeCharacter(charID)
	c.content.RemoveAll()
	c.content.Add(character)
	c.content.Add(layout.NewSpacer())
	c.content.Add(btnSwitch)
	c.content.Refresh()
	c.folders.update(charID)
	c.folders.updateMails()
	c.currentCharID = charID
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

func makeCharacter(charID int32) *fyne.Container {
	char, err := model.FetchCharacter(charID)
	var charName, corpName string
	var charURI, corpURI fyne.URI
	if err != nil {
		charName = "No characters"
		charURI, _ = images.CharacterPortraitURL(images.PlaceholderCharacterID, defaultIconSize)
		corpURI, _ = images.CorporationLogoURL(images.PlaceholderCharacterID, defaultIconSize)
	} else {
		charName = char.Name
		corpName = char.Corporation.Name
		charURI = char.PortraitURL(defaultIconSize)
		corpURI = char.Corporation.ImageURL(defaultIconSize)
	}
	charImage := canvas.NewImageFromURI(charURI)
	charImage.FillMode = canvas.ImageFillOriginal
	corpImage := canvas.NewImageFromURI(corpURI)
	corpImage.FillMode = canvas.ImageFillOriginal
	color := theme.ForegroundColor()
	character := canvas.NewText(charName, color)
	character.TextStyle = fyne.TextStyle{Bold: true}
	corporation := canvas.NewText(corpName, color)
	content := container.NewHBox(charImage, corpImage, container.NewVBox(character, corporation))
	return content
}

func (e *eveApp) newCharacters(f *folders) *characters {
	c := characters{esiApp: e, folders: f}
	c.content = container.NewHBox()
	return &c
}
