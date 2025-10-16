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
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
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
	corporationID      int32
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
	bottom         *widget.Label
	columnSorter   *iwidget.ColumnSorter
	corporation    *app.Corporation
	forCorporation bool // run in corporation mode when true, else in overview mode
	rows           []contractRow
	rowsFiltered   []contractRow
	selectAssignee *kxwidget.FilterChipSelect
	selectIssuer   *kxwidget.FilterChipSelect
	selectStatus   *kxwidget.FilterChipSelect
	selectTag      *kxwidget.FilterChipSelect
	selectType     *kxwidget.FilterChipSelect
	sortButton     *iwidget.SortButton
	u              *baseUI
}

const (
	contractsColName      = 0
	contractsColType      = 1
	contractsColIssuer    = 2
	contractsColAssignee  = 3
	contractsColStatus    = 4
	contractsColIssuedAt  = 5
	contractsColExpiresAt = 6
)

func newContractsForCorporation(u *baseUI) *contracts {
	return newContracts(u, true)
}

func newContractsForOverview(u *baseUI) *contracts {
	return newContracts(u, false)
}

func newContracts(u *baseUI, forCorporation bool) *contracts {
	headers := iwidget.NewDataTableDef([]iwidget.ColumnDef{{
		Col:   contractsColName,
		Label: "Contract",
		Width: 300,
	}, {
		Col:   contractsColType,
		Label: "Type",
		Width: 120,
	}, {
		Col:   contractsColIssuer,
		Label: "From",
		Width: 150,
	}, {
		Col:   contractsColAssignee,
		Label: "To",
		Width: 150,
	}, {
		Col:   contractsColStatus,
		Label: "Status",
		Width: 100,
	}, {
		Col:   contractsColIssuedAt,
		Label: "Date Issued",
		Width: columnWidthDateTime,
	}, {
		Col:   contractsColExpiresAt,
		Label: "Time Left",
		Width: 100,
	}})
	a := &contracts{
		forCorporation: forCorporation,
		columnSorter:   headers.NewColumnSorter(contractsColIssuedAt, iwidget.SortDesc),
		rows:           make([]contractRow, 0),
		bottom:         widget.NewLabel(""),
		u:              u,
	}
	a.ExtendBaseWidget(a)
	if a.u.isDesktop {
		a.body = iwidget.MakeDataTable(headers, &a.rowsFiltered,
			func(col int, r contractRow) []widget.RichTextSegment {
				switch col {
				case contractsColName:
					return iwidget.RichTextSegmentsFromText(r.name)
				case contractsColType:
					return iwidget.RichTextSegmentsFromText(r.typeName)
				case contractsColIssuer:
					return iwidget.RichTextSegmentsFromText(r.issuerName)
				case contractsColAssignee:
					return iwidget.RichTextSegmentsFromText(r.assigneeName)
				case contractsColStatus:
					return r.status.DisplayRichText()
				case contractsColIssuedAt:
					return iwidget.RichTextSegmentsFromText(r.dateIssued.Format(app.DateTimeFormat))
				case contractsColExpiresAt:
					return r.dateExpiredDisplay
				}
				return iwidget.RichTextSegmentsFromText("?")
			}, a.columnSorter, a.filterRows, func(column int, r contractRow) {
				if a.forCorporation {
					showCorporationContractWindow(a.u, r.corporationID, r.contractID)
				} else {
					showCharacterContractWindow(a.u, r.characterID, r.contractID)
				}
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
	a.sortButton = a.columnSorter.NewSortButton(func() {
		a.filterRows(-1)
	}, a.u.window)

	if a.forCorporation {
		a.u.currentCorporationExchanged.AddListener(
			func(_ context.Context, c *app.Corporation) {
				a.corporation = c
			},
		)
		a.u.corporationSectionChanged.AddListener(func(_ context.Context, arg corporationSectionUpdated) {
			if corporationIDOrZero(a.corporation) != arg.corporationID {
				return
			}
			if arg.section != app.SectionCorporationContracts {
				return
			}
			a.update()
		})
	} else {
		a.u.characterSectionChanged.AddListener(func(_ context.Context, arg characterSectionUpdated) {
			if arg.section == app.SectionCharacterContracts {
				a.update()
			}
		})
	}
	return a
}

func (a *contracts) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHBox(
		a.selectType,
		a.selectIssuer,
		a.selectAssignee,
		a.selectStatus,
	)
	if !a.forCorporation {
		filter.Add(a.selectTag)
	}
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
		defer l.UnselectAll()
		if id < 0 || id >= len(a.rowsFiltered) {
			return
		}
		r := a.rowsFiltered[id]
		if a.forCorporation {
			showCorporationContractWindow(a.u, r.corporationID, r.contractID)
		} else {
			showCharacterContractWindow(a.u, r.characterID, r.contractID)
		}
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
	a.columnSorter.Sort(sortCol, func(sortCol int, dir iwidget.SortDir) {
		slices.SortFunc(rows, func(a, b contractRow) int {
			var x int
			switch sortCol {
			case contractsColName:
				x = strings.Compare(a.name, b.name)
			case contractsColType:
				x = strings.Compare(a.typeName, b.typeName)
			case contractsColIssuer:
				x = xstrings.CompareIgnoreCase(a.issuerName, b.issuerName)
			case contractsColAssignee:
				x = xstrings.CompareIgnoreCase(a.assigneeName, b.assigneeName)
			case contractsColStatus:
				x = strings.Compare(a.statusText, b.statusText)
			case contractsColIssuedAt:
				x = a.dateIssued.Compare(b.dateIssued)
			case contractsColExpiresAt:
				x = a.dateExpired.Compare(b.dateExpired)
			}
			if dir == iwidget.SortAsc {
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
	var activeCount int
	var err error
	var rows []contractRow
	if a.forCorporation {
		rows, activeCount, err = a.fetchRowsCorporation()
	} else {
		rows, activeCount, err = a.fetchRowsOverview()
	}
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

func (a *contracts) fetchRowsCorporation() ([]contractRow, int, error) {
	corporationID := corporationIDOrZero(a.corporation)
	if corporationID == 0 {
		return []contractRow{}, 0, nil
	}
	ctx := context.Background()
	oo, err := a.u.rs.ListCorporationContracts(ctx, corporationID)
	if err != nil {
		return nil, 0, err
	}
	rows := make([]contractRow, 0)
	var activeCount int
	for _, c := range oo {
		r := contractRow{
			name:          c.NameDisplay(),
			typeName:      c.Type.Display(),
			issuerName:    c.IssuerEffective().Name,
			assigneeName:  entityNameOrFallback(c.Assignee, ""),
			statusText:    c.Status.Display(),
			status:        c.Status,
			dateIssued:    c.DateIssued,
			dateExpired:   c.DateExpired,
			isExpired:     c.IsExpired(),
			corporationID: c.CorporationID,
			contractID:    c.ContractID,
			isActive:      c.Status.IsActive(),
			isHistory:     c.Status.IsCompleted(),
			hasIssue:      c.HasIssue(),
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
		rows = append(rows, r)
		if c.Status.IsActive() {
			activeCount++
		}
	}
	return rows, activeCount, nil
}

func (a *contracts) fetchRowsOverview() ([]contractRow, int, error) {
	ctx := context.Background()
	oo, err := a.u.cs.ListAllContracts(ctx)
	if err != nil {
		return nil, 0, err
	}
	// Remove duplicate oo2 between the user's own characters
	oo2 := slices.CompactFunc(oo, func(a, b *app.CharacterContract) bool {
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
	for _, c := range oo2 {
		r := contractRow{
			name:         c.NameDisplay(),
			typeName:     c.Type.Display(),
			issuerName:   c.IssuerEffective().Name,
			assigneeName: entityNameOrFallback(c.Assignee, ""),
			statusText:   c.Status.Display(),
			status:       c.Status,
			dateIssued:   c.DateIssued,
			dateExpired:  c.DateExpired,
			isExpired:    c.IsExpired(),
			characterID:  c.CharacterID,
			contractID:   c.ContractID,
			isActive:     c.Status.IsActive(),
			isHistory:    c.Status.IsCompleted(),
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
		tags, err := a.u.cs.ListTagsForCharacter(ctx, c.CharacterID)
		if err != nil {
			return nil, 0, err
		}
		r.tags = tags
		rows = append(rows, r)
		if c.Status.IsActive() {
			activeCount++
		}
	}
	return rows, activeCount, nil
}

// showCharacterContractWindow shows the details of a character contract in a window.
func showCharacterContractWindow(u *baseUI, characterID, contractID int32) {
	ctx := context.Background()
	o, err := u.cs.GetContract(ctx, characterID, contractID)
	if err != nil {
		u.showErrorDialog("Failed to show contract", err, u.window)
		return
	}
	title := fmt.Sprintf("Contract #%d", contractID)
	characterName := u.scs.CharacterName(characterID)
	w, created := u.getOrCreateWindow(
		fmt.Sprintf("character-contract-%d-%d", characterID, contractID),
		title,
		characterName,
	)
	if !created {
		w.Show()
		return
	}

	var availability fyne.CanvasObject
	availabilityLabel := widget.NewLabel(o.Availability.Display())
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
		widget.NewFormItem("Owner", makeCharacterActionLabel(
			characterID,
			characterName,
			u.ShowEveEntityInfoWindow,
		)),
		widget.NewFormItem("Info by issuer", widget.NewLabel(o.Title)),
		widget.NewFormItem("Type", widget.NewLabel(o.Type.Display())),
		widget.NewFormItem("Issued By", makeEveEntityActionLabel(o.IssuerEffective(), u.ShowEveEntityInfoWindow)),
		widget.NewFormItem("Availability", availability),
	}
	if u.IsDeveloperMode() {
		fi = append(fi, widget.NewFormItem("Contract ID", u.makeCopyToClipboardLabel(fmt.Sprint(o.ContractID))))
	}
	if o.Type == app.ContractTypeCourier {
		fi = append(fi, widget.NewFormItem("Contractor", widget.NewLabel(entityNameOrFallback(o.Acceptor, "(none)"))))
	}
	fi = append(fi, widget.NewFormItem("Status", iwidget.NewRichText(o.Status.DisplayRichText()...)))
	fi = append(fi, widget.NewFormItem("Location", makeLocationLabel(o.StartLocation, u.ShowLocationInfoWindow)))

	if o.Type == app.ContractTypeCourier || o.Type == app.ContractTypeItemExchange {
		fi = append(fi, widget.NewFormItem("Date Issued", widget.NewLabel(o.DateIssued.Format(app.DateTimeFormat))))
		fi = append(fi, widget.NewFormItem("Date Accepted", widget.NewLabel(o.DateAccepted.StringFunc("", func(v time.Time) string {
			return v.Format(app.DateTimeFormat)
		}))))
		fi = append(fi, widget.NewFormItem("Date Expired", widget.NewLabel(makeContractExpiresString(o.DateExpired, o.IsExpired()))))
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
			{Text: "Expires", Widget: widget.NewLabel(makeContractExpiresString(o.DateExpired, o.IsExpired()))},
		})
	}

	makeItemsInfo := func(c *app.CharacterContract) (fyne.CanvasObject, error) {
		vb := container.NewVBox()
		items, err := u.cs.ListContractItems(ctx, c.ID)
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
					u.ShowTypeInfoWindowWithCharacter(it.Type.ID, characterID)
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

	subTitle := fmt.Sprintf("%s (%s)", o.NameDisplay(), o.Type.Display())
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
	setDetailWindow(detailWindowParams{
		title:   subTitle,
		content: main,
		window:  w,
	})
	w.Show()
}

// showCorporationContractWindow shows the details of a corporation contract in a window.
func showCorporationContractWindow(u *baseUI, corporationID, contractID int32) {
	ctx := context.Background()
	o, err := u.rs.GetContract(ctx, corporationID, contractID)
	if err != nil {
		u.showErrorDialog("Failed to show contract", err, u.window)
		return
	}
	title := fmt.Sprintf("Contract #%d", contractID)
	corporationName := u.scs.CorporationName(corporationID)
	w, created := u.getOrCreateWindow(
		fmt.Sprintf("corporation-contract-%d-%d", corporationID, contractID),
		title,
		corporationName,
	)
	if !created {
		w.Show()
		return
	}

	var availability fyne.CanvasObject
	availabilityLabel := widget.NewLabel(o.Availability.Display())
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
		widget.NewFormItem("Owner", makeCharacterActionLabel(
			corporationID,
			corporationName,
			u.ShowEveEntityInfoWindow,
		)),
		widget.NewFormItem("Info by issuer", widget.NewLabel(o.Title)),
		widget.NewFormItem("Type", widget.NewLabel(o.Type.Display())),
		widget.NewFormItem("Issued By", makeEveEntityActionLabel(o.IssuerEffective(), u.ShowEveEntityInfoWindow)),
		widget.NewFormItem("Availability", availability),
	}
	if u.IsDeveloperMode() {
		fi = append(fi, widget.NewFormItem("Contract ID", u.makeCopyToClipboardLabel(fmt.Sprint(o.ContractID))))
	}
	if o.Type == app.ContractTypeCourier {
		fi = append(fi, widget.NewFormItem("Contractor", widget.NewLabel(entityNameOrFallback(o.Acceptor, "(none)"))))
	}
	fi = append(fi, widget.NewFormItem("Status", iwidget.NewRichText(o.Status.DisplayRichText()...)))
	fi = append(fi, widget.NewFormItem("Location", makeLocationLabel(o.StartLocation, u.ShowLocationInfoWindow)))

	if o.Type == app.ContractTypeCourier || o.Type == app.ContractTypeItemExchange {
		fi = append(fi, widget.NewFormItem("Date Issued", widget.NewLabel(o.DateIssued.Format(app.DateTimeFormat))))
		fi = append(fi, widget.NewFormItem("Date Accepted", widget.NewLabel(o.DateAccepted.StringFunc("", func(v time.Time) string {
			return v.Format(app.DateTimeFormat)
		}))))
		fi = append(fi, widget.NewFormItem("Date Expired", widget.NewLabel(makeContractExpiresString(o.DateExpired, o.IsExpired()))))
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
			{Text: "Expires", Widget: widget.NewLabel(makeContractExpiresString(o.DateExpired, o.IsExpired()))},
		})
	}

	makeItemsInfo := func(c *app.CorporationContract) (fyne.CanvasObject, error) {
		vb := container.NewVBox()
		items, err := u.rs.ListContractItems(ctx, c.ID)
		if err != nil {
			return nil, err

		}
		var itemsIncluded, itemsRequested []*app.CorporationContractItem
		for _, it := range items {
			if it.IsIncluded {
				itemsIncluded = append(itemsIncluded, it)
			} else {
				itemsRequested = append(itemsRequested, it)
			}
		}
		makeItem := func(it *app.CorporationContractItem) fyne.CanvasObject {
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

	subTitle := fmt.Sprintf("%s (%s)", o.NameDisplay(), o.Type.Display())
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
	setDetailWindow(detailWindowParams{
		title:   subTitle,
		content: main,
		window:  w,
	})
	w.Show()
}

func makeContractExpiresString(dateExpired time.Time, isExpired bool) string {
	ts := dateExpired.Format(app.DateTimeFormat)
	var ds string
	if isExpired {
		ds = "EXPIRED"
	} else {
		ds = ihumanize.RelTime(dateExpired)
	}
	return fmt.Sprintf("%s (%s)", ts, ds)
}
