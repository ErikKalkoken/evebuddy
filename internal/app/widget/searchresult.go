package widget

import (
	"context"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverse"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type SearchResult struct {
	widget.BaseWidget
	eis                 app.EveImageService
	eus                 *eveuniverse.EveUniverseService
	name                *widget.Label
	image               *canvas.Image
	supportedCategories set.Set[app.EveEntityCategory]
}

func NewSearchResult(
	eis app.EveImageService,
	eus *eveuniverse.EveUniverseService,
	supportedCategories set.Set[app.EveEntityCategory]) *SearchResult {
	w := &SearchResult{
		eis:                 eis,
		eus:                 eus,
		supportedCategories: supportedCategories,
		name:                widget.NewLabel(""),
		image:               iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize)),
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *SearchResult) Set(o *app.EveEntity) {
	w.name.Text = o.Name
	var i widget.Importance
	if !w.supportedCategories.Contains(o.Category) {
		i = widget.LowImportance
	}
	w.name.Importance = i
	w.name.Refresh()
	imageCategory := o.Category.ToEveImage()
	if imageCategory == "" {
		w.image.Resource = icons.BlankSvg
		w.image.Refresh()
		return
	}
	go func() {
		ctx := context.Background()
		res, err := func() (fyne.Resource, error) {
			switch o.Category {
			case app.EveEntityInventoryType:
				et, err := w.eus.GetOrCreateEveTypeESI(ctx, o.ID)
				if err != nil {
					return nil, err
				}
				switch et.Group.Category.ID {
				case app.EveCategorySKINs:
					return w.eis.InventoryTypeSKIN(et.ID, app.IconPixelSize)
				case app.EveCategoryBlueprint:
					return w.eis.InventoryTypeBPO(et.ID, app.IconPixelSize)
				default:
					return w.eis.InventoryTypeIcon(et.ID, app.IconPixelSize)
				}
			default:
				return w.eis.EntityIcon(o.ID, imageCategory, app.IconPixelSize)
			}
		}()
		if err != nil {
			res = theme.BrokenImageIcon()
			slog.Error("failed to load w.image", "error", err)
		}
		w.image.Resource = res
		w.image.Refresh()
	}()
}

func (w *SearchResult) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(nil, nil, container.NewPadded(w.image), nil, w.name)
	return widget.NewSimpleRenderer(c)
}
