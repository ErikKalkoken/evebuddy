package ui

import (
	"fmt"
	"image/color"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type loadFuncAsync func(int64, int, func(fyne.Resource))

func newLabelWithWrapping() *widget.Label {
	l := widget.NewLabel("")
	l.Wrapping = fyne.TextWrapWord
	return l
}

func newLabelWithTruncation() *widget.Label {
	l := widget.NewLabel("")
	l.Truncation = fyne.TextTruncateEllipsis
	return l
}

// formatISKAmount returns a formatted ISK amount.
// This format is mainly used in detail windows.
func formatISKAmount(v float64) string {
	t := humanize.FormatFloat(app.FloatFormat, v) + " ISK"
	if math.Abs(v) > 999 {
		t += fmt.Sprintf(" (%s)", ihumanize.NumberF(v, 2))
	}
	return t
}

func colorISKAmount(amount optional.Optional[float64]) fyne.ThemeColorName {
	var color fyne.ThemeColorName
	if v, ok := amount.Value(); !ok {
		color = theme.ColorNameDisabled
	} else if v < 0 {
		color = theme.ColorNameError
	} else if v > 0 {
		color = theme.ColorNameSuccess
	} else {
		color = theme.ColorNameForeground
	}
	return color
}

func importanceISKAmount(amount optional.Optional[float64]) widget.Importance {
	if v, ok := amount.Value(); !ok {
		return widget.LowImportance
	} else if v > 0 {
		return widget.SuccessImportance
	} else if v < 0 {
		return widget.DangerImportance
	}
	return widget.MediumImportance
}

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

func makeLabelWithWrap(s string) *widget.Label {
	l := widget.NewLabel(s)
	l.Wrapping = fyne.TextWrapWord
	return l
}

func makeBoolLabel(v bool) *widget.Label {
	if v {
		l := widget.NewLabel("Yes")
		l.Importance = widget.SuccessImportance
		return l
	}
	l := widget.NewLabel("No")
	l.Importance = widget.DangerImportance
	return l
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

func newSpacer(s fyne.Size) fyne.CanvasObject {
	w := canvas.NewRectangle(color.Transparent)
	w.SetMinSize(s)
	return w
}

func newStandardSpacer() fyne.CanvasObject {
	return newSpacer(fyne.NewSquareSize(theme.Padding()))
}

// characterIDOrZero returns the ID of a character or 0 if the c does not exist.
func characterIDOrZero(c *app.Character) int64 {
	if c == nil {
		return 0
	}
	return c.ID
}

// corporationIDOrZero returns the ID of a corporation or 0 if the c does not exist.
func corporationIDOrZero(c *app.Corporation) int64 {
	if c == nil {
		return 0
	}
	return c.ID
}

// corporationNameOrZero returns the name of a corporation or "" if the c does not exist.
func corporationNameOrZero(c *app.Corporation) string {
	if c == nil || c.EveCorporation == nil {
		return ""
	}
	return c.EveCorporation.Name
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
