package widgets

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type columnsLayout struct {
	widths []float32
}

func NewColumnsLayout(widths ...float32) columnsLayout {
	if len(widths) == 0 {
		panic("Need to define at least one width")
	}
	l := columnsLayout{
		widths: widths,
	}
	return l
}

func (l columnsLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	w, h := float32(0), float32(0)
	if len(objects) > 0 {
		h = objects[0].MinSize().Height
	}
	for i, x := range l.widths {
		w += x
		if i < len(l.widths) {
			w += theme.Padding()
		}
	}
	s := fyne.NewSize(w, h)
	return s
}

func (l columnsLayout) Layout(objects []fyne.CanvasObject, containerSize fyne.Size) {
	if len(l.widths) < len(objects) {
		panic(fmt.Sprintf("not enough columns defined. Need: %d, Have: %d", len(objects), len(l.widths))) // FIXME
	}
	var lastX float32
	pos := fyne.NewPos(0, 0)
	padding := theme.Padding()
	for i, o := range objects {
		size := o.MinSize()
		var w float32
		if i < len(objects)-1 || containerSize.Width < 0 {
			w = l.widths[i]
		} else {
			w = max(containerSize.Width-pos.X-padding, l.widths[i])
		}
		o.Resize(fyne.Size{Width: w, Height: size.Height})
		o.Move(pos)
		var x float32
		if len(l.widths) > i {
			x = w
			lastX = x
		} else {
			x = lastX
		}
		pos = pos.AddXY(x+padding, 0)
	}
}
