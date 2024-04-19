package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"

	"example/evebuddy/internal/api/images"
	"example/evebuddy/internal/helper/humanize"
	"example/evebuddy/internal/model"
)

// characterArea is the UI area that shows the character sheet
type characterArea struct {
	content fyne.CanvasObject
	items   *fyne.Container
	ui      *ui
}

func (u *ui) NewCharacterArea() *characterArea {
	items := container.NewVBox()
	c := characterArea{ui: u, content: container.NewScroll(items), items: items}
	return &c
}

func (c *characterArea) Redraw() {
	c.items.RemoveAll()
	character := c.ui.CurrentChar()
	if character == nil {
		return
	}

	u, _ := images.CharacterPortraitURL(character.ID, 128)
	characterIcon := canvas.NewImageFromURI(u)
	characterIcon.FillMode = canvas.ImageFillOriginal
	c.items.Add(container.NewHBox(characterIcon, layout.NewSpacer()))
	fg := theme.ForegroundColor()
	x := canvas.NewText(character.Name, fg)
	x.TextSize = theme.TextHeadingSize()
	c.items.Add(container.NewPadded(x))
	var rows = []struct {
		label string
		value string
	}{
		{"Born", character.Birthday.Format(myDateTime)},
		{"Wallet Balance", numberOrDefault(character.WalletBalance, "-")},
		{"Security Status", fmt.Sprintf("%.1f", character.SecurityStatus)},
		{"Home Station", "?"},
		{"Corporation", character.Corporation.Name},
		{"Alliance", stringOrDefault(character.Alliance.Name, "-")},
		{"Faction", stringOrDefault(character.Faction.Name, "-")},
		{"Gender", character.Gender},
		{"Skill Points", numberOrDefault(character.SkillPoints, "-")},
	}
	form := container.New(layout.NewFormLayout())
	for _, row := range rows {
		label := canvas.NewText(row.label+":", fg)
		label.TextStyle = fyne.TextStyle{Bold: true}
		value := canvas.NewText(row.value, fg)
		form.Add(container.NewPadded(label))
		form.Add(container.NewPadded(value))
	}
	c.items.Add(form)
	// err := updateIcons(icons, character)
	// if err != nil {
	// 	slog.Error(err.Error())
	// }
}

func updateIcons(icons *fyne.Container, c *model.Character) error {
	u, err := images.CharacterPortraitURL(c.ID, 128)
	if err != nil {
		return err
	}
	character := canvas.NewImageFromURI(u)
	character.FillMode = canvas.ImageFillOriginal
	icons.Add(layout.NewSpacer())
	icons.Add(character)

	u, err = c.Corporation.IconURL(128)
	if err != nil {
		return err
	}
	corp := canvas.NewImageFromURI(u)
	corp.FillMode = canvas.ImageFillOriginal
	icons.Add(corp)

	if c.HasAlliance() {
		u, err = c.Alliance.IconURL(128)
		if err != nil {
			return err
		}
		image := canvas.NewImageFromURI(u)
		image.FillMode = canvas.ImageFillOriginal
		icons.Add(image)
	}

	if c.HasFaction() {
		u, err = c.Faction.IconURL(128)
		if err != nil {
			return err
		}
		image := canvas.NewImageFromURI(u)
		image.FillMode = canvas.ImageFillOriginal
		icons.Add(image)
	}

	icons.Add(layout.NewSpacer())
	return nil
}

func stringOrDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

func numberOrDefault[T int | float64](v T, d string) string {
	if v == 0 {
		return d
	}
	return humanize.Number(float64(v), 1)
}
