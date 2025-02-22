package widget

import (
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
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

// EveEntityEntry represents an entry widgets for entering Eve Entities.
// Entities can be added and removed.
type EveEntityEntry struct {
	widget.DisableableWidget

	Label      string
	LabelWidth float32
	Recipients []*app.EveEntity

	bg            *canvas.Rectangle
	hovered       bool
	iconLoader    func(*canvas.Image, *app.EveEntity)
	main          *fyne.Container
	placeholder   *widget.RichText
	showAddDialog func(func(*app.EveEntity))
	mu            sync.Mutex
}

var _ fyne.Tappable = (*EveEntityEntry)(nil)
var _ desktop.Hoverable = (*EveEntityEntry)(nil)

func NewEveEntityEntry(
	label string,
	labelWidth float32,
	iconLoader func(*canvas.Image, *app.EveEntity),
	showAddDialog func(onSelected func(*app.EveEntity)),
) *EveEntityEntry {
	bg := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	bg.StrokeColor = theme.Color(theme.ColorNameInputBorder)
	bg.StrokeWidth = theme.Size(theme.SizeNameInputBorder)
	bg.CornerRadius = theme.Size(theme.SizeNameInputRadius)
	placeholder := widget.NewRichText(&widget.TextSegment{
		Style: widget.RichTextStyle{ColorName: theme.ColorNamePlaceHolder},
		Text:  "Tap to add recipients...",
	})
	w := &EveEntityEntry{
		bg:            bg,
		iconLoader:    iconLoader,
		Label:         label,
		LabelWidth:    labelWidth,
		main:          container.New(layout.NewCustomPaddedVBoxLayout(0)),
		placeholder:   placeholder,
		Recipients:    make([]*app.EveEntity, 0),
		showAddDialog: showAddDialog,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *EveEntityEntry) Set(ee []*app.EveEntity) {
	w.mu.Lock()
	w.Recipients = ee
	w.mu.Unlock()
	w.updateMain()
	w.Refresh()
}

func (w *EveEntityEntry) Add(ee *app.EveEntity) {
	added := func() bool {
		w.mu.Lock()
		defer w.mu.Unlock()
		for _, o := range w.Recipients {
			if o.ID == ee.ID {
				return false
			}
		}
		w.Recipients = append(w.Recipients, ee)
		return true
	}()
	if added {
		w.updateMain()
		w.Refresh()
	}
}

func (w *EveEntityEntry) remove(id int32) {
	removed := func() bool {
		w.mu.Lock()
		defer w.mu.Unlock()
		for i, o := range w.Recipients {
			if o.ID == id {
				w.Recipients = slices.Delete(w.Recipients, i, i+1)
				return true
			}
		}
		return false
	}()
	if removed {
		w.updateMain()
		w.Refresh()
	}
}

func (w *EveEntityEntry) IsEmpty() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.Recipients) == 0
}

func (w *EveEntityEntry) updateMain() {
	w.mu.Lock()
	defer w.mu.Unlock()
	th := w.Theme()
	isDisabled := w.Disabled()
	w.main.RemoveAll()
	colums := kxlayout.NewColumns(w.LabelWidth)
	if len(w.Recipients) == 0 {
		w.main.Add(container.New(colums, widget.NewLabel(w.Label), w.placeholder))
	} else {
		padding := th.Size(theme.SizeNamePadding)
		iconSize := th.Size(theme.SizeNameInlineIcon)
		deleteIcon := th.Icon(theme.IconNameDelete)
		questionIcon := th.Icon(theme.IconNameQuestion)
		firstRow := true
		for _, r := range w.Recipients {
			name := widget.NewLabel(r.Name)
			name.Truncation = fyne.TextTruncateEllipsis
			if isDisabled {
				name.Importance = widget.LowImportance
			}
			icon := iwidget.NewImageFromResource(questionIcon, fyne.NewSquareSize(iconSize))
			var delete fyne.CanvasObject
			if !isDisabled {
				delete = iwidget.NewIconButton(deleteIcon, func() {
					w.remove(r.ID)
				})
			} else {
				delete = container.NewPadded()
			}
			var label string
			if firstRow {
				label = w.Label
				firstRow = false
			} else {
				label = ""
			}
			row := container.New(
				colums,
				widget.NewLabel(label),
				container.NewBorder(
					nil,
					nil,
					container.New(layout.NewCustomPaddedLayout(0, 0, padding, 0), icon),
					delete,
					name,
				))
			w.main.Add(row)
			w.iconLoader(icon, r)
		}
	}
	// w.main.Add(container.NewHBox(
	// 	NewIconButton(theme.NewPrimaryThemedResource(theme.ContentAddIcon()), func() {
	// 		w.showAddDialog(func(ee *app.EveEntity) {
	// 			w.Add(ee)
	// 		})
	// 	})))
}

func (w *EveEntityEntry) Tapped(_ *fyne.PointEvent) {
	if w.Disabled() || w.showAddDialog == nil {
		return
	}
	w.showAddDialog(func(ee *app.EveEntity) {
		w.Add(ee)
	})
}

func (w *EveEntityEntry) TappedSecondary(_ *fyne.PointEvent) {
}

func (w *EveEntityEntry) Refresh() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	w.bg.FillColor = th.Color(theme.ColorNameInputBackground, v)
	w.bg.StrokeColor = th.Color(theme.ColorNameInputBorder, v)
	w.updateMain()
	w.BaseWidget.Refresh()
	w.main.Refresh()
	w.bg.Refresh()
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

// Cursor returns the cursor type of this widget
func (w *EveEntityEntry) Cursor() desktop.Cursor {
	if w.hovered {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

// MouseIn is a hook that is called if the mouse pointer enters the element.
func (w *EveEntityEntry) MouseIn(e *desktop.MouseEvent) {
	w.hovered = true
}

func (w *EveEntityEntry) MouseMoved(*desktop.MouseEvent) {
	// needed to satisfy the interface only
}

// MouseOut is a hook that is called if the mouse pointer leaves the element.
func (w *EveEntityEntry) MouseOut() {
	w.hovered = false
}

func (w *EveEntityEntry) CreateRenderer() fyne.WidgetRenderer {
	w.updateMain()
	c := container.NewStack(
		w.bg,
		w.main,
	)
	return widget.NewSimpleRenderer(c)
}
