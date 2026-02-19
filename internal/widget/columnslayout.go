package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type columnsLayoutByRatio struct {
	ratios []float32
}

// NewColumnsByRatio returns a new columns layout.
//
// Columns arranges all objects in a row, with each in their own column.
// The width of each column is given as ratio of it's container.
// Ratios are expected to be values between 0.0 and 1.0.
// If the sum of all ratios is below 1.0 another column will be added
// to fill the remaining space.
func NewColumnsByRatio(ratios ...float32) fyne.Layout {
	var total float32
	for _, r := range ratios {
		total += r
	}
	if total < 0 || total > 1 {
		panic("the sum of all ratios must be between 0 and 1")
	}
	if total < 1 {
		ratios = append(ratios, 1.0-total)
	}
	l := columnsLayoutByRatio{
		ratios: ratios,
	}
	return l
}

func (l columnsLayoutByRatio) MinSize(objects []fyne.CanvasObject) fyne.Size {
	wTotal, hTotal := float32(0), float32(0)
	for _, o := range objects {
		hTotal = fyne.Max(hTotal, o.MinSize().Height)
	}
	var w float32
	for i := range objects {
		if i < len(l.ratios) {
			w = l.ratios[i]
		}
		wTotal += w
		if i < len(l.ratios) {
			wTotal += theme.Padding()
		}
	}
	return fyne.NewSize(wTotal, hTotal)
}

func (l columnsLayoutByRatio) Layout(objects []fyne.CanvasObject, containerSize fyne.Size) {
	pos := fyne.NewPos(0, 0)
	padding := theme.Padding()
	var w float32
	for i, o := range objects {
		size := o.MinSize()
		if i < len(l.ratios) {
			ratio := l.ratios[i]
			w = fyne.Max(containerSize.Width*ratio, size.Width)
		}
		o.Resize(fyne.Size{Width: w, Height: size.Height})
		o.Move(pos)
		pos = pos.AddXY(w+padding, 0)
	}
}
