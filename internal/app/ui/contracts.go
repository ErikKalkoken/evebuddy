package ui

import (
	"context"
	"fmt"
	"log/slog"
	"math"
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
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

const (
	contractStatusAllActive   = "All active"
	contractStatusOutstanding = "Outstanding"
	contractStatusInProgress  = "In progress"
	contractStatusHasIssue    = "Has issues"
	contractStatusHistory     = "History"
)

type contractRow struct {
	assigneeName string
	characterID  int32
	contractID   int32
	dateExpired  time.Time
	dateIssued   time.Time
	isExpired    bool
	issuerName   string
	name         string
	status       app.ContractStatus
	statusText   string
	typeName     string
	isActive     bool
	hasIssue     bool
	isHistory    bool
}

func (r contractRow) dateExpiredDisplay() []widget.RichTextSegment {
	var text string
	var color fyne.ThemeColorName
	if r.isExpired {
		text = "EXPIRED"
		color = theme.ColorNameError
	} else {
		text = ihumanize.RelTime(r.dateExpired)
		color = theme.ColorNameForeground
	}
	return iwidget.NewRichTextSegmentFromText(text, widget.RichTextStyle{
		ColorName: color,
	})
}

type contracts struct {
	widget.BaseWidget

	OnUpdate func(active int)

	body           fyne.CanvasObject
	columnSorter   *columnSorter
	rows           []contractRow
	rowsFiltered   []contractRow
	selectAssignee *selectFilter
	selectIssuer   *selectFilter
	selectStatus   *widget.Select
	selectType     *selectFilter
	sortButton     *sortButton
	top            *widget.Label
	u              *BaseUI
}

func newContracts(u *BaseUI) *contracts {
	headers := []headerDef{
		{Text: "Contract", Width: 300},
		{Text: "Type", Width: 120},
		{Text: "From", Width: 150},
		{Text: "To", Width: 150},
		{Text: "Status", Width: 100},
		{Text: "Date Issued", Width: 150},
		{Text: "Time Left", Width: 100},
	}
	a := &contracts{
		columnSorter: newColumnSorter(headers),
		rows:         make([]contractRow, 0),
		top:          makeTopLabel(),
		u:            u,
	}
	a.ExtendBaseWidget(a)
	if a.u.isDesktop {
		a.body = makeDataTable(headers, &a.rowsFiltered,
			func(col int, r contractRow) []widget.RichTextSegment {
				switch col {
				case 0:
					return iwidget.NewRichTextSegmentFromText(r.name)
				case 1:
					return iwidget.NewRichTextSegmentFromText(r.typeName)
				case 2:
					return iwidget.NewRichTextSegmentFromText(r.issuerName)
				case 3:
					return iwidget.NewRichTextSegmentFromText(r.assigneeName)
				case 4:
					return r.status.DisplayRichText()
				case 5:
					return iwidget.NewRichTextSegmentFromText(r.dateIssued.Format(app.DateTimeFormat))
				case 6:
					return r.dateExpiredDisplay()
				}
				return iwidget.NewRichTextSegmentFromText("?")
			}, a.columnSorter, a.filterRows, func(column int, r contractRow) {
				a.showContract(r)
			},
		)
	} else {
		a.body = a.makeDataList()
	}

	a.selectAssignee = newSelectFilter("Any assignee", func() {
		a.filterRows(-1)
	})
	a.selectIssuer = newSelectFilter("Any issuer", func() {
		a.filterRows(-1)
	})
	a.selectType = newSelectFilter("Any type", func() {
		a.filterRows(-1)
	})

	a.selectStatus = widget.NewSelect([]string{
		contractStatusAllActive,
		contractStatusOutstanding,
		contractStatusInProgress,
		contractStatusHasIssue,
		contractStatusHistory,
	}, func(string) {
		a.filterRows(-1)
	})
	a.selectStatus.Selected = contractStatusAllActive

	a.sortButton = a.columnSorter.newSortButton(headers, func() {
		a.filterRows(-1)
	}, a.u.window)

	return a
}

func (a *contracts) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHBox(a.selectType, a.selectIssuer, a.selectAssignee, a.selectStatus)
	if !a.u.isDesktop {
		filter.Add(a.sortButton)
	}
	c := container.NewBorder(
		container.NewVBox(a.top, container.NewHScroll(filter)), nil, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *contracts) makeDataList() *widget.List {
	p := theme.Padding()
	l := widget.NewList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			title := widget.NewLabelWithStyle("Template", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			type_ := widget.NewLabel("Template")
			status := widget.NewRichTextWithText("Template")
			issuer := widget.NewLabel("Template")
			assignee := widget.NewLabel("Template")
			dateExpired := widget.NewRichTextWithText("Template")
			return container.New(layout.NewCustomPaddedVBoxLayout(-p),
				title,
				container.NewHBox(type_, layout.NewSpacer(), status),
				issuer,
				assignee,
				dateExpired,
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id < 0 || id >= len(a.rowsFiltered) {
				return
			}
			r := a.rowsFiltered[id]
			main := co.(*fyne.Container).Objects
			main[0].(*widget.Label).SetText(r.name)
			box := main[1].(*fyne.Container).Objects
			box[0].(*widget.Label).SetText(r.typeName)
			iwidget.SetRichText(box[2].(*widget.RichText), r.status.DisplayRichText()...)

			main[2].(*widget.Label).SetText("From " + r.issuerName)
			assignee := "To "
			if r.assigneeName == "" {
				assignee += "..."
			} else {
				assignee += r.assigneeName
			}
			main[3].(*widget.Label).SetText(assignee)

			iwidget.SetRichText(main[4].(*widget.RichText), iwidget.InlineRichTextSegments(
				iwidget.NewRichTextSegmentFromText("Expires "),
				r.dateExpiredDisplay(),
			)...)
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(a.rowsFiltered) {
			return
		}
		a.showContract(a.rowsFiltered[id])
	}
	return l
}

func (a *contracts) filterRows(sortCol int) {
	rows := slices.Clone(a.rows)
	// filter
	a.selectIssuer.applyFilter(func(selected string) {
		rows = xslices.Filter(rows, func(r contractRow) bool {
			return r.issuerName == selected
		})
	})
	a.selectAssignee.applyFilter(func(selected string) {
		rows = xslices.Filter(rows, func(r contractRow) bool {
			return r.assigneeName == selected
		})
	})
	rows = xslices.Filter(rows, func(r contractRow) bool {
		switch a.selectStatus.Selected {
		case contractStatusAllActive:
			return r.isActive
		case contractStatusOutstanding:
			return r.status == app.ContractStatusOutstanding
		case contractStatusInProgress:
			return r.status == app.ContractStatusInProgress
		case contractStatusHasIssue:
			return r.hasIssue
		case contractStatusHistory:
			return r.isHistory
		}
		return false
	})
	a.selectType.applyFilter(func(selected string) {
		rows = xslices.Filter(rows, func(r contractRow) bool {
			return r.typeName == selected
		})
	})
	// sort
	a.columnSorter.sort(sortCol, func(sortCol int, dir sortDir) {
		slices.SortFunc(rows, func(a, b contractRow) int {
			var x int
			switch sortCol {
			case 0:
				x = strings.Compare(a.name, b.name)
			case 1:
				x = strings.Compare(a.typeName, b.typeName)
			case 2:
				x = strings.Compare(a.issuerName, b.issuerName)
			case 3:
				x = strings.Compare(a.assigneeName, b.assigneeName)
			case 4:
				x = strings.Compare(a.statusText, b.statusText)
			case 5:
				x = a.dateIssued.Compare(b.dateIssued)
			case 6:
				x = a.dateExpired.Compare(b.dateExpired)
			}
			if dir == sortAsc {
				return x
			} else {
				return -1 * x
			}
		})
	})
	a.selectIssuer.setOptions(xiter.MapSlice(rows, func(r contractRow) string {
		return r.issuerName
	}))
	a.selectAssignee.setOptions(xiter.MapSlice(rows, func(r contractRow) string {
		return r.assigneeName
	}))
	a.selectType.setOptions(xiter.MapSlice(rows, func(r contractRow) string {
		return r.typeName
	}))
	a.rowsFiltered = rows
	a.body.Refresh()
}

func (a *contracts) update() {
	contracts, err := a.u.cs.ListAllContracts(context.Background())
	if err != nil {
		slog.Error("Failed to refresh contracts UI", "err", err)
		fyne.Do(func() {
			a.top.Text = fmt.Sprintf("ERROR: %s", a.u.humanizeError(err))
			a.top.Importance = widget.DangerImportance
			a.top.Refresh()
			a.top.Show()
		})
		return
	}
	rows := make([]contractRow, 0)
	var activeCount int
	for _, c := range contracts {
		r := contractRow{
			name:         c.NameDisplay(),
			typeName:     c.TypeDisplay(),
			issuerName:   c.IssuerEffective().Name,
			assigneeName: c.AssigneeName(),
			statusText:   c.StatusDisplay(),
			status:       c.Status,
			dateIssued:   c.DateIssued,
			dateExpired:  c.DateExpired,
			isExpired:    c.IsExpired(),
			characterID:  c.CharacterID,
			contractID:   c.ContractID,
			isActive:     c.IsActive(),
			isHistory:    c.IsCompleted(),
			hasIssue:     c.HasIssue(),
		}
		rows = append(rows, r)
		if c.IsActive() {
			activeCount++
		}
	}
	fyne.Do(func() {
		a.top.Hide()
		a.rows = rows
		a.filterRows(-1)
	})
	if a.OnUpdate != nil {
		a.OnUpdate(activeCount)
	}
}

func (a *contracts) showContract(r contractRow) {
	c, err := a.u.cs.GetContract(context.Background(), r.characterID, r.contractID)
	if err != nil {
		panic(err)
	}
	var w fyne.Window
	makeExpiresString := func(c *app.CharacterContract) string {
		ts := c.DateExpired.Format(app.DateTimeFormat)
		var ds string
		if c.IsExpired() {
			ds = "EXPIRED"
		} else {
			ds = ihumanize.RelTime(c.DateExpired)
		}
		return fmt.Sprintf("%s (%s)", ts, ds)
	}
	makeEntity := func(ee *app.EveEntity) *kxwidget.TappableLabel {
		return kxwidget.NewTappableLabel(ee.Name, func() {
			a.u.ShowEveEntityInfoWindow(ee)
		})
	}
	makeLocation := func(l *app.EveLocationShort) *iwidget.TappableRichText {
		if l == nil {
			return iwidget.NewTappableRichTextWithText("?", nil)
		}
		x := iwidget.NewTappableRichText(func() {
			a.u.ShowLocationInfoWindow(l.ID)
		},
			l.DisplayRichText()...,
		)
		return x
	}
	makeISKString := func(v float64) string {
		t := humanize.Commaf(v) + " ISK"
		if math.Abs(v) > 999 {
			t += fmt.Sprintf(" (%s)", ihumanize.Number(v, 1))
		}
		return t
	}

	availability := container.NewHBox(widget.NewLabel(c.AvailabilityDisplay()))
	if c.Assignee != nil {
		availability.Add(makeEntity(c.Assignee))
	}
	fi := []*widget.FormItem{
		widget.NewFormItem("Info by issuer", widget.NewLabel(c.TitleDisplay())),
		widget.NewFormItem("Type", widget.NewLabel(c.TypeDisplay())),
		widget.NewFormItem("Issued By", makeEntity(c.IssuerEffective())),
		widget.NewFormItem("Availability", availability),
	}
	if a.u.IsDeveloperMode() {
		fi = append(fi, widget.NewFormItem("Contract ID", a.u.makeCopyToClipboardLabel(fmt.Sprint(c.ContractID))))
	}
	if c.Type == app.ContractTypeCourier {
		fi = append(fi, widget.NewFormItem("Contractor", widget.NewLabel(c.ContractorDisplay())))
	}
	fi = append(fi, widget.NewFormItem("Status", widget.NewRichText(c.StatusDisplayRichText()...)))
	fi = append(fi, widget.NewFormItem("Location", makeLocation(c.StartLocation)))

	if c.Type == app.ContractTypeCourier || c.Type == app.ContractTypeItemExchange {
		fi = append(fi, widget.NewFormItem("Date Issued", widget.NewLabel(c.DateIssued.Format(app.DateTimeFormat))))
		fi = append(fi, widget.NewFormItem("Date Accepted", widget.NewLabel(c.DateAccepted.StringFunc("", func(v time.Time) string {
			return v.Format(app.DateTimeFormat)
		}))))
		fi = append(fi, widget.NewFormItem("Date Expired", widget.NewLabel(makeExpiresString(c))))
		fi = append(fi, widget.NewFormItem("Date Completed", widget.NewLabel(c.DateCompleted.StringFunc("", func(v time.Time) string {
			return v.Format(app.DateTimeFormat)
		}))))
	}

	switch c.Type {
	case app.ContractTypeCourier:
		var collateral string
		if c.Collateral == 0 {
			collateral = "(None)"
		} else {
			collateral = makeISKString(c.Collateral)
		}
		fi = slices.Concat(fi, []*widget.FormItem{
			{Text: "Complete In", Widget: widget.NewLabel(fmt.Sprintf("%d days", c.DaysToComplete))},
			{Text: "Volume", Widget: widget.NewLabel(fmt.Sprintf("%f m3", c.Volume))},
			{Text: "Reward", Widget: widget.NewLabel(makeISKString(c.Reward))},
			{Text: "Collateral", Widget: widget.NewLabel(collateral)},
			{Text: "Destination", Widget: makeLocation(c.EndLocation)},
		})
	case app.ContractTypeItemExchange:
		if c.Price > 0 {
			x := widget.NewLabel(makeISKString(c.Price))
			x.Importance = widget.DangerImportance
			fi = append(fi, widget.NewFormItem("Buyer Will Pay", x))
		} else {
			x := widget.NewLabel(makeISKString(c.Reward))
			x.Importance = widget.SuccessImportance
			fi = append(fi, widget.NewFormItem("Buyer Will Get", x))
		}
	case app.ContractTypeAuction:
		ctx := context.TODO()
		total, err := a.u.cs.CountContractBids(ctx, c.ID)
		if err != nil {
			d := a.u.NewErrorDialog("Failed to count contract bids", err, w)
			d.SetOnClosed(w.Hide)
			d.Show()
		}
		var currentBid string
		if total == 0 {
			currentBid = "(None)"
		} else {
			top, err := a.u.cs.GetContractTopBid(ctx, c.ID)
			if err != nil {
				d := a.u.NewErrorDialog("Failed to get top bid", err, w)
				d.SetOnClosed(w.Hide)
				d.Show()
			}
			currentBid = fmt.Sprintf("%s (%d bids so far)", makeISKString(float64(top.Amount)), total)
		}
		fi = slices.Concat(fi, []*widget.FormItem{
			{Text: "Starting Bid", Widget: widget.NewLabel(makeISKString(c.Price))},
			{Text: "Buyout Price", Widget: widget.NewLabel(makeISKString(c.Buyout))},
			{Text: "Current Bid", Widget: widget.NewLabel(currentBid)},
			{Text: "Expires", Widget: widget.NewLabel(makeExpiresString(c))},
		})
	}

	makeItemsInfo := func(c *app.CharacterContract) fyne.CanvasObject {
		vb := container.NewVBox()
		items, err := a.u.cs.ListContractItems(context.TODO(), c.ID)
		if err != nil {
			d := a.u.NewErrorDialog("Failed to fetch contract items", err, w)
			d.SetOnClosed(w.Hide)
			d.Show()
		}
		var itemsIncluded, itemsRequested []*app.CharacterContractItem
		for _, it := range items {
			if it.IsIncluded {
				itemsIncluded = append(itemsIncluded, it)
			} else {
				itemsRequested = append(itemsRequested, it)
			}
		}
		makeItem := func(it *app.CharacterContractItem) fyne.CanvasObject {
			x := kxwidget.NewTappableLabel(it.Type.Name, func() {
				a.u.ShowTypeInfoWindow(it.Type.ID)
			})
			return container.NewHBox(
				x,
				widget.NewLabel(fmt.Sprintf("(%s)", it.Type.Group.Name)),
				widget.NewLabel(fmt.Sprintf("x %s ", humanize.Comma(int64(it.Quantity)))),
			)
		}
		// included items
		if len(itemsIncluded) > 0 {
			t := widget.NewLabel("Buyer Will Get")
			t.Importance = widget.SuccessImportance
			vb.Add(t)
			for _, it := range itemsIncluded {
				vb.Add(makeItem(it))
			}
		}
		// requested items
		if len(itemsRequested) > 0 {
			t := widget.NewLabel("Buyer Will Provide")
			t.Importance = widget.DangerImportance
			vb.Add(t)
			for _, it := range itemsRequested {
				vb.Add(makeItem(it))
			}
		}
		return vb
	}

	subTitle := fmt.Sprintf("%s (%s)", c.NameDisplay(), c.TypeDisplay())
	f := widget.NewForm(fi...)
	f.Orientation = widget.Adaptive
	main := container.NewVBox(f)
	if c.Type == app.ContractTypeItemExchange || c.Type == app.ContractTypeAuction {
		main.Add(widget.NewSeparator())
		main.Add(makeItemsInfo(c))
	}
	w = a.u.makeDetailWindow("Contract", subTitle, main)
	w.Show()
}
