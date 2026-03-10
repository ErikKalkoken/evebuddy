package gamesearch

import (
	"context"

	"fyne.io/fyne/v2"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/awidget"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xsync"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type searchResultEIS interface {
	AllianceLogo(int64, int) (fyne.Resource, error)
	CharacterPortrait(int64, int) (fyne.Resource, error)
	CorporationLogo(int64, int) (fyne.Resource, error)
	FactionLogo(int64, int) (fyne.Resource, error)
	InventoryTypeIcon(int64, int) (fyne.Resource, error)
	InventoryTypeSKIN(int64, int) (fyne.Resource, error)
	InventoryTypeBPO(int64, int) (fyne.Resource, error)
}

type searchResultEUS interface {
	GetOrCreateTypeESI(context.Context, int64) (*app.EveType, error)
}

var searchResultResourceCache xsync.Map[int64, fyne.Resource]

func loadIconFunc(eis searchResultEIS, eus searchResultEUS) func(o *app.EveEntity, setIcon func(r fyne.Resource)) {
	return func(o *app.EveEntity, setIcon func(r fyne.Resource)) {
		xwidget.LoadResourceAsyncWithCache(
			icons.BlankSvg,
			func() (fyne.Resource, bool) {
				return searchResultResourceCache.Load(o.ID)
			},
			setIcon,
			func() (fyne.Resource, error) {
				switch o.Category {
				case app.EveEntityInventoryType:
					et, err := eus.GetOrCreateTypeESI(context.Background(), o.ID)
					if err != nil {
						return nil, err
					}
					switch et.Group.Category.ID {
					case app.EveCategorySKINs:
						return eis.InventoryTypeSKIN(et.ID, app.IconPixelSize)
					case app.EveCategoryBlueprint:
						return eis.InventoryTypeBPO(et.ID, app.IconPixelSize)
					default:
						return eis.InventoryTypeIcon(et.ID, app.IconPixelSize)
					}
				default:
					return awidget.EveEntityIcon(eis, o, app.IconPixelSize, icons.BlankSvg)
				}
			},
			func(r fyne.Resource) {
				searchResultResourceCache.Store(o.ID, r)
			},
		)
	}
}
