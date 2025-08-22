package ui

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

const (
	marketOrderStateActive  = "Active"
	marketOrderStateHistory = "History"
)

type marketOrderRow struct {
	characterID   int32
	characterName string
	escrow        optional.Optional[float64]
	expires       time.Time
	IsBuyOrder    bool
	isCorporation bool
	location      *app.EveLocationShort
	locationID    int64
	locationName  string
	minVolume     optional.Optional[int]
	orderID       int64
	price         float64
	regionID      int32
	regionName    string
	state         app.MarketOrderState
	tags          set.Set[string]
	typeID        int32
	typeName      string
	volumeRemain  int
	volumeTotal   int
}

func (r marketOrderRow) isExpired() bool {
	return time.Until(r.expires) <= 0
}

func (r marketOrderRow) stateCorrected() app.MarketOrderState {
	if r.state == app.OrderOpen && r.isExpired() {
		return app.OrderExpired
	}
	return r.state
}

func (r marketOrderRow) stateCorrectedDisplay() string {
	return stringTitle(r.stateCorrected().String())
}

func (r marketOrderRow) stateImportance() widget.Importance {
	switch r.stateCorrected() {
	case app.OrderOpen:
		return widget.SuccessImportance
	case app.OrderExpired:
		return widget.DangerImportance
	case app.OrderCancelled:
		return widget.MediumImportance
	}
	return widget.MediumImportance
}

func (r marketOrderRow) stateColor() fyne.ThemeColorName {
	switch r.stateCorrected() {
	case app.OrderOpen:
		return theme.ColorNameSuccess
	case app.OrderExpired:
		return theme.ColorNameError
	case app.OrderCancelled:
		return theme.ColorNameForeground
	}
	return theme.ColorNameForeground
}

func (r marketOrderRow) remaining() string {
	if r.stateCorrected() != app.OrderOpen {
		return "N/A"
	}
	return ihumanize.Duration(time.Until(r.expires))
}

func (r *marketOrderRow) stateDisplay() string {
	if r.stateCorrected() == app.OrderOpen {
		return r.remaining()
	}
	return r.stateCorrectedDisplay()

}

type marketOrders struct {
	widget.BaseWidget

	body         fyne.CanvasObject
	bottom       *widget.Label
	columnSorter *columnSorter
	isBuyOrders  bool
	rows         []marketOrderRow
	rowsFiltered []marketOrderRow
	selectOwner  *kxwidget.FilterChipSelect
	selectRegion *kxwidget.FilterChipSelect
	selectState  *kxwidget.FilterChipSelect
	selectTag    *kxwidget.FilterChipSelect
	selectType   *kxwidget.FilterChipSelect
	sortButton   *sortButton
	u            *baseUI
}

