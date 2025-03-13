package infowindow

import "fyne.io/fyne/v2"

type topLeftLayout struct{}

func (d *topLeftLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	w, h := float32(0), float32(0)
	for _, o := range objects {
		childSize := o.MinSize()
		if childSize.Width > w {
			w = childSize.Width
		}
		if childSize.Height > h {
			h = childSize.Height
		}
	}
	return fyne.NewSize(w, h)
}

func (d *topLeftLayout) Layout(objects []fyne.CanvasObject, containerSize fyne.Size) {
	pos := fyne.NewPos(0, 0)
	for _, o := range objects {
		size := o.MinSize()
		o.Resize(size)
		o.Move(pos)
	}
}
