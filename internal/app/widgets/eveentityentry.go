package widgets

import (
	"slices"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

// EveEntityEntry represents an entry widgets for entering Eve Entities.
// Entities can be added and removed.
type EveEntityEntry struct {
	widget.BaseWidget

	Recipients []*app.EveEntity

	bg            *canvas.Rectangle
	main          *fyne.Container
	placeholder   *widget.RichText
	showAddDialog func(func(*app.EveEntity))
	mu            sync.Mutex
}

var _ fyne.Tappable = (*EveEntityEntry)(nil)

func NewEveEntityEntry(showAddDialog func(onSelected func(*app.EveEntity))) *EveEntityEntry {
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
		main:          container.NewGridWithColumns(1),
		Recipients:    make([]*app.EveEntity, 0),
		placeholder:   placeholder,
		showAddDialog: showAddDialog,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *EveEntityEntry) Set(ee []*app.EveEntity) {
	w.mu.Lock()
	w.Recipients = ee
	w.mu.Unlock()
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
	w.main.RemoveAll()
	if len(w.Recipients) == 0 {
		w.main.Add(w.placeholder)
	} else {
		for _, r := range w.Recipients {
			name := widget.NewLabel(r.Name)
			name.Truncation = fyne.TextTruncateEllipsis
			category := NewLabelWithSize(r.CategoryDisplay(), theme.SizeNameCaptionText)
			w.main.Add(container.NewBorder(
				nil,
				nil,
				nil,
				container.NewHBox(
					category,
					NewIconButton(theme.DeleteIcon(), func() {
						w.remove(r.ID)
					})),
				name,
			))
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

func (w *EveEntityEntry) CreateRenderer() fyne.WidgetRenderer {
	w.updateMain()
	c := container.NewStack(
		w.bg,
		w.main,
	)
	return widget.NewSimpleRenderer(c)
}
