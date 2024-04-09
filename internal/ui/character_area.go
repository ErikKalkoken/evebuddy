package ui

import (
	"example/evebuddy/internal/api/images"
	"example/evebuddy/internal/model"
	"example/evebuddy/internal/widgets"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const defaultIconSize = 64

// characterArea is the UI area that shows the active character
type characterArea struct {
	content *fyne.Container
	ui      *ui
}

func (u *ui) NewCharacterArea() *characterArea {
	c := characterArea{ui: u}
	c.content = container.NewVBox(container.NewHBox(), widget.NewSeparator())
	return &c
}

func (c *characterArea) Redraw() {
	content := c.content.Objects[0].(*fyne.Container)
	content.RemoveAll()
	badge := c.makeCharacterBadge()
	content.Add(badge)
	content.Add(layout.NewSpacer())
	button, err := c.makeSwitchButton()
	if err != nil {
		slog.Error("Failed to make switch button", "error", "err")
	} else {
		content.Add(button)
	}
	content.Refresh()
	c.ui.folderArea.Redraw()
	character := c.ui.CurrentChar()
	c.ui.statusArea.clearInfo()
	if character != nil {
		if character.MailUpdatedAt.Before(time.Now().Add(time.Second * settingMailUpdateTimeoutSeconds * -1)) {
			c.ui.folderArea.UpdateMails()
		}
	}
}

func (c *characterArea) makeCharacterBadge() *fyne.Container {
	char, err := model.FetchCharacter(c.ui.CurrentCharID())
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

func (c *characterArea) makeSwitchButton() (*widgets.ContextMenuButton, error) {
	menu, ok, err := c.makeSwitchMenu(c.ui.CurrentCharID())
	if err != nil {
		return nil, err
	}
	button := widgets.NewContextMenuButton("Switch Character", menu)
	if !ok {
		button.Disable()
	}
	return button, nil
}

func (c *characterArea) makeSwitchMenu(charID int32) (*fyne.Menu, bool, error) {
	menu := fyne.NewMenu("")
	characters, err := model.FetchAllCharacters()
	if err != nil {
		return nil, false, err
	}
	if len(characters) == 0 {
		return menu, false, nil
	}
	var items []*fyne.MenuItem
	for _, char := range characters {
		item := fyne.NewMenuItem(char.Name, func() {
			c.ui.SetCurrentCharacter(&char)
			c.Redraw()
		})
		if char.ID == charID {
			item.Disabled = true
		}
		items = append(items, item)
	}
	if charID != 0 && len(characters) < 2 {
		return menu, false, nil
	}
	menu.Items = items
	return menu, true, nil
}
