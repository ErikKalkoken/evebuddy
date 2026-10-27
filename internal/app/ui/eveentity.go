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

// entityKey identifies an EveEntity.
type entityKey struct {
	id       int64
	category app.EveEntityCategory
}

// EveEntityListItem is a list item widget that renders an EveEntity with icon and name.
type EveEntityListItem struct {
	widget.BaseWidget

	IconPixelSize int // takes effect when calling Set()
	IconUnitSize  float32
	IsAvatar      bool
	Truncation    fyne.TextTruncation

	entity     entityKey // entity currently shown
	iconImage  *canvas.Image
	iconLoader EveEntityIconLoader
	nameLabel  *widget.Label
}

// NewEveEntityListItem returns a new EveEntityListItem widget.
func NewEveEntityListItem(iconLoader EveEntityIconLoader) *EveEntityListItem {
	iconImage := xwidget.NewImageFromResource(
		icons.BlankSvg,
		fyne.NewSquareSize(IconUnitSize),
	)
	w := &EveEntityListItem{
		iconImage:     iconImage,
		IconPixelSize: IconPixelSize,
		IconUnitSize:  IconUnitSize,
		iconLoader:    iconLoader,
		nameLabel:     widget.NewLabel(""),
		Truncation:    fyne.TextTruncateClip,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *EveEntityListItem) CreateRenderer() fyne.WidgetRenderer {
	w.updateFromFields()
	p := theme.Padding()
	c := container.NewBorder(
		nil,
		nil,
		container.NewVBox(
			layout.NewSpacer(),
			container.New(layout.NewCustomPaddedLayout(p, p, 2*p, -p), w.iconImage),
			layout.NewSpacer(),
		),
		nil,
		container.NewVBox(
			layout.NewSpacer(),
			w.nameLabel,
			layout.NewSpacer(),
		),
	)
	return widget.NewSimpleRenderer(c)
}

func (w *EveEntityListItem) Refresh() {
	w.updateFromFields()
	w.BaseWidget.Refresh()
}

// updateFromFields applies the exported fields to the inner widgets.
func (w *EveEntityListItem) updateFromFields() {
	w.nameLabel.Truncation = w.Truncation
	w.iconImage.SetMinSize(fyne.NewSquareSize(w.IconUnitSize))
	if w.IsAvatar {
		w.iconImage.CornerRadius = w.IconUnitSize / 2
	} else {
		w.iconImage.CornerRadius = 0
	}
}

// Set updates the widget.
//
// If o.ID is zero it will render no icon and the name in bold style.
// This can be used to show totals in tables.
func (w *EveEntityListItem) Set(o *app.EveEntity) {
	if o == nil {
		w.entity = entityKey{}
		w.nameLabel.Text = "?"
		w.nameLabel.TextStyle.Bold = false
	} else {
		w.entity = entityKey{id: o.ID, category: o.Category}
		w.nameLabel.Text = o.Name
		w.nameLabel.TextStyle.Bold = o.ID == 0
	}
	w.iconImage.Resource = icons.BlankSvg
	w.Refresh()
	if o == nil || o.ID == 0 {
		return
	}
	loading := w.entity
	w.iconLoader(o, w.IconPixelSize, func(r fyne.Resource) {
		if w.entity != loading {
			return // row now shows another entity
		}
		w.iconImage.Resource = r
		w.iconImage.Refresh()
	})
}

// Set2 updates the widget.
//
// If id is zero it will render no icon and the name in bold style.
// This can be used to show totals in tables.
func (w *EveEntityListItem) Set2(id int64, name string, category app.EveEntityCategory) {
	w.Set(&app.EveEntity{
		Category: category,
		ID:       id,
		Name:     name,
	})
}

// MakeEveEntityColumnParams represents the parameters for MakeEveEntityColumn()
type MakeEveEntityColumnParams[T any] struct {
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
			return xstrings.CompareIgnoreCase(arg.GetEntity(a).NameOrZero(), arg.GetEntity(b).NameOrZero())
		},
	}
	return c
}
