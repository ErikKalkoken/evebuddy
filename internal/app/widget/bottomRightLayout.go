package widget

import "fyne.io/fyne/v2"

type bottomRightLayout struct{}

func (d *bottomRightLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
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

func (d *bottomRightLayout) Layout(objects []fyne.CanvasObject, containerSize fyne.Size) {
	pos := fyne.NewPos(containerSize.Width, containerSize.Height)
	for _, o := range objects {
		size := o.MinSize()
		o.Resize(size)
		o.Move(pos.SubtractXY(size.Width, size.Height))
	}
}
