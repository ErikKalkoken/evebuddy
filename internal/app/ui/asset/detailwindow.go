package asset

import (
	"fmt"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// ShowAssetDetailWindow shows the details for an assets in a new window.
func ShowAssetDetailWindow(u coreUI, r assetRow) {
	w, created := u.GetOrCreateWindow(
		fmt.Sprintf("asset-%d-%d", r.owner.ID, r.itemID),
		"Asset: Information",
		r.owner.Name,
	)
	if !created {
		w.Show()
		return
	}
	item := ui.MakeLinkLabelWithWrap(r.typeName, func() {
		if r.owner.IsCharacter() {
			u.InfoWindow().ShowType(r.typeID, r.owner.ID)
		} else {
			u.InfoWindow().ShowType(r.typeID, 0)
		}
	})
	var location, region fyne.CanvasObject
	if r.location != nil {
		location = ui.MakeLocationLabel(r.location, u.InfoWindow().ShowLocation)
		region = ui.MakeLinkLabel(r.regionName, func() {
			u.InfoWindow().Show(&app.EveEntity{Category: app.EveEntityRegion, ID: r.regionID})
		})
	} else {
		location = widget.NewLabel("?")
		region = widget.NewLabel("?")
	}

	var p string
	if len(r.locationPath) > 0 {
		p = strings.Join(r.locationPath, " / ")
	} else {
		p = "-"
	}
	path := widget.NewLabel(p)
	path.Wrapping = fyne.TextWrapWord

	items := []*widget.FormItem{
		widget.NewFormItem("Owner", ui.MakeCharacterActionLabel(
			r.owner.ID,
			r.owner.Name,
			u.InfoWindow().Show,
		)),
		widget.NewFormItem("Item", item),
		widget.NewFormItem("Group", widget.NewLabel(r.groupName)),
		widget.NewFormItem("Category", widget.NewLabel(r.categoryName)),
		widget.NewFormItem("Location", location),
		widget.NewFormItem("Path", path),
		widget.NewFormItem("Region", region),
		widget.NewFormItem(
			"Price",
			widget.NewLabel(r.price.StringFunc("?", func(v float64) string {
				return ui.FormatISKAmount(v)
			})),
		),
		widget.NewFormItem("Quantity", widget.NewLabel(r.quantityDisplay)),
		widget.NewFormItem(
			"Total",
			widget.NewLabel(r.total.StringFunc("?", func(v float64) string {
				return ui.FormatISKAmount(v)
			})),
		),
	}
	if u.IsDeveloperMode() {
		items = slices.Concat(items, []*widget.FormItem{
			widget.NewFormItem("Location Flag", widget.NewLabel(r.locationFlag.String())),
			widget.NewFormItem("Item ID", xwidget.NewTappableLabelWithClipboardCopy(fmt.Sprint(r.itemID))),
		})
	}

	f := widget.NewForm(items...)
	f.Orientation = widget.Adaptive
	ui.MakeDetailWindow(ui.MakeDetailWindowParams{
		Content: f,
		ImageAction: func() {
			u.InfoWindow().ShowType(r.typeID, 0)
		},
		ImageLoader: func(setter func(r fyne.Resource)) {
			switch r.variant {
			case app.VariantSKIN:
				u.EVEImage().InventoryTypeSKINAsync(r.typeID, app.IconPixelSize, setter)
			case app.VariantBPO:
				u.EVEImage().InventoryTypeBPOAsync(r.typeID, app.IconPixelSize, setter)
			case app.VariantBPC:
				u.EVEImage().InventoryTypeBPCAsync(r.typeID, app.IconPixelSize, setter)
			default:
				u.EVEImage().InventoryTypeIconAsync(r.typeID, app.IconPixelSize, setter)
			}
		},
		MinSize: fyne.NewSize(500, 450),
		Title:   r.name,
		Window:  w,
	})
	w.Show()
}
