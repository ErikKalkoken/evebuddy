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
	"github.com/ErikKalkoken/go-set"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
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
	issued        time.Time
	location      *app.EveLocationShort
	locationID    int64
	locationName  string
	minVolume     optional.Optional[int]
	orderID       int64
	ownerID       int32
	owner         *app.EveEntity
	ownerName     string
	price         float64
	range_        string
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
	return xstrings.Title(r.stateCorrected().String())
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

func (r marketOrderRow) stateDisplay() string {
	if r.stateCorrected() == app.OrderOpen {
		return r.remaining()
	}
	return r.stateCorrectedDisplay()

}

func (r marketOrderRow) volumeDisplay() string {
	return fmt.Sprintf("%s / %s", ihumanize.Comma(r.volumeRemain), ihumanize.Comma(r.volumeTotal))
}

type marketOrders struct {
	widget.BaseWidget

	columnSorter *iwidget.ColumnSorter
	footer       *widget.Label
	isBuyOrders  bool
	issue        *widget.Label
	main         fyne.CanvasObject
	rows         []marketOrderRow
	rowsFiltered []marketOrderRow
	selectOwner  *kxwidget.FilterChipSelect
	selectRegion *kxwidget.FilterChipSelect
	selectState  *kxwidget.FilterChipSelect
	selectTag    *kxwidget.FilterChipSelect
	selectType   *kxwidget.FilterChipSelect
	sortButton   *iwidget.SortButton
	u            *baseUI
}

const (
	marketOrdersColType     = 0
	marketOrdersColVolume   = 1
	marketOrdersColPrice    = 2
	marketOrdersColState    = 3
	marketOrdersColLocation = 4
	marketOrdersColRegion   = 5
	marketOrdersColOwner    = 6
)

