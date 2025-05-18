package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// characterBiography shows the attributes for the current character.
type characterBiography struct {
	widget.BaseWidget

	body *widget.Label
	u    *baseUI
}

func newCharacterBiography(u *baseUI) *characterBiography {
	text := widget.NewLabel("")
	text.Wrapping = fyne.TextWrapWord
	w := &characterBiography{
		body: text,
		u:    u,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (a *characterBiography) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewVScroll(a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *characterBiography) update() {
	character := a.u.currentCharacter()
	if character == nil || character.EveCharacter == nil {
		fyne.Do(func() {
			a.body.Text = "Waiting for character data to be loaded..."
			a.body.Importance = widget.WarningImportance
			a.body.Refresh()
		})
	} else {
		fyne.Do(func() {
			a.body.Text = character.EveCharacter.DescriptionPlain()
			a.body.Importance = widget.MediumImportance
			a.body.Refresh()
		})
	}
}
