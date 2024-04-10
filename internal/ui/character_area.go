package ui

import (
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
	var rows = []struct {
		label string
		value string
	}{
		{"Name", character.Name},
		{"Corporation", character.Corporation.Name},
		{"Alliance", character.Corporation.Name},
		{"Faction", character.Corporation.Name},
		{"Skill points", "1.000.000"},
		{"Wallet", "999 ISK"},
	}
	for _, row := range rows {
		label := widget.NewLabel(row.label)
		label.TextStyle = fyne.TextStyle{Bold: true}
		value := widget.NewLabel(row.value)
		c.items.Add(container.NewHBox(label, layout.NewSpacer(), value))
		c.items.Add(widget.NewSeparator())
	}
}
