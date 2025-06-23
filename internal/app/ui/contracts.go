package ui

import (
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
	"github.com/ErikKalkoken/evebuddy/internal/set"
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
	assigneeName       string
	characterID        int32
	contractID         int32
	dateExpired        time.Time
	dateExpiredDisplay []widget.RichTextSegment
	dateIssued         time.Time
	hasIssue           bool
	isActive           bool
	isExpired          bool
	isHistory          bool
	issuerName         string
	name               string
	status             app.ContractStatus
	statusText         string
	tags               set.Set[string]
	typeName           string
}

type contracts struct {
	widget.BaseWidget

	OnUpdate func(active int)

	body           fyne.CanvasObject
	columnSorter   *columnSorter
	rows           []contractRow
	rowsFiltered   []contractRow
	selectAssignee *kxwidget.FilterChipSelect
	selectIssuer   *kxwidget.FilterChipSelect
	selectStatus   *kxwidget.FilterChipSelect
	selectTag      *kxwidget.FilterChipSelect
	selectType     *kxwidget.FilterChipSelect
	sortButton     *sortButton
	bottom         *widget.Label
	u              *baseUI
}

func newContracts(u *baseUI) *contracts {
	headers := []headerDef{
		{label: "Contract", width: 300},
		{label: "Type", width: 120},
		{label: "From", width: 150},
		{label: "To", width: 150},
		{label: "Status", width: 100},
		{label: "Date Issued", width: columnWidthDateTime},
		{label: "Time Left", width: 100},
	}
	a := &contracts{
		columnSorter: newColumnSorter(headers),
		rows:         make([]contractRow, 0),
		bottom:       widget.NewLabel(""),
		u:            u,
	}
	a.ExtendBaseWidget(a)
	if a.u.isDesktop {
		a.body = makeDataTable(headers, &a.rowsFiltered,
			func(col int, r contractRow) []widget.RichTextSegment {
				switch col {
				case 0:
					return iwidget.RichTextSegmentsFromText(r.name)
				case 1:
					return iwidget.RichTextSegmentsFromText(r.typeName)
				case 2:
					return iwidget.RichTextSegmentsFromText(r.issuerName)
				case 3:
					return iwidget.RichTextSegmentsFromText(r.assigneeName)
				case 4:
					return r.status.DisplayRichText()
				case 5:
					return iwidget.RichTextSegmentsFromText(r.dateIssued.Format(app.DateTimeFormat))
				case 6:
					return r.dateExpiredDisplay
				}
				return iwidget.RichTextSegmentsFromText("?")
			}, a.columnSorter, a.filterRows, func(column int, r contractRow) {
				showContract(a.u, r.characterID, r.contractID)
			},
		)
	} else {
		a.body = a.makeDataList()
	}

	a.selectAssignee = kxwidget.NewFilterChipSelectWithSearch("Assignee", []string{}, func(string) {
		a.filterRows(-1)
	}, a.u.window)
	a.selectIssuer = kxwidget.NewFilterChipSelectWithSearch("Issuer", []string{}, func(string) {
		a.filterRows(-1)
	}, a.u.window)
	a.selectType = kxwidget.NewFilterChipSelect("Type", []string{}, func(string) {
		a.filterRows(-1)
	})

	a.selectStatus = kxwidget.NewFilterChipSelect("", []string{
		contractStatusAllActive,
		contractStatusOutstanding,
		contractStatusInProgress,
		contractStatusHasIssue,
		contractStatusHistory,
	}, func(string) {
		a.filterRows(-1)
	})
	a.selectStatus.Selected = contractStatusAllActive
	a.selectStatus.SortDisabled = true
	a.selectTag = kxwidget.NewFilterChipSelect("Tag", []string{}, func(string) {
		a.filterRows(-1)
	})
	a.sortButton = a.columnSorter.newSortButton(headers, func() {
		a.filterRows(-1)
	}, a.u.window)

	return a
}

