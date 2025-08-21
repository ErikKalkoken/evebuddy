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
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

// [ ] Create detail window
// [ ] Create custom version for mobile

const (
	marketOrderStateActive  = "Active"
	marketOrderStateHistory = "History"
)

type marketOrderRow struct {
	characterID   int32
	characterName string
	expires       time.Time
	state         app.MarketOrderState
	locationID    int64
	locationName  string
	regionID      int32
	regionName    string
	tags          set.Set[string]
	typeID        int32
	typeName      string
	volumeRemain  int
	volumeTotal   int
}

func (r marketOrderRow) stateCorrected() app.MarketOrderState {
	if r.state == app.OrderOpen && time.Until(r.expires) <= 0 {
		return app.OrderExpired
	}
	return r.state
}

type marketOrders struct {
	widget.BaseWidget

	body         fyne.CanvasObject
	bottom       *widget.Label
	columnSorter *columnSorter
	isBuyOrders  bool
	rows         []marketOrderRow
	rowsFiltered []marketOrderRow
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
		{label: "Station", width: columnWidthLocation},
		{label: "Region", width: columnWidthRegion},
		{label: "State", width: 100},
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
			return iwidget.RichTextSegmentsFromText(r.locationName)
		case 3:
			return iwidget.RichTextSegmentsFromText(r.regionName)
		case 4:
			s := r.stateCorrected()
			if s == app.OrderOpen {
				s := ihumanize.Duration(time.Until(r.expires))
				return iwidget.RichTextSegmentsFromText(s)
			}
			return iwidget.RichTextSegmentsFromText(s.String())
		case 5:
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
		a.body = makeDataList(headers, &a.rowsFiltered, makeCell, func(r marketOrderRow) {
			showMarketOrderWindow(u, r)
		})
	}

	a.selectRegion = kxwidget.NewFilterChipSelect("Region", []string{}, func(string) {
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
	filter := container.NewHBox(a.selectType, a.selectState, a.selectRegion, a.selectTag)
	if !a.u.isDesktop {
		filter.Add(a.sortButton)
	}
	c := container.NewBorder(container.NewHScroll(filter), a.bottom, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}

// func (a *marketOrders) makeDataList() *iwidget.StripedList {
// 	p := theme.Padding()
// 	var l *iwidget.StripedList
// 	l = iwidget.NewStripedList(
// 		func() int {
// 			return len(a.rowsFiltered)
// 		},
// 		func() fyne.CanvasObject {
// 			character := widget.NewLabel("Template")
// 			character.Wrapping = fyne.TextWrapWord
// 			character.SizeName = theme.SizeNameSubHeadingText
// 			location := iwidget.NewRichTextWithText("Template")
// 			location.Wrapping = fyne.TextWrapWord
// 			ship := widget.NewLabel("Template")
// 			return container.New(layout.NewCustomPaddedVBoxLayout(-p),
// 				character,
// 				location,
// 				ship,
// 			)
// 		},
// 		func(id widget.ListItemID, co fyne.CanvasObject) {
// 			if id < 0 || id >= len(a.rowsFiltered) {
// 				return
// 			}
// 			r := a.rowsFiltered[id]
// 			c := co.(*fyne.Container).Objects
// 			c[0].(*widget.Label).SetText(r.characterName)
// 			c[1].(*iwidget.RichText).Set(r.locationDisplay)
// 			c[2].(*widget.Label).SetText(r.typeName)
// 			l.SetItemHeight(id, co.(*fyne.Container).MinSize().Height)
// 		},
// 	)
// 	l.OnSelected = func(id widget.ListItemID) {
// 		defer l.UnselectAll()
// 		if id < 0 || id >= len(a.rowsFiltered) {
// 			return
// 		}
// 		r := a.rowsFiltered[id]
// 		showMarketOrderWindow(a.u, r)
// 	}
// 	return l
// }

func (a *marketOrders) filterRows(sortCol int) {
	rows := slices.Clone(a.rows)
	// filter
	if x := a.selectRegion.Selected; x != "" {
		rows = xslices.Filter(rows, func(r marketOrderRow) bool {
			return r.regionName == x
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
				x = strings.Compare(a.regionName, b.regionName)
			case 3:
				x = strings.Compare(a.locationName, b.locationName)
			case 4:
				x = a.expires.Compare(b.expires)
			case 5:
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
	a.selectTag.SetOptions(slices.Sorted(set.Union(xslices.Map(rows, func(r marketOrderRow) set.Set[string] {
		return r.tags
	})...).All()))
	a.selectRegion.SetOptions(xslices.Map(rows, func(r marketOrderRow) string {
		return r.regionName
	}))
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
			expires:       o.Issued.Add(time.Duration(o.Duration) * time.Hour * 24),
			locationID:    o.Location.ID,
			locationName:  o.Location.Name,
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
	w, ok := u.getOrCreateWindow(fmt.Sprintf("location-%d", r.characterID), "Character Location", r.characterName)
	if !ok {
		w.Show()
		return
	}
	ship := makeLinkLabelWithWrap(r.typeName, func() {
		u.ShowTypeInfoWindowWithCharacter(r.typeID, r.characterID)
	})
	var location fyne.CanvasObject
	// if r.location != nil {
	// 	location = makeLocationLabel(r.location, u.ShowLocationInfoWindow)
	// } else {
	// 	location = widget.NewLabel("?")
	// }
	var region fyne.CanvasObject
	if r.regionID != 0 {
		region = makeLinkLabel(r.regionName, func() {
			u.ShowInfoWindow(app.EveEntityRegion, r.regionID)
		})
	} else {
		region = widget.NewLabel(r.regionName)
	}
	fi := []*widget.FormItem{
		widget.NewFormItem("Owner", makeOwnerActionLabel(
			r.characterID,
			r.characterName,
			u.ShowEveEntityInfoWindow,
		)),
		widget.NewFormItem("Location", location),
		widget.NewFormItem("Region", region),
		widget.NewFormItem("Ship", ship),
	}

	f := widget.NewForm(fi...)
	f.Orientation = widget.Adaptive
	subTitle := fmt.Sprintf("Location of %s", r.characterName)
	setDetailWindow(detailWindowParams{
		content: f,
		minSize: fyne.NewSize(500, 250),
		imageAction: func() {
			u.ShowInfoWindow(app.EveEntityCharacter, r.characterID)
		},
		imageLoader: func() (fyne.Resource, error) {
			return u.eis.CharacterPortrait(r.characterID, 512)
		},
		title:  subTitle,
		window: w,
	})
	w.Show()
}