func newMarketOrders(u *baseUI, isBuyOrders bool) *marketOrders {
	headers := iwidget.NewDataTableDef([]iwidget.ColumnDef{{
		Col:   marketOrdersColType,
		Label: "Type",
		Width: columnWidthEntity,
	}, {
		Col:   marketOrdersColVolume,
		Label: "Quantity",
		Width: 100,
	}, {
		Col:   marketOrdersColPrice,
		Label: "Price",
		Width: 100,
	}, {
		Col:   marketOrdersColState,
		Label: "State",
		Width: 100,
	}, {
		Col:   marketOrdersColLocation,
		Label: "Location",
		Width: columnWidthLocation,
	}, {
		Col:   marketOrdersColRegion,
		Label: "Region",
		Width: columnWidthRegion,
	}, {
		Col:   marketOrdersColOwner,
		Label: "Owner",
		Width: columnWidthEntity,
	}})
	a := &marketOrders{
		columnSorter: headers.NewColumnSorter(marketOrdersColType, iwidget.SortAsc),
		footer:       widget.NewLabel(""),
		isBuyOrders:  isBuyOrders,
		issue:        makeTopLabel(),
		rows:         make([]marketOrderRow, 0),
		rowsFiltered: make([]marketOrderRow, 0),
		u:            u,
	}
	a.ExtendBaseWidget(a)
	makeCell := func(col int, r marketOrderRow) []widget.RichTextSegment {
		switch col {
		case marketOrdersColType:
			return iwidget.RichTextSegmentsFromText(r.typeName)
		case marketOrdersColVolume:
			return iwidget.RichTextSegmentsFromText(r.volumeDisplay(), widget.RichTextStyle{
				Alignment: fyne.TextAlignTrailing,
			})
		case marketOrdersColPrice:
			return iwidget.RichTextSegmentsFromText(humanize.FormatFloat(app.FloatFormat, r.price), widget.RichTextStyle{
				Alignment: fyne.TextAlignTrailing,
			})
		case marketOrdersColState:
			return iwidget.RichTextSegmentsFromText(r.stateDisplay(), widget.RichTextStyle{
				ColorName: r.stateColor(),
			})
		case marketOrdersColLocation:
			return iwidget.RichTextSegmentsFromText(r.locationName)
		case marketOrdersColRegion:
			return iwidget.RichTextSegmentsFromText(r.regionName)
		case marketOrdersColOwner:
			return iwidget.RichTextSegmentsFromText(r.ownerName)
		}
		return iwidget.RichTextSegmentsFromText("?")
	}
	if !a.u.isMobile {
		a.main = iwidget.MakeDataTable(
			headers,
			&a.rowsFiltered,
			makeCell,
			a.columnSorter,
			a.filterRows, func(_ int, r marketOrderRow) {
				showMarketOrderWindow(a.u, r)
			})
	} else {
		a.main = a.makeDataList()
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
	a.sortButton = a.columnSorter.NewSortButton(func() {
		a.filterRows(-1)
	}, a.u.window)

	a.u.characterSectionChanged.AddListener(func(_ context.Context, arg characterSectionUpdated) {
		switch arg.section {
		case app.SectionCharacterMarketOrders:
			a.update()
		}
	})
	a.u.characterAdded.AddListener(func(_ context.Context, _ *app.Character) {
		a.update()
	})
	a.u.characterRemoved.AddListener(func(_ context.Context, _ *app.EntityShort[int32]) {
		a.update()
	})
	a.u.tagsChanged.AddListener(func(ctx context.Context, s struct{}) {
		a.update()
	})
	a.u.refreshTickerExpired.AddListener(func(_ context.Context, _ struct{}) {
		fyne.Do(func() {
			a.main.Refresh()
		})
	})
	return a
}

func (a *marketOrders) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHBox(a.selectType, a.selectState, a.selectRegion, a.selectOwner, a.selectTag)
	if a.u.isMobile {
		filter.Add(a.sortButton)
	}
	p := theme.Padding()
	c := container.NewBorder(
		container.NewVBox(container.NewHScroll(filter), a.issue),
		container.New(layout.NewCustomPaddedLayout(p, p, 0, 0), a.footer),
		nil,
		nil,
		a.main,
	)
	return widget.NewSimpleRenderer(c)
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
			b1[0].(*widget.Label).SetText(ihumanize.NumberF(r.price, 2) + " ISK")
			b1[1].(*widget.Label).SetText(r.volumeDisplay())

			c[2].(*iwidget.RichText).Set(r.location.DisplayRichText())
			c[3].(*widget.Label).SetText(r.ownerName)
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
	a.columnSorter.Sort(sortCol, func(sortCol int, dir iwidget.SortDir) {
		slices.SortFunc(rows, func(a, b marketOrderRow) int {
			var x int
			switch sortCol {
			case marketOrdersColType:
				x = strings.Compare(a.typeName, b.typeName)
			case marketOrdersColVolume:
				x = cmp.Compare(a.volumeRemain, b.volumeRemain)
			case marketOrdersColPrice:
				x = cmp.Compare(a.price, b.price)
			case marketOrdersColState:
				x = a.expires.Compare(b.expires)
			case marketOrdersColRegion:
				x = strings.Compare(a.regionName, b.regionName)
			case marketOrdersColLocation:
				x = strings.Compare(a.locationName, b.locationName)
			case marketOrdersColOwner:
				x = xstrings.CompareIgnoreCase(a.ownerName, b.ownerName)
			}
			if dir == iwidget.SortAsc {
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
		return r.ownerName
	}))
	a.selectTag.SetOptions(slices.Sorted(set.Union(xslices.Map(rows, func(r marketOrderRow) set.Set[string] {
		return r.tags
	})...).All()))
	a.selectType.SetOptions(xslices.Map(rows, func(r marketOrderRow) string {
		return r.typeName
	}))
	a.rowsFiltered = rows
	a.main.Refresh()
	var total optional.Optional[float64]
	for _, r := range rows {
		total.Set(total.ValueOrZero() + r.price*float64(r.volumeRemain))
	}
	a.footer.SetText(fmt.Sprintf("Orders total: %s ISK", total.StringFunc("?", func(v float64) string {
		return ihumanize.NumberF(v, 1)
	})))
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
			a.issue.Text = t
			a.issue.Importance = i
			a.issue.Refresh()
			a.issue.Show()
		} else {
			a.issue.Hide()
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
			issued:        o.Issued,
			location:      o.Location,
			locationID:    o.Location.ID,
			locationName:  o.Location.Name.ValueOrFallback("?"),
			minVolume:     o.MinVolume,
			orderID:       o.OrderID,
			ownerID:       o.Owner.ID,
			owner:         o.Owner,
			ownerName:     o.Owner.Name,
			price:         o.Price,
			range_:        o.Range,
			regionID:      o.Region.ID,
			regionName:    o.Region.Name,
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
	w, created := u.getOrCreateWindow(
		fmt.Sprintf("market-order-%d-%d", r.characterID, r.orderID),
		title,
		r.characterName,
	)
	if !created {
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
		widget.NewFormItem("Owner", makeEveEntityActionLabel(
			r.owner,
			u.ShowEveEntityInfoWindow,
		)),
		widget.NewFormItem("Type", item),
		widget.NewFormItem("Price", widget.NewLabel(formatISKAmount(r.price))),
		widget.NewFormItem("Variant", widget.NewLabel(buySell)),
		widget.NewFormItem("State", state),
		widget.NewFormItem("Volume Total", widget.NewLabel(ihumanize.Comma(r.volumeTotal))),
		widget.NewFormItem("Volume Remain", widget.NewLabel(ihumanize.Comma(r.volumeRemain))),
		widget.NewFormItem("Issued", widget.NewLabel(r.issued.Format(app.DateTimeFormat))),
		widget.NewFormItem("Expires", widget.NewLabel(expires)),
		widget.NewFormItem("Location", makeLocationLabel(r.location, u.ShowLocationInfoWindow)),
		widget.NewFormItem("Region", region),
	}
	if r.IsBuyOrder {
		items = append(items, widget.NewFormItem(
			"Volume Min",
			widget.NewLabel(r.minVolume.StringFunc("?", func(v int) string {
				return ihumanize.Comma(v)
			})),
		))
		items = append(items, widget.NewFormItem(
			"Escrow",
			widget.NewLabel(r.escrow.StringFunc("-", func(v float64) string {
				return humanize.FormatFloat(app.FloatFormat, v)
			})),
		))
	}
	items = append(items, widget.NewFormItem("Range", widget.NewLabel(xstrings.Title(r.range_))))
	items = append(items, widget.NewFormItem("For corporation", makeBoolLabel(r.isCorporation)))
	items = append(items, widget.NewFormItem("Character", makeCharacterActionLabel(
		r.characterID,
		r.characterName,
		u.ShowEveEntityInfoWindow,
	)))

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