func (a *contracts) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHBox(
		a.selectType,
		a.selectIssuer,
		a.selectAssignee,
		a.selectStatus,
		a.selectTag,
	)
	if !a.u.isDesktop {
		filter.Add(a.sortButton)
	}
	c := container.NewBorder(
		container.NewVBox(container.NewHScroll(filter)),
		a.bottom,
		nil,
		nil,
		a.body,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *contracts) makeDataList() *iwidget.StripedList {
	p := theme.Padding()
	l := iwidget.NewStripedList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			title := widget.NewLabelWithStyle("Template", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			type_ := widget.NewLabel("Template")
			status := iwidget.NewRichTextWithText("Template")
			issuer := widget.NewLabel("Template")
			assignee := widget.NewLabel("Template")
			dateExpired := iwidget.NewRichTextWithText("Template")
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
			(box[2].(*iwidget.RichText).Set(r.status.DisplayRichText()))

			main[2].(*widget.Label).SetText("From " + r.issuerName)
			assignee := "To "
			if r.assigneeName == "" {
				assignee += "..."
			} else {
				assignee += r.assigneeName
			}
			main[3].(*widget.Label).SetText(assignee)

			main[4].(*iwidget.RichText).Set(iwidget.InlineRichTextSegments(
				iwidget.RichTextSegmentsFromText("Expires "),
				r.dateExpiredDisplay,
			))
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(a.rowsFiltered) {
			return
		}
		r := a.rowsFiltered[id]
		showContract(a.u, r.characterID, r.contractID)
	}
	return l
}

func (a *contracts) filterRows(sortCol int) {
	rows := slices.Clone(a.rows)
	// filter
	if x := a.selectIssuer.Selected; x != "" {
		rows = xslices.Filter(rows, func(r contractRow) bool {
			return r.issuerName == x
		})
	}
	if x := a.selectAssignee.Selected; x != "" {
		rows = xslices.Filter(rows, func(r contractRow) bool {
			return r.assigneeName == x
		})
	}
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
	if x := a.selectType.Selected; x != "" {
		rows = xslices.Filter(rows, func(r contractRow) bool {
			return r.typeName == x
		})
	}
	if x := a.selectTag.Selected; x != "" {
		rows = xslices.Filter(rows, func(r contractRow) bool {
			return r.tags.Contains(x)
		})
	}
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
	// set data & refresh
	a.selectTag.SetOptions(slices.Sorted(set.Union(xslices.Map(rows, func(r contractRow) set.Set[string] {
		return r.tags
	})...).All()))
	a.selectIssuer.SetOptions(xslices.Map(rows, func(r contractRow) string {
		return r.issuerName
	}))
	a.selectAssignee.SetOptions(xslices.Map(rows, func(r contractRow) string {
		return r.assigneeName
	}))
	a.selectType.SetOptions(xslices.Map(rows, func(r contractRow) string {
		return r.typeName
	}))
	a.rowsFiltered = rows
	a.body.Refresh()
}

func (a *contracts) update() {
	rows, activeCount, err := a.fetchRows(a.u.services())
	if err != nil {
		slog.Error("Failed to refresh contracts UI", "err", err)
		fyne.Do(func() {
			a.bottom.Text = fmt.Sprintf("ERROR: %s", a.u.humanizeError(err))
			a.bottom.Importance = widget.DangerImportance
			a.bottom.Refresh()
			a.bottom.Show()
		})
		return
	}
	fyne.Do(func() {
		a.bottom.Hide()
		a.rows = rows
		a.filterRows(-1)
	})
	if a.OnUpdate != nil {
		a.OnUpdate(activeCount)
	}
}

