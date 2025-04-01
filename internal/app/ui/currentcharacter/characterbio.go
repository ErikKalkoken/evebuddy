package currentcharacter

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
)

// CharacterBiography shows the attributes for the current character.
type CharacterBiography struct {
	widget.BaseWidget

	text *widget.Label
	top  *widget.Label
	u    app.UI
}

func NewCharacterBiography(u app.UI) *CharacterBiography {
	text := widget.NewLabel("")
	text.Wrapping = fyne.TextWrapWord
	w := &CharacterBiography{
		text: text,
		top:  appwidget.MakeTopLabel(),
		u:    u,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (a *CharacterBiography) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(a.top, nil, nil, nil, container.NewVScroll(a.text))
	return widget.NewSimpleRenderer(c)
}

func (a *CharacterBiography) Update() {
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
