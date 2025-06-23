package ui

import (
	"fmt"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/dustin/go-humanize"

	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
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

func formatISKAmount(v float64) string {
	t := humanize.Commaf(v) + " ISK"
	if math.Abs(v) > 999 {
		t += fmt.Sprintf(" (%s)", ihumanize.Number(v, 2))
	}
	return t
}

func importanceISKAmount(v float64) widget.Importance {
	if v > 0 {
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

func makeOwnerActionLabel(id int32, name string, action func(o *app.EveEntity)) fyne.CanvasObject {
	o := &app.EveEntity{
		ID:       id,
		Name:     name,
		Category: app.EveEntityCharacter,
	}
	return makeEveEntityActionLabel(o, action)
}

func makeEveEntityActionLabel(o *app.EveEntity, action func(o *app.EveEntity)) fyne.CanvasObject {
	if o == nil {
		return widget.NewLabel("-")
	}
	return makeLinkLabelWithWrap(o.Name, func() {
		action(o)
	})
}

func makeLabelWithWrap(s string) *widget.Label {
	l := widget.NewLabel(s)
	l.Wrapping = fyne.TextWrapWord
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

// setDetailWindow sets the content of a window to create a "detail window".
// Detail windows are used to show more information about objects in data lists.
func setDetailWindow(title string, content fyne.CanvasObject, window fyne.Window) {
	setDetailWindowWithSize(title, fyne.NewSize(600, 500), content, window)
}

func setDetailWindowWithSize(title string, minSize fyne.Size, content fyne.CanvasObject, w fyne.Window) {
	t := widget.NewLabel(title)
	t.SizeName = theme.SizeNameSubHeadingText
	top := container.NewVBox(t, widget.NewSeparator())
	vs := container.NewVScroll(content)
	vs.SetMinSize(minSize)
	c := container.NewBorder(
		top,
		nil,
		nil,
		nil,
		vs,
	)
	c.Refresh()
	w.SetContent(container.NewPadded(c))
}
