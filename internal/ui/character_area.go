package ui

import (
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
		{"Alliance", valueOrDefault(character.Alliance.Name, "-")},
		{"Faction", valueOrDefault(character.Faction.Name, "-")},
		{"Birthday", character.Birthday.Format(myDateTime)},
		{"Gender", character.Gender},
		{"Security Status", fmt.Sprintf("%.2f", character.SecurityStatus)},
	}
	for _, row := range rows {
		label := widget.NewLabel(row.label)
		label.TextStyle = fyne.TextStyle{Bold: true}
		value := widget.NewLabel(row.value)
		c.items.Add(container.NewHBox(label, layout.NewSpacer(), value))
		c.items.Add(widget.NewSeparator())
	}
}

func valueOrDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}
