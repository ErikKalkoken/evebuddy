package corporationui

import (
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
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

func makeCorporationActionLabel(id int64, name string, action func(o *app.EveEntity)) fyne.CanvasObject {
	o := &app.EveEntity{
		ID:       id,
		Name:     name,
		Category: app.EveEntityCorporation,
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

func makeSolarSystemLabel(o *app.EveSolarSystem, show func(o *app.EveEntity)) fyne.CanvasObject {
	if o == nil {
		return widget.NewLabel("?")
	}
	segs := slices.Concat(
		o.SecurityStatusRichText(),
		xwidget.RichTextSegmentsFromText(" ", widget.RichTextStyleInline),
		xwidget.RichTextSegmentsFromText(o.Name, widget.RichTextStyle{
			ColorName: theme.ColorNamePrimary,
		}))
	x := xwidget.NewTappableRichText(segs, func() {
		o := &app.EveEntity{
			ID:       o.ID,
			Name:     o.Name,
			Category: app.EveEntitySolarSystem,
		}
		show(o)
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