func newMarketOrders(u *baseUI, isBuyOrders bool) *marketOrders {
	headers := []headerDef{
		{label: "Type", width: columnWidthEntity},
		{label: "Quantity", width: 100},
		{label: "Price", width: 100},
		{label: "State", width: 100},
		{label: "Location", width: columnWidthLocation},
		{label: "Region", width: columnWidthRegion},
		{label: "Owner", width: columnWidthEntity},
	}
	a := &marketOrders{
		bottom:       makeTopLabel(),
		columnSorter: newColumnSorterWithInit(headers, 0, sortAsc),
		isBuyOrders:  isBuyOrders,
		rows:         make([]marketOrderRow, 0),
		rowsFiltered: make([]marketOrderRow, 0),
		u:            u,
	}
	a.ExtendBaseWidget(a)
	makeCell := func(col int, r marketOrderRow) []widget.RichTextSegment {
		switch col {
		case 0:
			return iwidget.RichTextSegmentsFromText(r.typeName)
		case 1:
			s := fmt.Sprintf("%d/%d", r.volumeRemain, r.volumeTotal)
			return iwidget.RichTextSegmentsFromText(s)
		case 2:
			return iwidget.RichTextSegmentsFromText(humanize.FormatFloat(app.FloatFormat, r.price))
		case 3:
			return iwidget.RichTextSegmentsFromText(r.stateDisplay(), widget.RichTextStyle{
				ColorName: r.stateColor(),
			})
		case 4:
			return iwidget.RichTextSegmentsFromText(r.locationName)
		case 5:
			return iwidget.RichTextSegmentsFromText(r.regionName)
		case 6:
			return iwidget.RichTextSegmentsFromText(r.characterName)
		}
		return iwidget.RichTextSegmentsFromText("?")
	}
	if a.u.isDesktop {
		a.body = makeDataTable(
			headers,
			&a.rowsFiltered,
			makeCell,
			a.columnSorter,
			a.filterRows, func(_ int, r marketOrderRow) {
				showMarketOrderWindow(a.u, r)
			})
	} else {
		// a.body = makeDataList(headers, &a.rowsFiltered, makeCell, func(r marketOrderRow) {
		// 	showMarketOrderWindow(u, r)
		// })
		a.body = a.makeDataList()
	}

	a.selectRegion = kxwidget.NewFilterChipSelect("Region", []string{}, func(string) {
		a.filterRows(-1)
	})
	a.selectOwner = kxwidget.NewFilterChipSelect("Owner", []string{}, func(string) {
		a.filterRows(-1)
	})
	a.selectState = kxwidget.NewFilterChipSelect("", []string{
		marketOrderStateActive,
		marketOrderStateHistory,
	}, func(_ string) {
		a.filterRows(-1)
	})
	a.selectState.Selected = marketOrderStateActive
	a.selectState.SortDisabled = true
	a.selectType = kxwidget.NewFilterChipSelectWithSearch("Type", []string{}, func(string) {
		a.filterRows(-1)
	}, a.u.window)
	a.selectTag = kxwidget.NewFilterChipSelect("Tag", []string{}, func(string) {
		a.filterRows(-1)
	})
	a.sortButton = a.columnSorter.newSortButton(headers, func() {
		a.filterRows(-1)
	}, a.u.window)

	a.u.characterSectionChanged.AddListener(func(_ context.Context, arg characterSectionUpdated) {
		switch arg.section {
		case app.SectionCharacterMarketOrders:
			a.update()
		}
	})
	return a
}

func (a *marketOrders) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHBox(a.selectType, a.selectState, a.selectRegion, a.selectOwner, a.selectTag)
	if !a.u.isDesktop {
		filter.Add(a.sortButton)
	}
	c := container.NewBorder(container.NewHScroll(filter), a.bottom, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *marketOrders) startUpdateTicker() {
	ticker := time.NewTicker(time.Second * 60)
	go func() {
		for {
			<-ticker.C
			fyne.DoAndWait(func() {
				a.body.Refresh()
			})
		}
	}()
}

func (a *marketOrders) makeDataList() *iwidget.StripedList {
	p := theme.Padding()
	l := iwidget.NewStripedList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			item := widget.NewLabel("Template")
			item.Truncation = fyne.TextTruncateClip
			item.TextStyle.Bold = true
			state := widget.NewLabel("Template")
			state.Alignment = fyne.TextAlignTrailing
			price := widget.NewLabel("Template")
			price.Truncation = fyne.TextTruncateClip
			volume := widget.NewLabel("Template")
			volume.Alignment = fyne.TextAlignTrailing
			location := iwidget.NewRichText()
			location.Truncation = fyne.TextTruncateClip
			owner := widget.NewLabel("Template")
			owner.Truncation = fyne.TextTruncateClip
			return container.New(layout.NewCustomPaddedVBoxLayout(-p),
				container.NewBorder(nil, nil, nil, state, item),
				container.NewBorder(nil, nil, nil, volume, price),
				location,
				owner,
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id < 0 || id >= len(a.rowsFiltered) {
				return
			}
			r := a.rowsFiltered[id]
			c := co.(*fyne.Container).Objects

			b0 := c[0].(*fyne.Container).Objects
			b0[0].(*widget.Label).SetText(r.typeName)
			state := b0[1].(*widget.Label)
			state.Text = r.stateDisplay()
			state.Importance = r.stateImportance()
			state.Refresh()

			b1 := c[1].(*fyne.Container).Objects
			b1[0].(*widget.Label).SetText(ihumanize.Number(r.price, 2) + " ISK")
			b1[1].(*widget.Label).SetText(fmt.Sprintf("%d/%d", r.volumeRemain, r.volumeTotal))

			c[2].(*iwidget.RichText).Set(r.location.DisplayRichText())
			c[3].(*widget.Label).SetText(r.characterName)
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll()
		if id < 0 || id >= len(a.rowsFiltered) {
			return
		}
		r := a.rowsFiltered[id]
		showMarketOrderWindow(a.u, r)
	}
	l.HideSeparators = true
	return l
}

