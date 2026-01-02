package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// TappableImage is widget which shows an image and runs a function when tapped.
type TappableImage struct {
	widget.BaseWidget

	// The function that is called when the label is tapped.
	OnTapped func()

	image   *canvas.Image
	hovered bool
	menu    *fyne.Menu
	pos     fyne.Position // current mouse position
}

var _ fyne.Tappable = (*TappableImage)(nil)
var _ desktop.Hoverable = (*TappableImage)(nil)

// NewTappableImageWithMenu returns a new instance of a [TappableImage] widget with a context menu.
func NewTappableImageWithMenu(res fyne.Resource, menu *fyne.Menu) *TappableImage {
	w := newTappableImage(res, nil)
	w.menu = menu
	w.OnTapped = func() {
		if len(w.menu.Items) == 0 {
			return
		}
		c := fyne.CurrentApp().Driver().CanvasForObject(w)
		m := widget.NewPopUpMenu(w.menu, c)
		m.ShowAtPosition(w.pos)
	}
	return w
}

// NewTappableImage returns a new instance of a [TappableImage] widget.
func NewTappableImage(res fyne.Resource, tapped func()) *TappableImage {
	return newTappableImage(res, tapped)
}

func newTappableImage(res fyne.Resource, tapped func()) *TappableImage {
	ti := &TappableImage{OnTapped: tapped, image: canvas.NewImageFromResource(res)}
	ti.image.CornerRadius = theme.InputRadiusSize()
	ti.ExtendBaseWidget(ti)
	return ti
}

// SetCornerRadius sets corner radius.
func (w *TappableImage) SetCornerRadius(cornerRadius float32) {
	w.image.CornerRadius = cornerRadius
}

// SetFillMode sets the fill mode of the image.
func (w *TappableImage) SetFillMode(fillMode canvas.ImageFill) {
	w.image.FillMode = fillMode
}

// SetMinSize sets the minimum size of the image.
func (w *TappableImage) SetMinSize(size fyne.Size) {
	w.image.SetMinSize(size)
}

// SetResource sets the resource of the image.
func (w *TappableImage) SetResource(r fyne.Resource) {
	w.image.Resource = r
	w.image.Refresh()
}

// SetMenuItems replaces the menu items.
func (w *TappableImage) SetMenuItems(menuItems []*fyne.MenuItem) {
	if w.menu == nil {
		return
	}
	w.menu.Items = menuItems
	w.menu.Refresh()
}

func (w *TappableImage) Tapped(pe *fyne.PointEvent) {
	w.pos = pe.AbsolutePosition
	if w.OnTapped != nil {
		w.OnTapped()
	}
}

func (w *TappableImage) TappedSecondary(_ *fyne.PointEvent) {
}

// Cursor returns the cursor type of this widget
func (w *TappableImage) Cursor() desktop.Cursor {
	if w.hovered {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

// MouseIn is a hook that is called if the mouse pointer enters the element.
func (w *TappableImage) MouseIn(e *desktop.MouseEvent) {
	w.hovered = true
}

func (w *TappableImage) MouseMoved(me *desktop.MouseEvent) {
	w.pos = me.AbsolutePosition
}

// MouseOut is a hook that is called if the mouse pointer leaves the element.
func (w *TappableImage) MouseOut() {
	w.hovered = false
}

func (w *TappableImage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewPadded(w.image))
}
