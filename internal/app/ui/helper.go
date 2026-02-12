package ui

import (
	"fmt"
	"image/color"
	"math"
	"math/rand/v2"
	"slices"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"

	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

// makeGridOrList makes and returns a GridWrap on desktop and a List on mobile.
//
// This allows the grid items to render nicely as list on mobile and also enable truncation.
func makeGridOrList(isMobile bool, length func() int, makeCreateItem func(trunc fyne.TextTruncation) func() fyne.CanvasObject, updateItem func(id int, co fyne.CanvasObject), makeOnSelected func(unselectAll func()) func(int)) fyne.CanvasObject {
	var w fyne.CanvasObject
	if isMobile {
		w = widget.NewList(length, makeCreateItem(fyne.TextTruncateEllipsis), updateItem)
		l := w.(*widget.List)
		l.OnSelected = makeOnSelected(func() {
			l.UnselectAll()
		})
	} else {
		w = widget.NewGridWrap(length, makeCreateItem(fyne.TextTruncateOff), updateItem)
		g := w.(*widget.GridWrap)
		g.OnSelected = makeOnSelected(func() {
			g.UnselectAll()
		})
	}
	return w
}

// makeTopLabel returns a new empty label meant for the top bar on a screen.
func makeTopLabel() *widget.Label {
	l := widget.NewLabel("")
	l.Wrapping = fyne.TextWrapWord
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

func makeSolarSystemLabel(o *app.EveSolarSystem, show func(o *app.EveEntity)) fyne.CanvasObject {
	if o == nil {
		return widget.NewLabel("?")
	}
	segs := slices.Concat(
		o.SecurityStatusRichText(),
		iwidget.RichTextSegmentsFromText(" ", widget.RichTextStyleInline),
		iwidget.RichTextSegmentsFromText(o.Name, widget.RichTextStyle{
			ColorName: theme.ColorNamePrimary,
		}))
	x := iwidget.NewTappableRichText(segs, func() {
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

// uniqueID returns a pseudo unique ID.
func uniqueID() string {
	currentTime := time.Now().UnixNano()
	randomNumber := rand.Uint64()
	return fmt.Sprintf("%d-%d", currentTime, randomNumber)
}
