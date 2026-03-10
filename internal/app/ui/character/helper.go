package character

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
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

func makeCharacterActionLabel(id int64, name string, action func(o *app.EveEntity)) fyne.CanvasObject {
	o := &app.EveEntity{
		ID:       id,
		Name:     name,
		Category: app.EveEntityCharacter,
	}
	return makeEveEntityActionLabel(o, action)
}

// makeEveEntityActionLabel returns a Hyperlink for existing entities or a placeholder label otherwise.
func makeEveEntityActionLabel(o *app.EveEntity, action func(o *app.EveEntity)) fyne.CanvasObject {
	if o == nil {
		return widget.NewLabel("-")
	}
	return makeLinkLabelWithWrap(o.Name, func() {
		action(o)
	})
}

// makeEveEntityActionLabel returns a Hyperlink for existing entities or a placeholder label otherwise.
func makeEveEntityActionLabel2(o optional.Optional[*app.EveEntity], action func(o *app.EveEntity)) fyne.CanvasObject {
	v, ok := o.Value()
	if !ok {
		return widget.NewLabel("-")
	}
	return makeLinkLabelWithWrap(v.Name, func() {
		action(v)
	})
}

func makeLocationLabel2(o optional.Optional[*app.EveLocationShort], show func(int64)) fyne.CanvasObject {
	el, ok := o.Value()
	if !ok {
		return widget.NewLabel("?")
	}
	x := makeLinkLabelWithWrap(el.DisplayName(), func() {
		show(el.ID)
	})
	x.Wrapping = fyne.TextWrapWord
	return x
}

// TODO: Remove this helper

// makeTopText makes the content for the top label of a gui element.
func makeTopText(characterID int64, hasData bool, err error, create func() (string, widget.Importance)) (string, widget.Importance) {
	if err != nil {
		return "ERROR: " + app.ErrorDisplay(err), widget.DangerImportance
	}
	if characterID == 0 {
		return "No entity", widget.LowImportance
	}
	if !hasData {
		return "No data", widget.WarningImportance
	}
	if create == nil {
		return "", widget.MediumImportance
	}
	return create()
}