func (a *marketOrders) filterRows(sortCol int) {
	rows := slices.Clone(a.rows)
	// filter
	if x := a.selectRegion.Selected; x != "" {
		rows = xslices.Filter(rows, func(r marketOrderRow) bool {
			return r.regionName == x
		})
	}
	if x := a.selectOwner.Selected; x != "" {
		rows = xslices.Filter(rows, func(r marketOrderRow) bool {
			return r.characterName == x
		})
	}
	rows = xslices.Filter(rows, func(r marketOrderRow) bool {
		s := r.stateCorrected()
		switch a.selectState.Selected {
		case marketOrderStateActive:
			return s == app.OrderOpen
		case marketOrderStateHistory:
			return s == app.OrderCancelled || s == app.OrderExpired
		}
		return false
	})
	if x := a.selectType.Selected; x != "" {
		rows = xslices.Filter(rows, func(r marketOrderRow) bool {
			return r.typeName == x
		})
	}
	if x := a.selectTag.Selected; x != "" {
		rows = xslices.Filter(rows, func(r marketOrderRow) bool {
			return r.tags.Contains(x)
		})
	}
	// sort
	a.columnSorter.sort(sortCol, func(sortCol int, dir sortDir) {
		slices.SortFunc(rows, func(a, b marketOrderRow) int {
			var x int
			switch sortCol {
			case 0:
				x = strings.Compare(a.typeName, b.typeName)
			case 1:
				x = cmp.Compare(a.volumeRemain, b.volumeRemain)
			case 2:
				x = cmp.Compare(a.price, b.price)
			case 3:
				x = a.expires.Compare(b.expires)
			case 4:
				x = strings.Compare(a.regionName, b.regionName)
			case 5:
				x = strings.Compare(a.locationName, b.locationName)
			case 6:
				x = strings.Compare(a.characterName, b.characterName)
			}
			if dir == sortAsc {
				return x
			} else {
				return -1 * x
			}
		})
	})
	// set data & refresh
	a.selectRegion.SetOptions(xslices.Map(rows, func(r marketOrderRow) string {
		return r.regionName
	}))
	a.selectOwner.SetOptions(xslices.Map(rows, func(r marketOrderRow) string {
		return r.characterName
	}))
	a.selectTag.SetOptions(slices.Sorted(set.Union(xslices.Map(rows, func(r marketOrderRow) set.Set[string] {
		return r.tags
	})...).All()))
	a.selectType.SetOptions(xslices.Map(rows, func(r marketOrderRow) string {
		return r.typeName
	}))
	a.rowsFiltered = rows
	a.body.Refresh()
}

func (a *marketOrders) update() {
	rows := make([]marketOrderRow, 0)
	t, i, err := func() (string, widget.Importance, error) {
		cc, err := a.fetchData(a.u.services(), a.isBuyOrders)
		if err != nil {
			return "", 0, err
		}
		if len(cc) == 0 {
			return "No characters", widget.LowImportance, nil
		}
		rows = cc
		return "", widget.MediumImportance, nil
	}()
	if err != nil {
		slog.Error("Failed to refresh locations UI", "err", err)
		t = "ERROR: " + a.u.humanizeError(err)
		i = widget.DangerImportance
	}
	fyne.Do(func() {
		if t != "" {
			a.bottom.Text = t
			a.bottom.Importance = i
			a.bottom.Refresh()
			a.bottom.Show()
		} else {
			a.bottom.Hide()
		}
	})
	fyne.Do(func() {
		a.rows = rows
		a.filterRows(-1)
	})
}