func (a *contracts) fetchRows(s services) ([]contractRow, int, error) {
	ctx := context.Background()
	oo, err := s.cs.ListAllContracts(ctx)
	if err != nil {
		return nil, 0, err
	}
	// Remove duplicate contracts between the user's own characters
	contracts := slices.CompactFunc(oo, func(a, b *app.CharacterContract) bool {
		return a.Assignee != nil &&
			b.Assignee != nil &&
			a.ContractID == b.ContractID &&
			a.Type == b.Type &&
			a.Issuer.ID == b.Issuer.ID &&
			a.Assignee.ID == b.Assignee.ID &&
			a.DateIssued.Equal(b.DateIssued)
	})
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
		var text string
		var color fyne.ThemeColorName
		if r.isExpired {
			text = "EXPIRED"
			color = theme.ColorNameError
		} else {
			text = ihumanize.RelTime(r.dateExpired)
			color = theme.ColorNameForeground
		}
		r.dateExpiredDisplay = iwidget.RichTextSegmentsFromText(text, widget.RichTextStyle{
			ColorName: color,
		})
		tags, err := s.cs.ListTagsForCharacter(ctx, c.CharacterID)
		if err != nil {
			return nil, 0, err
		}
		r.tags = set.Collect(xiter.MapSlice(tags, func(x *app.CharacterTag) string {
			return x.Name
		}))
		rows = append(rows, r)
		if c.IsActive() {
			activeCount++
		}
	}
	return rows, activeCount, nil
}

