package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// BiographyArea is the UI area that shows the skillqueue
type BiographyArea struct {
	Content fyne.CanvasObject
	text    *widget.Label
	u       *BaseUI
}

func NewBiographyArea(u *BaseUI) *BiographyArea {
	a := &BiographyArea{u: u, text: widget.NewLabel("")}
	a.text.Wrapping = fyne.TextWrapBreak
	a.Content = a.text
	return a
}

func (a *BiographyArea) Refresh() {
	var s string
	c := a.u.CurrentCharacter()
	if c == nil {
		s = ""
	} else {
		s = c.EveCharacter.DescriptionPlain()
	}
	a.text.SetText(s)
}
