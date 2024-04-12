package ui

import (
	"database/sql"
	"example/evebuddy/internal/api/images"
	"example/evebuddy/internal/helper/humanize"
	"example/evebuddy/internal/model"
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
	err := character.GetAlliance()
	if err != nil {
		slog.Error(err.Error())
		return
	}
	err = character.GetFaction()
	if err != nil {
		slog.Error(err.Error())
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
		{"Skill Points", int64OrDefault(character.SkillPoints, "-")},
		{"Wallet Balance", float64OrDefault(character.WalletBalance, "-")},
	}
	for _, row := range rows {
		label := widget.NewLabel(row.label)
		label.TextStyle = fyne.TextStyle{Bold: true}
		value := widget.NewLabel(row.value)
		c.items.Add(container.NewHBox(label, layout.NewSpacer(), value))
		c.items.Add(widget.NewSeparator())
	}
	err = updateIcons(icons, character)
	if err != nil {
		slog.Error(err.Error())
	}
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

	u, err = images.CorporationLogoURL(c.CorporationID, 128)
	if err != nil {
		return err
	}
	corp := canvas.NewImageFromURI(u)
	corp.FillMode = canvas.ImageFillOriginal
	icons.Add(corp)

	if c.AllianceID.Valid {
		u, err = images.AllianceLogoURL(c.AllianceID.Int32, 128)
		if err != nil {
			return err
		}
		image := canvas.NewImageFromURI(u)
		image.FillMode = canvas.ImageFillOriginal
		icons.Add(image)
	}

	if c.FactionID.Valid {
		u, err = images.FactionLogoURL(c.FactionID.Int32, 128)
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

func float64OrDefault(v sql.NullFloat64, d string) string {
	if !v.Valid {
		return d
	}
	return humanize.Number(v.Float64, 1)
}

func int64OrDefault(v sql.NullInt64, d string) string {
	if !v.Valid {
		return d
	}
	return humanize.Number(float64(v.Int64), 1)
}
