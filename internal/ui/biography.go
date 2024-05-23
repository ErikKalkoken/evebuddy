package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/eveonline/converter"
)

// biographyArea is the UI area that shows the skillqueue
type biographyArea struct {
	content fyne.CanvasObject
	text    *widget.Label
	ui      *ui
}

func (u *ui) NewBiographyArea() *biographyArea {
	a := &biographyArea{ui: u, text: widget.NewLabel("")}
	a.text.Wrapping = fyne.TextWrapBreak
	a.content = container.NewVScroll(a.text)
	return a
}

func (a *biographyArea) Refresh() {
	var s string
	c := a.ui.CurrentChar()
	if c == nil {
		s = ""
	} else {
		s = converter.EveHTMLToPlain(c.EveCharacter.Description)
	}
	a.text.SetText(s)
}
