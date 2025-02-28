package widget

import (
	"fmt"
	"image/color"
	"log/slog"
	"slices"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxlayout "github.com/ErikKalkoken/fyne-kx/layout"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icon"
	"github.com/ErikKalkoken/evebuddy/internal/fynetools"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

const (
	DefaultIconPixelSize = 64
)

// EveEntityEntry represents an entry widget for Eve Entity items.
type EveEntityEntry struct {
	widget.DisableableWidget

	Placeholder  string
	FallbackIcon fyne.Resource

	field       *canvas.Rectangle
	eis         app.EveImageService
	hovered     bool
	label       fyne.CanvasObject
	labelWidth  float32
	main        *fyne.Container
	mu          sync.Mutex
	placeholder *widget.RichText
	s           []*app.EveEntity
}

func NewEveEntityEntry(label fyne.CanvasObject, labelWidth float32, eis app.EveImageService) *EveEntityEntry {
	bg := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	bg.StrokeColor = theme.Color(theme.ColorNameInputBorder)
	bg.StrokeWidth = theme.Size(theme.SizeNameInputBorder)
	bg.CornerRadius = theme.Size(theme.SizeNameInputRadius)
	w := &EveEntityEntry{
		field:        bg,
		eis:          eis,
		FallbackIcon: icon.Questionmark32Png,
		label:        label,
		labelWidth:   labelWidth,
		main:         container.New(layout.NewCustomPaddedVBoxLayout(0)),
		placeholder: widget.NewRichText(&widget.TextSegment{
			Style: widget.RichTextStyle{ColorName: theme.ColorNamePlaceHolder},
		}),
		s: make([]*app.EveEntity, 0),
	}
	w.ExtendBaseWidget(w)
	return w
}

// Items returns the current list of EveEnties items.
func (w *EveEntityEntry) Items() []*app.EveEntity {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.s
}

// Set replaces the list of items.
func (w *EveEntityEntry) Set(s []*app.EveEntity) {
	w.mu.Lock()
	w.s = s
	w.mu.Unlock()
	w.Refresh()
}

func (w *EveEntityEntry) Add(ee *app.EveEntity) {
	added := func() bool {
		w.mu.Lock()
		defer w.mu.Unlock()
		for _, o := range w.s {
			if o.ID == ee.ID {
				return false
			}
		}
		w.s = append(w.s, ee)
		return true
	}()
	if added {
		w.Refresh()
	}
}

func (w *EveEntityEntry) Remove(id int32) {
	removed := func() bool {
		w.mu.Lock()
		defer w.mu.Unlock()
		for i, o := range w.s {
			if o.ID == id {
				w.s = slices.Delete(w.s, i, i+1)
				return true
			}
		}
		return false
	}()
	if removed {
		w.Refresh()
	}
}

func (w *EveEntityEntry) IsEmpty() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.s) == 0
}

func (w *EveEntityEntry) update() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.main.RemoveAll()
	colums := kxlayout.NewColumns(w.labelWidth)
	if len(w.s) == 0 {
		w.placeholder.Segments[0].(*widget.TextSegment).Text = w.Placeholder
		w.main.Add(container.New(colums, w.label, w.placeholder))
	} else {
		firstRow := true
		isDisabled := w.Disabled()
		for _, ee := range w.s {
			var label fyne.CanvasObject
			if firstRow {
				label = w.label
				firstRow = false
			} else {
				label = layout.NewSpacer()
			}
			badge := newEveEntityBadge(ee, w.eis, nil)
			if isDisabled {
				badge.Disable()
			} else {
				badge.OnTapped = func() {
					s := fmt.Sprintf("%s (%s)", ee.Name, ee.CategoryDisplay())
					name := fyne.NewMenuItem(s, nil)
					// name.Icon = fetchImage(w.eis, ee, w.FallbackIcon)
					name.Disabled = true
					remove := fyne.NewMenuItem("Remove", func() {
						w.Remove(ee.ID)
					})
					remove.Icon = theme.DeleteIcon()
					menu := fyne.NewMenu("",
						name,
						fyne.NewMenuItemSeparator(),
						remove,
					)
					pm := widget.NewPopUpMenu(menu, fyne.CurrentApp().Driver().CanvasForObject(badge))
					pm.ShowAtRelativePosition(fyne.Position{}, badge)
					// go func() {
					// 	title.Icon = fetchImage(w.eis, ee, w.FallbackIcon)
					// 	pm.Refresh()
					// }()
				}
			}
			w.main.Add(container.New(colums, label, badge))
		}
	}
}