func (*marketOrders) fetchData(s services, isBuyOrders bool) ([]marketOrderRow, error) {
	ctx := context.Background()
	orders, err := s.cs.ListAllMarketOrder(ctx, isBuyOrders)
	if err != nil {
		return nil, err
	}
	rows := make([]marketOrderRow, 0)
	for _, o := range orders {
		r := marketOrderRow{
			characterID:   o.CharacterID,
			characterName: s.scs.CharacterName(o.CharacterID),
			escrow:        o.Escrow,
			expires:       o.Issued.Add(time.Duration(o.Duration) * time.Hour * 24),
			IsBuyOrder:    o.IsBuyOrder,
			isCorporation: o.IsCorporation,
			locationID:    o.Location.ID,
			locationName:  o.Location.Name.ValueOrFallback("?"),
			location:      o.Location,
			minVolume:     o.MinVolume,
			orderID:       o.OrderID,
			regionID:      o.Region.ID,
			regionName:    o.Region.Name,
			price:         o.Price,
			state:         o.State,
			typeID:        o.Type.ID,
			typeName:      o.Type.Name,
			volumeRemain:  o.VolumeRemains,
			volumeTotal:   o.VolumeTotal,
		}
		tags, err := s.cs.ListTagsForCharacter(ctx, o.CharacterID)
		if err != nil {
			return nil, err
		}
		r.tags = tags
		rows = append(rows, r)
	}
	return rows, nil
}

// showMarketOrderWindow shows the location of a character in a new window.
func showMarketOrderWindow(u *baseUI, r marketOrderRow) {
	title := fmt.Sprintf("Market Order #%d", r.orderID)
	w, ok := u.getOrCreateWindow(
		fmt.Sprintf("market-order-%d-%d", r.characterID, r.orderID),
		title,
		r.characterName,
	)
	if !ok {
		w.Show()
		return
	}
	item := makeLinkLabelWithWrap(r.typeName, func() {
		u.ShowTypeInfoWindowWithCharacter(r.typeID, r.characterID)
	})
	region := makeLinkLabel(r.regionName, func() {
		u.ShowInfoWindow(app.EveEntityRegion, r.regionID)
	})
	var buySell string
	if r.IsBuyOrder {
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
		widget.NewFormItem("Owner", makeOwnerActionLabel(
			r.characterID,
			r.characterName,
			u.ShowEveEntityInfoWindow,
		)),
		widget.NewFormItem("Type", item),
		widget.NewFormItem("Price", widget.NewLabel(formatISKAmount(r.price))),
		widget.NewFormItem("Variant", widget.NewLabel(buySell)),
		widget.NewFormItem("State", state),
		widget.NewFormItem("Volume Total", widget.NewLabel(ihumanize.Comma(r.volumeTotal))),
		widget.NewFormItem("Volume Remain", widget.NewLabel(ihumanize.Comma(r.volumeRemain))),
		widget.NewFormItem("Volume Min", widget.NewLabel(r.minVolume.StringFunc("-", func(v int) string {
			return ihumanize.Comma(v)
		}))),
		widget.NewFormItem("Expires at", widget.NewLabel(expires)),
		widget.NewFormItem("Location", makeLocationLabel(r.location, u.ShowLocationInfoWindow)),
		widget.NewFormItem("Region", region),
		widget.NewFormItem("For corporation", makeBoolLabel(r.isCorporation)),
	}
	if r.IsBuyOrder {
		items = slices.Insert(items, 8, widget.NewFormItem("Escrow", widget.NewLabel(r.escrow.StringFunc("-", func(v float64) string {
			return humanize.FormatFloat(app.FloatFormat, v)
		}))))
	}

	if u.IsDeveloperMode() {
		items = append(items, widget.NewFormItem("Order ID", u.makeCopyToClipboardLabel(fmt.Sprint(r.orderID))))
	}
	f := widget.NewForm(items...)
	f.Orientation = widget.Adaptive
	setDetailWindow(detailWindowParams{
		content: f,
		imageAction: func() {
			u.ShowTypeInfoWindow(r.typeID)
		},
		imageLoader: func() (fyne.Resource, error) {
			return u.eis.InventoryTypeIcon(r.typeID, 256)
		},
		title:  title,
		window: w,
	})
	w.Show()
}
