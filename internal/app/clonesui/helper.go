package clonesui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

type loadFuncAsync func(int64, int, func(fyne.Resource))

func makeLinkLabelWithWrap(text string, action func()) *widget.Hyperlink {
	x := makeLinkLabel(text, action)
	x.Wrapping = fyne.TextWrapWord
	return x
}

func makeLinkLabel(text string, action func()) *widget.Hyperlink {
	x := widget.NewHyperlink(text, nil)
	x.OnTapped = action
	return x
}

func makeLocationLabel(o *app.EveLocationShort, show func(int64)) fyne.CanvasObject {
	if o == nil {
		return widget.NewLabel("?")
	}
	x := makeLinkLabelWithWrap(o.DisplayName(), func() {
		show(o.ID)
	})
	x.Wrapping = fyne.TextWrapWord
	return x
}

// TODO: Remove this helper

// makeTopText makes the content for the top label of a gui element.
func makeTopText(characterID int64, hasData bool, err error, make func() (string, widget.Importance)) (string, widget.Importance) {
	if err != nil {
		return "ERROR: " + app.ErrorDisplay(err), widget.DangerImportance
	}
	if characterID == 0 {
		return "No entity", widget.LowImportance
	}
	if !hasData {
		return "No data", widget.WarningImportance
	}
	if make == nil {
		return "", widget.MediumImportance
	}
	return make()
}
