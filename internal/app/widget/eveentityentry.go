package widget

import (
	"fmt"
	"log/slog"
	"slices"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxlayout "github.com/ErikKalkoken/fyne-kx/layout"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icon"
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

	bg          *canvas.Rectangle
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
		bg:           bg,
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
	w.update()
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
		w.update()
		w.Refresh()
	}
}

func (w *EveEntityEntry) remove(id int32) {
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
		w.update()
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
	th := w.Theme()
	isDisabled := w.Disabled()
	w.main.RemoveAll()
	colums := kxlayout.NewColumns(w.labelWidth)
	if len(w.s) == 0 {
		w.placeholder.Segments[0].(*widget.TextSegment).Text = w.Placeholder
		w.main.Add(container.New(colums, w.label, w.placeholder))
	} else {
		padding := th.Size(theme.SizeNamePadding)
		iconSize := th.Size(theme.SizeNameInlineIcon)
		deleteIcon := th.Icon(theme.IconNameDelete)
		firstRow := true
		for _, r := range w.s {
			name := widget.NewLabel(r.Name)
			name.Truncation = fyne.TextTruncateEllipsis
			if isDisabled {
				name.Importance = widget.LowImportance
			}
			icon := iwidget.NewImageFromResource(w.FallbackIcon, fyne.NewSquareSize(iconSize))
			var delete fyne.CanvasObject
			if !isDisabled {
				delete = iwidget.NewIconButton(deleteIcon, func() {
					w.remove(r.ID)
				})
			} else {
				delete = container.NewPadded()
			}
			var label fyne.CanvasObject
			if firstRow {
				label = w.label
				firstRow = false
			} else {
				label = layout.NewSpacer()
			}
			row := container.New(
				colums,
				label,
				container.NewBorder(
					nil,
					nil,
					container.New(layout.NewCustomPaddedLayout(0, 0, padding, 0), icon),
					delete,
					name,
				))
			w.main.Add(row)
			go func() {
				res, err := FetchEveEntityIcon(w.eis, r, DefaultIconPixelSize)
				if err != nil {
					slog.Error("eve entity entry icon update", "error", err)
					res = w.FallbackIcon
				}
				res, err = iwidget.MakeAvatar(res)
				if err != nil {
					slog.Error("eve entity entry make avatar", "error", err)
					res = w.FallbackIcon
				}
				icon.Resource = res
				icon.Refresh()
			}()
		}
	}
}

func (w *EveEntityEntry) Refresh() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	w.bg.FillColor = th.Color(theme.ColorNameInputBackground, v)
	w.bg.StrokeColor = th.Color(theme.ColorNameInputBorder, v)
	w.BaseWidget.Refresh()
	w.main.Refresh()
	w.bg.Refresh()
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
	c := container.NewStack(
		w.bg,
		w.main,
	)
	return widget.NewSimpleRenderer(c)
}

// FetchEveEntityIcon fetches and returns an icon for an EveEntity.
func FetchEveEntityIcon(s app.EveImageService, ee *app.EveEntity, size int) (fyne.Resource, error) {
	res, err := func() (fyne.Resource, error) {
		switch ee.Category {
		case app.EveEntityCharacter:
			return s.CharacterPortrait(ee.ID, size)
		case app.EveEntityAlliance:
			return s.AllianceLogo(ee.ID, size)
		case app.EveEntityCorporation:
			return s.CorporationLogo(ee.ID, size)
		default:
			return nil, fmt.Errorf("unsuported category: %s", ee.Category)
		}
	}()
	if err != nil {
		return nil, fmt.Errorf("fetch eve entity icon for %+v: %w", ee, err)
	}
	return res, nil
}
