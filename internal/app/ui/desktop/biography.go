package desktop

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/evehtml"
)

// BiographyArea is the UI area that shows the skillqueue
type BiographyArea struct {
	Content fyne.CanvasObject
	text    *widget.Label
	u       *DesktopUI
}

func (u *DesktopUI) NewBiographyArea() *BiographyArea {
	a := &BiographyArea{u: u, text: widget.NewLabel("")}
	a.text.Wrapping = fyne.TextWrapBreak
	a.Content = a.text
	return a
}

func (a *BiographyArea) refresh() {
	var s string
	c := a.u.CurrentCharacter()
	if c == nil {
		s = ""
	} else {
		s = evehtml.ToPlain(c.EveCharacter.Description)
	}
	a.text.SetText(s)
}
