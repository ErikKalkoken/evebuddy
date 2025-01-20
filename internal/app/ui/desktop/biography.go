package desktop

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/evehtml"
)

// biographyArea is the UI area that shows the skillqueue
type biographyArea struct {
	content fyne.CanvasObject
	text    *widget.Label
	u       *DesktopUI
}

func (u *DesktopUI) newBiographyArea() *biographyArea {
	a := &biographyArea{u: u, text: widget.NewLabel("")}
	a.text.Wrapping = fyne.TextWrapBreak
	a.content = a.text
	return a
}

func (a *biographyArea) refresh() {
	var s string
	c := a.u.CurrentCharacter()
	if c == nil {
		s = ""
	} else {
		s = evehtml.ToPlain(c.EveCharacter.Description)
	}
	a.text.SetText(s)
}
