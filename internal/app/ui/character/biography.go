package character

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
)

// Biography shows the attributes for the current character.
type Biography struct {
	widget.BaseWidget

	text *widget.Label
	top  *widget.Label
	u    app.UI
}

func NewBiography(u app.UI) *Biography {
	text := widget.NewLabel("")
	text.Wrapping = fyne.TextWrapWord
	w := &Biography{
		text: text,
		top:  appwidget.MakeTopLabel(),
		u:    u,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (a *Biography) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(a.top, nil, nil, nil, container.NewVScroll(a.text))
	return widget.NewSimpleRenderer(c)
}

func (a *Biography) Update() {
	var t string
	var i widget.Importance

	c := a.u.CurrentCharacter()
	if c == nil {
		t = "Waiting for character data to be loaded..."
		i = widget.WarningImportance
		a.text.SetText("")
	} else {
		a.text.SetText(c.EveCharacter.DescriptionPlain())
	}
	a.top.Text = t
	a.top.Importance = i
	a.top.Refresh()
}
