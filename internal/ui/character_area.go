package ui

import (
	"example/evebuddy/internal/api/images"
	"example/evebuddy/internal/helper/humanize"
	"example/evebuddy/internal/repository"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
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
	icons := container.NewHBox()
	c.items.Add(icons)
	c.items.Add(widget.NewSeparator())
	var rows = []struct {
		label string
		value string
	}{
		{"Name", character.Name},
		{"Corporation", character.Corporation.Name},
		{"Alliance", stringOrDefault(character.Alliance.Name, "-")},
		{"Faction", stringOrDefault(character.Faction.Name, "-")},
		{"Birthday", character.Birthday.Format(myDateTime)},
		{"Gender", character.Gender},
		{"Security Status", fmt.Sprintf("%.1f", character.SecurityStatus)},
		{"Skill Points", numberOrDefault(character.SkillPoints, "-")},
		{"Wallet Balance", numberOrDefault(character.WalletBalance, "-")},
	}
	for _, row := range rows {
		label := widget.NewLabel(row.label)
		label.TextStyle = fyne.TextStyle{Bold: true}
		value := widget.NewLabel(row.value)
		c.items.Add(container.NewHBox(label, layout.NewSpacer(), value))
		c.items.Add(widget.NewSeparator())
	}
	err := updateIcons(icons, character)
	if err != nil {
		slog.Error(err.Error())
	}
}

func updateIcons(icons *fyne.Container, c *repository.Character) error {
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