func (w *EveEntityEntry) Refresh() {
	w.update()
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	w.field.FillColor = th.Color(theme.ColorNameInputBackground, v)
	w.field.StrokeColor = th.Color(theme.ColorNameInputBorder, v)
	w.BaseWidget.Refresh()
	w.main.Refresh()
	w.field.Refresh()
	w.placeholder.Refresh()
}

func (w *EveEntityEntry) MinSize() fyne.Size {
	th := w.Theme()
	innerPadding := th.Size(theme.SizeNameInnerPadding)
	textSize := th.Size(theme.SizeNameText)
	minSize := fyne.MeasureText("M", textSize, fyne.TextStyle{})
	minSize = minSize.Add(fyne.NewSquareSize(innerPadding))
	minSize = minSize.AddWidthHeight(innerPadding*2, innerPadding)
	return minSize.Max(w.BaseWidget.MinSize())
}

func (w *EveEntityEntry) CreateRenderer() fyne.WidgetRenderer {
	w.update()
	c := container.NewStack(w.field, w.main)
	return widget.NewSimpleRenderer(c)
}

type eveEntityBadge struct {
	widget.DisableableWidget

	FallbackIcon fyne.Resource
	OnTapped     func()

	ee      *app.EveEntity
	eis     app.EveImageService
	hovered bool
}

var _ fyne.Tappable = (*eveEntityBadge)(nil)
var _ desktop.Hoverable = (*eveEntityBadge)(nil)

func newEveEntityBadge(ee *app.EveEntity, eis app.EveImageService, onTapped func()) *eveEntityBadge {
	w := &eveEntityBadge{
		ee:           ee,
		eis:          eis,
		FallbackIcon: icon.Questionmark32Png,
		OnTapped:     onTapped,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *eveEntityBadge) CreateRenderer() fyne.WidgetRenderer {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	p := th.Size(theme.SizeNamePadding)

	name := widget.NewLabel(w.ee.Name)
	if w.Disabled() {
		name.Importance = widget.LowImportance
	}
	icon := iwidget.NewImageFromResource(w.FallbackIcon, fyne.NewSquareSize(th.Size(theme.SizeNameInlineIcon)))
	rect := canvas.NewRectangle(color.Transparent)
	rect.StrokeColor = th.Color(theme.ColorNameInputBorder, v)
	rect.StrokeWidth = 1
	rect.CornerRadius = 10
	c := container.New(layout.NewCustomPaddedLayout(p, p, 0, 0),
		container.NewHBox(
			container.NewStack(
				rect,
				container.New(layout.NewCustomPaddedLayout(-p, -p, 0, 0),
					container.NewHBox(
						container.New(layout.NewCustomPaddedLayout(0, 0, p, 0), icon), name,
					))),
		),
	)
	go func() {
		icon.Resource = fetchEveEntityAvatar(w.eis, w.ee, w.FallbackIcon)
		icon.Refresh()
	}()
	return widget.NewSimpleRenderer(c)
}

func (w *eveEntityBadge) Tapped(_ *fyne.PointEvent) {
	if w.Disabled() {
		return
	}
	if w.OnTapped != nil {
		w.OnTapped()
	}
}

func (w *eveEntityBadge) TappedSecondary(_ *fyne.PointEvent) {
}

// Cursor returns the cursor type of this widget
func (w *eveEntityBadge) Cursor() desktop.Cursor {
	if !w.Disabled() && w.hovered {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

// MouseIn is a hook that is called if the mouse pointer enters the element.
func (w *eveEntityBadge) MouseIn(e *desktop.MouseEvent) {
	if w.Disabled() {
		return
	}
	w.hovered = true
}

func (w *eveEntityBadge) MouseMoved(*desktop.MouseEvent) {
	// needed to satisfy the interface only
}

// MouseOut is a hook that is called if the mouse pointer leaves the element.
func (w *eveEntityBadge) MouseOut() {
	if w.Disabled() {
		return
	}
	w.hovered = false
}

func fetchEveEntityAvatar(eis app.EveImageService, ee *app.EveEntity, fallback fyne.Resource) fyne.Resource {
	if ee == nil {
		return fallback
	}
	res, err := eis.EntityIcon(ee.ID, ee.Category.ToEveImage(), DefaultIconPixelSize)
	if err != nil {
		slog.Error("eve entity entry icon update", "error", err)
		res = fallback
	}
	res, err = fynetools.MakeAvatar(res)
	if err != nil {
		slog.Error("eve entity entry make avatar", "error", err)
		res = fallback
	}
	return res
}
