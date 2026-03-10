package ui

import (
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
	"github.com/ErikKalkoken/evebuddy/internal/xsync"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type EveEntityEIS interface {
	AllianceLogo(int64, int) (fyne.Resource, error)
	CharacterPortrait(int64, int) (fyne.Resource, error)
	CorporationLogo(int64, int) (fyne.Resource, error)
	FactionLogo(int64, int) (fyne.Resource, error)
	InventoryTypeIcon(int64, int) (fyne.Resource, error)
}

var eveEntityResourceCache xsync.Map[int64, fyne.Resource]

// LoadEveEntityIconAsync fetches an icon for an EveEntity and returns it in avatar style.
func LoadEveEntityIconAsync(eis EveEntityEIS, ee *app.EveEntity, size int, setIcon func(r fyne.Resource)) {
	if ee == nil {
		setIcon(theme.BrokenImageIcon())
		return
	}
	if ee.Category == app.EveEntityMailList {
		setIcon(theme.MailComposeIcon())
		return
	}
	xwidget.LoadResourceAsyncWithCache(
		icons.BlankSvg,
		func() (fyne.Resource, bool) {
			return eveEntityResourceCache.Load(ee.ID)
		},
		func(r fyne.Resource) {
			setIcon(r)
		},
		func() (fyne.Resource, error) {
			return EveEntityIcon(eis, ee, size, theme.BrokenImageIcon())
		},
		func(r fyne.Resource) {
			eveEntityResourceCache.Store(ee.ID, r)
		},
	)
}

type EveEntityIconLoader func(*app.EveEntity, int, func(r fyne.Resource))

// LoadEveEntityIconFunc is an adapter that returns a loadIcon function for LoadEveEntityIconAsync.
func LoadEveEntityIconFunc(eis EveEntityEIS) EveEntityIconLoader {
	return func(o *app.EveEntity, size int, setIcon func(r fyne.Resource)) {
		LoadEveEntityIconAsync(eis, o, size, setIcon)
	}
}

// EveEntityIcon returns an icon from EveImageService for supported categories.
// Or the fallback for unsupported categories.
func EveEntityIcon(eis EveEntityEIS, ee *app.EveEntity, size int, fallback fyne.Resource) (fyne.Resource, error) {
	var r fyne.Resource
	var err error
	switch ee.Category {
	case app.EveEntityAlliance:
		r, err = eis.AllianceLogo(ee.ID, size)
	case app.EveEntityCharacter:
		r, err = eis.CharacterPortrait(ee.ID, size)
	case app.EveEntityCorporation:
		r, err = eis.CorporationLogo(ee.ID, size)
	case app.EveEntityFaction:
		r, err = eis.FactionLogo(ee.ID, size)
	case app.EveEntityInventoryType:
		r, err = eis.InventoryTypeIcon(ee.ID, size)
	default:
		if fallback != nil {
			return fallback, nil
		}
		slog.Warn("unsupported category. Falling back to default", "entity", ee)
		return theme.BrokenImageIcon(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("entity icon %v %d: %w", ee, size, err)
	}
	return r, nil
}

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

func NewEveEntityListItem(loadIcon EveEntityIconLoader) *EveEntityListItem {
	w := &EveEntityListItem{
		icon: xwidget.NewImageFromResource(
			icons.BlankSvg,
			fyne.NewSquareSize(app.IconUnitSize),
		),
		IconPixelSize: app.IconPixelSize,
		IconUnitSize:  app.IconUnitSize,
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
	EIS       EveEntityEIS
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
			x := NewEveEntityListItem(LoadEveEntityIconFunc(arg.EIS))
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
