package assetui

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/xwindow"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// ShowAssetDetailWindow shows the details for an assets in a new window.
func ShowAssetDetailWindow(u ui, r assetRow) {
	w, created := u.GetOrCreateWindow(
		fmt.Sprintf("asset-%d-%d", r.owner.ID, r.itemID),
		"Asset: Information",
		r.owner.Name,
	)
	if !created {
		w.Show()
		return
	}
	item := makeLinkLabelWithWrap(r.typeName, func() {
		if r.owner.IsCharacter() {
			u.InfoWindow().ShowTypeWithCharacter(r.typeID, r.owner.ID)
		} else {
			u.InfoWindow().ShowType(r.typeID)
		}
	})
	var location, region fyne.CanvasObject
	if r.location != nil {
		location = makeLocationLabel(r.location, u.InfoWindow().ShowLocation)
		region = makeLinkLabel(r.regionName, func() {
			u.InfoWindow().Show(app.EveEntityRegion, r.regionID)
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
		widget.NewFormItem("Owner", makeCharacterActionLabel(
			r.owner.ID,
			r.owner.Name,
			u.InfoWindow().ShowEntity,
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
				return formatISKAmount(v)
			})),
		),
		widget.NewFormItem("Quantity", widget.NewLabel(r.quantityDisplay)),
		widget.NewFormItem(
			"Total",
			widget.NewLabel(r.total.StringFunc("?", func(v float64) string {
				return formatISKAmount(v)
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
	xwindow.Set(xwindow.Params{
		Content: f,
		ImageAction: func() {
			u.InfoWindow().ShowType(r.typeID)
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

// formatISKAmount returns a formatted ISK amount.
// This format is mainly used in detail windows.
func formatISKAmount(v float64) string {
	t := humanize.FormatFloat(app.FloatFormat, v) + " ISK"
	if math.Abs(v) > 999 {
		t += fmt.Sprintf(" (%s)", ihumanize.NumberF(v, 2))
	}
	return t
}

func makeLocationLabel(o *app.EveLocationShort, show func(int64)) fyne.CanvasObject {
	if o == nil {
		return widget.NewLabel("?")
	}
	x := makeLinkLabelWithWrap(o.DisplayName(), func() {
		show(o.ID)
	})
	x.Wrapping = fyne.TextWrapWord
	return x
}

func makeLinkLabelWithWrap(text string, action func()) *widget.Hyperlink {
	x := makeLinkLabel(text, action)
	x.Wrapping = fyne.TextWrapWord
	return x
}

func makeLinkLabel(text string, action func()) *widget.Hyperlink {
	x := widget.NewHyperlink(text, nil)
	x.OnTapped = action
	return x
}

func makeCharacterActionLabel(id int64, name string, action func(o *app.EveEntity)) fyne.CanvasObject {
	o := &app.EveEntity{
		ID:       id,
		Name:     name,
		Category: app.EveEntityCharacter,
	}
	return makeEveEntityActionLabel(o, action)
}

// makeEveEntityActionLabel returns a Hyperlink for existing entities or a placeholder label otherwise.
func makeEveEntityActionLabel(o *app.EveEntity, action func(o *app.EveEntity)) fyne.CanvasObject {
	if o == nil {
		return widget.NewLabel("-")
	}
	return makeLinkLabelWithWrap(o.Name, func() {
		action(o)
	})
}