func showContract(u *baseUI, characterID, contractID int32) {
	title := fmt.Sprintf("Contract #%d", contractID)
	w, ok := u.getOrCreateWindow(fmt.Sprintf("%d-%d", characterID, contractID), title, u.scs.CharacterName(characterID))
	if !ok {
		w.Show()
		return
	}
	o, err := u.cs.GetContract(context.Background(), characterID, contractID)
	if err != nil {
		u.showErrorDialog("Failed to show contract", err, u.window)
		return
	}
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

	var availability fyne.CanvasObject
	availabilityLabel := widget.NewLabel(o.AvailabilityDisplay())
	if o.Assignee != nil {
		availability = container.NewBorder(
			nil,
			nil,
			availabilityLabel,
			nil,
			makeEveEntityActionLabel(o.Assignee, u.ShowEveEntityInfoWindow),
		)
	} else {
		availability = availabilityLabel
	}
	fi := []*widget.FormItem{
		widget.NewFormItem("Owner", makeOwnerActionLabel(
			characterID,
			u.scs.CharacterName(characterID),
			u.ShowEveEntityInfoWindow,
		)),
		widget.NewFormItem("Info by issuer", widget.NewLabel(o.TitleDisplay())),
		widget.NewFormItem("Type", widget.NewLabel(o.TypeDisplay())),
		widget.NewFormItem("Issued By", makeEveEntityActionLabel(o.IssuerEffective(), u.ShowEveEntityInfoWindow)),
		widget.NewFormItem("Availability", availability),
	}
	if u.IsDeveloperMode() {
		fi = append(fi, widget.NewFormItem("Contract ID", u.makeCopyToClipboardLabel(fmt.Sprint(o.ContractID))))
	}
	if o.Type == app.ContractTypeCourier {
		fi = append(fi, widget.NewFormItem("Contractor", widget.NewLabel(o.AcceptorDisplay())))
	}
	fi = append(fi, widget.NewFormItem("Status", iwidget.NewRichText(o.StatusDisplayRichText()...)))
	fi = append(fi, widget.NewFormItem("Location", makeLocationLabel(o.StartLocation, u.ShowLocationInfoWindow)))

	if o.Type == app.ContractTypeCourier || o.Type == app.ContractTypeItemExchange {
		fi = append(fi, widget.NewFormItem("Date Issued", widget.NewLabel(o.DateIssued.Format(app.DateTimeFormat))))
		fi = append(fi, widget.NewFormItem("Date Accepted", widget.NewLabel(o.DateAccepted.StringFunc("", func(v time.Time) string {
			return v.Format(app.DateTimeFormat)
		}))))
		fi = append(fi, widget.NewFormItem("Date Expired", widget.NewLabel(makeExpiresString(o))))
		fi = append(fi, widget.NewFormItem("Date Completed", widget.NewLabel(o.DateCompleted.StringFunc("", func(v time.Time) string {
			return v.Format(app.DateTimeFormat)
		}))))
	}

	switch o.Type {
	case app.ContractTypeCourier:
		var collateral string
		if o.Collateral == 0 {
			collateral = "(None)"
		} else {
			collateral = formatISKAmount(o.Collateral)
		}
		fi = slices.Concat(fi, []*widget.FormItem{
			{Text: "Complete In", Widget: widget.NewLabel(fmt.Sprintf("%d days", o.DaysToComplete))},
			{Text: "Volume", Widget: widget.NewLabel(fmt.Sprintf("%f m3", o.Volume))},
			{Text: "Reward", Widget: widget.NewLabel(formatISKAmount(o.Reward))},
			{Text: "Collateral", Widget: widget.NewLabel(collateral)},
			{Text: "Destination", Widget: makeLocationLabel(o.EndLocation, u.ShowLocationInfoWindow)},
		})
	case app.ContractTypeItemExchange:
		if o.Price > 0 {
			x := widget.NewLabel(formatISKAmount(o.Price))
			x.Importance = widget.DangerImportance
			fi = append(fi, widget.NewFormItem("Buyer Will Pay", x))
		} else {
			x := widget.NewLabel(formatISKAmount(o.Reward))
			x.Importance = widget.SuccessImportance
			fi = append(fi, widget.NewFormItem("Buyer Will Get", x))
		}
	case app.ContractTypeAuction:
		ctx := context.TODO()
		total, err := u.cs.CountContractBids(ctx, o.ID)
		if err != nil {
			u.showErrorDialog("Failed to show contract bids", err, u.window)
			return
		}
		var currentBid string
		if total == 0 {
			currentBid = "(None)"
		} else {
			top, err := u.cs.GetContractTopBid(ctx, o.ID)
			if err != nil {
				u.showErrorDialog("Failed to show contract top bid", err, u.window)
				return
			}
			currentBid = fmt.Sprintf("%s (%d bids so far)", formatISKAmount(float64(top.Amount)), total)
		}
		fi = slices.Concat(fi, []*widget.FormItem{
			{Text: "Starting Bid", Widget: widget.NewLabel(formatISKAmount(o.Price))},
			{Text: "Buyout Price", Widget: widget.NewLabel(formatISKAmount(o.Buyout))},
			{Text: "Current Bid", Widget: widget.NewLabel(currentBid)},
			{Text: "Expires", Widget: widget.NewLabel(makeExpiresString(o))},
		})
	}

	makeItemsInfo := func(c *app.CharacterContract) (fyne.CanvasObject, error) {
		vb := container.NewVBox()
		items, err := u.cs.ListContractItems(context.TODO(), c.ID)
		if err != nil {
			return nil, err

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
			c := container.NewHBox(
				makeLinkLabel(it.Type.Name, func() {
					u.ShowTypeInfoWindow(it.Type.ID)
				}),
				widget.NewLabel(fmt.Sprintf("(%s)", it.Type.Group.Name)),
				widget.NewLabel(fmt.Sprintf("x %s ", humanize.Comma(int64(it.Quantity)))),
			)
			return c
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
		return vb, nil
	}

	subTitle := fmt.Sprintf("%s (%s)", o.NameDisplay(), o.TypeDisplay())
	f := widget.NewForm(fi...)
	f.Orientation = widget.Adaptive
	main := container.NewVBox(f)
	if o.Type == app.ContractTypeItemExchange || o.Type == app.ContractTypeAuction {
		main.Add(widget.NewSeparator())
		x, err := makeItemsInfo(o)
		if err != nil {
			u.showErrorDialog("Failed to show contract items", err, u.window)
			return
		}
		main.Add(x)
	}
	setDetailWindow(subTitle, main, w)
	w.Show()
}
