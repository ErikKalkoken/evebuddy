package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// EveEntityIconLoader is defines the function signature for a EveEntity icon loader
type EveEntityIconLoader func(*app.EveEntity, int, func(r fyne.Resource))

// EveEntityListItem is a list item widget that renders an EveEntity with icon and name.
type EveEntityListItem struct {
	widget.BaseWidget

	IconPixelSize int                 // can be set before first render
	IconUnitSize  float32             // can be set before first render
	IsAvatar      bool                // can be set always
	Truncation    fyne.TextTruncation // can be set before first render

	icon     *canvas.Image
	loadIcon EveEntityIconLoader
	name     *widget.Label
}

// NewEveEntityListItem returns a new EveEntityListItem widget.
func NewEveEntityListItem(loadIcon EveEntityIconLoader) *EveEntityListItem {
	w := &EveEntityListItem{
		icon: xwidget.NewImageFromResource(
			icons.BlankSvg,
			fyne.NewSquareSize(IconUnitSize),
		),
		IconPixelSize: IconPixelSize,
		IconUnitSize:  IconUnitSize,
		loadIcon:      loadIcon,
		name:          widget.NewLabel(""),
		Truncation:    fyne.TextTruncateClip,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *EveEntityListItem) CreateRenderer() fyne.WidgetRenderer {
	w.name.Truncation = w.Truncation
	w.icon.SetMinSize(fyne.NewSquareSize(w.IconUnitSize))
	if w.IsAvatar {
		w.icon.CornerRadius = w.IconUnitSize / 2
	}
	p := theme.Padding()
	c := container.NewBorder(
		nil,
		nil,
		container.NewVBox(
			layout.NewSpacer(),
			container.New(layout.NewCustomPaddedLayout(p, p, 2*p, -p), w.icon),
			layout.NewSpacer(),
		),
		nil,
		container.NewVBox(
			layout.NewSpacer(),
			w.name,
			layout.NewSpacer(),
		),
	)
	return widget.NewSimpleRenderer(c)
}

// Set updates the widget.
func (w *EveEntityListItem) Set(o *app.EveEntity) {
	if w.IsAvatar {
		w.icon.CornerRadius = w.IconUnitSize / 2
	}
	w.loadIcon(o, w.IconPixelSize, func(r fyne.Resource) {
		w.icon.Resource = r
		w.icon.Refresh()
	})
	w.name.SetText(o.Name)
}

// Set2 updates the widget.
func (w *EveEntityListItem) Set2(id int64, name string, category app.EveEntityCategory) {
	w.Set(&app.EveEntity{
		Category: category,
		ID:       id,
		Name:     name,
	})
}

// MakeEveEntityColumnParams represents the parameters for MakeEveEntityColumn()
type MakeEveEntityColumnParams[T any] struct {
	ColumnID  int
	EIS       EVEImageService
	GetEntity func(r T) *app.EveEntity
	IsAvatar  bool
	Label     string
	Width     int
}

// MakeEveEntityColumn returns a new data column for showing an entity.
func MakeEveEntityColumn[T any](arg MakeEveEntityColumnParams[T]) xwidget.DataColumn[T] {
	// set defaults
	if arg.Width == 0 {
		arg.Width = 220
	}
	if arg.GetEntity == nil {
		panic("must define entity getter")
	}
	if arg.EIS == nil {
		panic("must define eis")
	}
	c := xwidget.DataColumn[T]{
		ID:    arg.ColumnID,
		Label: arg.Label,
		Width: float32(arg.Width),
		Create: func() fyne.CanvasObject {
			x := NewEveEntityListItem(arg.EIS.EveEntityLogoAsync)
			x.IsAvatar = arg.IsAvatar
			return x
		},
		Update: func(r T, co fyne.CanvasObject) {
			co.(*EveEntityListItem).Set(arg.GetEntity(r))
		},
		Sort: func(a, b T) int {
			return xstrings.CompareIgnoreCase(arg.GetEntity(a).Name, arg.GetEntity(b).Name)
		},
	}
	return c
}
