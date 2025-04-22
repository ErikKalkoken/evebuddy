package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// CharacterBiography shows the attributes for the current character.
type CharacterBiography struct {
	widget.BaseWidget

	body *widget.Label
	u    *BaseUI
}

func NewCharacterBiography(u *BaseUI) *CharacterBiography {
	text := widget.NewLabel("")
	text.Wrapping = fyne.TextWrapWord
	w := &CharacterBiography{
		body: text,
		u:    u,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (a *CharacterBiography) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewVScroll(a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *CharacterBiography) update() {
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
