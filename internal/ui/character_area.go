package ui

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

// characterArea is the UI area showing the active character
type characterArea struct {
	content       *fyne.Container
	folderArea    *folderArea
	ui            *ui
	currentCharID int32
}

func (u *ui) NewCharacterArea(f *folderArea) *characterArea {
	c := characterArea{ui: u, folderArea: f}
	c.content = container.NewHBox()
	return &c
}

func (c *characterArea) Redraw(charID int32) {
	button, err := c.makeSwitchButton(charID)
	if err != nil {
		panic(err)
	}
	character := makeCharacter(charID)
	c.content.RemoveAll()
	c.content.Add(character)
	c.content.Add(layout.NewSpacer())
	c.content.Add(button)
	c.content.Refresh()
	c.folderArea.Redraw(charID)
	c.folderArea.UpdateMails()
	c.currentCharID = charID
}

func (c *characterArea) makeSwitchButton(charID int32) (*widgets.ContextMenuButton, error) {
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

func (c *characterArea) makeSwitchMenu(charID int32) (*fyne.Menu, bool, error) {
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
			c.Redraw(char.ID)
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
		charName = "No character"
		charURI, _ = images.CharacterPortraitURL(images.PlaceholderCharacterID, defaultIconSize)
		corpURI, _ = images.CorporationLogoURL(images.PlaceholderCorporationID, defaultIconSize)
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
	names := container.NewVBox(character, corporation)
	content := container.NewHBox(charImage, corpImage, names)
	return content
}
