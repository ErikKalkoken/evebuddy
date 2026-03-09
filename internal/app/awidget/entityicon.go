package awidget

import (
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
	"github.com/ErikKalkoken/evebuddy/internal/xsync"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type eis interface {
	AllianceLogo(int64, int) (fyne.Resource, error)
	CharacterPortrait(int64, int) (fyne.Resource, error)
	CorporationLogo(int64, int) (fyne.Resource, error)
	FactionLogo(int64, int) (fyne.Resource, error)
	InventoryTypeIcon(int64, int) (fyne.Resource, error)
}

var eveEntityResourceCache xsync.Map[int64, fyne.Resource]

// LoadEveEntityIconAsync fetches an icon for an EveEntity and returns it in avatar style.
func LoadEveEntityIconAsync(eis eis, ee *app.EveEntity, setIcon func(r fyne.Resource)) {
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
			return EntityIcon(eis, ee, app.IconPixelSize, theme.NewThemedResource(icons.QuestionmarkSvg))
		},
		func(r fyne.Resource) {
			eveEntityResourceCache.Store(ee.ID, r)
		},
	)
}

// EntityIcon returns an icon form EveImageService for several entity categories.
// It returns the fallback for unsupported categories.
func EntityIcon(eis eis, ee *app.EveEntity, size int, fallback fyne.Resource) (fyne.Resource, error) {
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
		return icons.Questionmark32Png, nil
	}
	if err != nil {
		return nil, fmt.Errorf("entity icon %v %d: %w", ee, size, err)
	}
	return r, nil
}

// LoadIconFunc is an adaptor that returns a loadIcon function for LoadEveEntityIconAsync.
func LoadIconFunc(eis eis) func(o *app.EveEntity, setIcon func(r fyne.Resource)) {
	return func(o *app.EveEntity, setIcon func(r fyne.Resource)) {
		LoadEveEntityIconAsync(eis, o, setIcon)
	}
}

// MakeEveEntityColumnParams represents the parameters for MakeEveEntityColumn()
type MakeEveEntityColumnParams[T any] struct {
	ColumnID  int
	EIS       eis
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
			icon := xwidget.NewImageFromResource(
				icons.Characterplaceholder64Jpeg,
				fyne.NewSquareSize(app.IconUnitSize),
			)
			if arg.IsAvatar {
				icon.CornerRadius = app.IconUnitSize / 2
			}
			name := widget.NewLabel(arg.Label)
			name.Truncation = fyne.TextTruncateClip
			return container.NewBorder(nil, nil, icon, nil, name)
		},
		Update: func(r T, co fyne.CanvasObject) {
			ee := arg.GetEntity(r)
			border := co.(*fyne.Container).Objects
			border[0].(*widget.Label).SetText(ee.Name)
			x := border[1].(*canvas.Image)
			LoadEveEntityIconAsync(arg.EIS, ee, func(r fyne.Resource) {
				x.Resource = r
				x.Refresh()
			})
		},
		Sort: func(a, b T) int {
			return xstrings.CompareIgnoreCase(arg.GetEntity(a).Name, arg.GetEntity(b).Name)
		},
	}
	return c
}
