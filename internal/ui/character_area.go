package ui

import (
	"database/sql"
	"example/evebuddy/internal/helper/humanize"
	"fmt"

	"fyne.io/fyne/v2"
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
	character.FetchAlliance()
	character.FetchFaction()
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
