package industry

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// ShowMarketOrderWindow shows the location of a character in a new window.
func ShowMarketOrderWindow(u baseUI, r marketOrderRow) {
	title := fmt.Sprintf("Market Order #%d", r.orderID)
	w, created := u.GetOrCreateWindow(
		fmt.Sprintf("market-order-%d-%d", r.characterID, r.orderID),
		title,
		r.characterName,
	)
	if !created {
		w.Show()
		return
	}
	item := ui.MakeLinkLabelWithWrap(r.typeName, func() {
		u.InfoViewer().ShowType(r.typeID, r.characterID)
	})
	region := ui.MakeLinkLabel(r.regionName, func() {
		u.InfoViewer().Show(&app.EveEntity{Category: app.EveEntityRegion, ID: r.regionID})
	})
	var buySell string
	if r.IsBuyOrder.ValueOrZero() {
		buySell = "buy"
	} else {
		buySell = "sell"
	}

	var expires string
	if r.isExpired() {
		expires = "-"
	} else {
		expires = r.expires.Format(app.DateTimeFormat)
	}

	state := widget.NewLabel(r.stateCorrectedDisplay())
	state.Importance = r.stateImportance()
	items := []*widget.FormItem{
		widget.NewFormItem("Owner", ui.MakeEveEntityActionLabel(
			r.owner,
			u.InfoViewer().Show,
		)),
		widget.NewFormItem("Type", item),
		widget.NewFormItem("Price", widget.NewLabel(ui.FormatISKAmount(r.price))),
		widget.NewFormItem("Variant", widget.NewLabel(buySell)),
		widget.NewFormItem("State", state),
		widget.NewFormItem("Volume Total", widget.NewLabel(ihumanize.Comma(r.volumeTotal))),
		widget.NewFormItem("Volume Remain", widget.NewLabel(ihumanize.Comma(r.volumeRemain))),
		widget.NewFormItem("Issued", widget.NewLabel(r.issued.Format(app.DateTimeFormat))),
		widget.NewFormItem("Expires", widget.NewLabel(expires)),
		widget.NewFormItem("Location", ui.MakeLocationLabel(r.location, u.InfoViewer().ShowLocation)),
		widget.NewFormItem("Region", region),
	}
	if r.IsBuyOrder.ValueOrZero() {
		items = append(items, widget.NewFormItem(
			"Volume Min",
			widget.NewLabel(r.minVolume.StringFunc("?", func(v int64) string {
				return ihumanize.Comma(v)
			})),
		))
		items = append(items, widget.NewFormItem(
			"Escrow",
			widget.NewLabel(r.escrow.StringFunc("-", func(v float64) string {
				return humanize.FormatFloat(ui.FloatFormatISK, v)
			})),
		))
	}
	items = append(items, widget.NewFormItem("Range", widget.NewLabel(xstrings.Title(r.rangeInfo))))
	items = append(items, widget.NewFormItem("For corporation", makeBoolLabel(r.isCorporation)))
	items = append(items, widget.NewFormItem("Character", ui.MakeCharacterActionLabel(
		r.characterID,
		r.characterName,
		u.InfoViewer().Show,
	)))

	if u.IsDeveloperMode() {
		items = append(items, widget.NewFormItem("Order ID", xwidget.NewTappableLabelWithClipboardCopy(fmt.Sprint(r.orderID))))
	}
	f := widget.NewForm(items...)
	f.Orientation = widget.Adaptive
	ui.MakeDetailWindow(ui.MakeDetailWindowParams{
		Content: f,
		ImageAction: func() {
			u.InfoViewer().ShowType(r.typeID, 0)
		},
		ImageLoader: func(setter func(r fyne.Resource)) {
			u.EVEImage().InventoryTypeIconAsync(r.typeID, 256, setter)
		},
		Title:  title,
		Window: w,
	})
	w.Show()
}

func makeBoolLabel(v bool) *widget.Label {
	if v {
		l := widget.NewLabel("Yes")
		l.Importance = widget.SuccessImportance
		return l
	}
	l := widget.NewLabel("No")
	l.Importance = widget.DangerImportance
	return l
}
