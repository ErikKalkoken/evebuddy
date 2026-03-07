package ui

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync/atomic"
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
	"github.com/ErikKalkoken/evebuddy/internal/app/uiservices"
	"github.com/ErikKalkoken/evebuddy/internal/app/xdialog"
	"github.com/ErikKalkoken/evebuddy/internal/app/xwindow"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
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
	characterID        int64
	corporationID      int64
	contractID         int64
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

// contracts is a UI element for showing contracts.
// It either shows all character contracts or the contracts for a corporation.
type contracts struct {
	widget.BaseWidget

	OnUpdate func(active int)

	body           fyne.CanvasObject
	footer         *widget.Label
	columnSorter   *xwidget.ColumnSorter[contractRow]
	corporation    atomic.Pointer[app.Corporation]
	forCorporation bool // reports whether it runs in corporation mode
	rows           []contractRow
	rowsFiltered   []contractRow
	selectAssignee *kxwidget.FilterChipSelect
	selectIssuer   *kxwidget.FilterChipSelect
	selectStatus   *kxwidget.FilterChipSelect
	selectTag      *kxwidget.FilterChipSelect
	selectType     *kxwidget.FilterChipSelect
	sortButton     *xwidget.SortButton
	u         uiservices.UIServices
}

const (
	contractsColName = iota + 1
	contractsColType
	contractsColIssuer
	contractsColAssignee
	contractsColStatus
	contractsColIssuedAt
	contractsColExpiresAt
)

func newContractsForCorporation(u         uiservices.UIServices) *contracts {
	return newContracts(u, true)
}

func newContractsForCharacters(u         uiservices.UIServices) *contracts {
	return newContracts(u, false)
}

func newContracts(u         uiservices.UIServices, forCorporation bool) *contracts {
	columns := xwidget.NewDataColumns([]xwidget.DataColumn[contractRow]{{
		ID:    contractsColName,
		Label: "Contract",
		Width: 300,
		Sort: func(a, b contractRow) int {
			return strings.Compare(a.name, b.name)
		},
		Update: func(r contractRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.name)
		},
	}, {
		ID:    contractsColType,
		Label: "Type",
		Width: 120,
		Sort: func(a, b contractRow) int {
			return strings.Compare(a.typeName, b.typeName)
		},
		Update: func(r contractRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.typeName)
		},
	}, {
		ID:    contractsColIssuer,
		Label: "From",
		Width: 150,
		Sort: func(a, b contractRow) int {
			return xstrings.CompareIgnoreCase(a.issuerName, b.issuerName)
		},
		Update: func(r contractRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.issuerName)
		},
	}, {
		ID:    contractsColAssignee,
		Label: "To",
		Width: 150,
		Sort: func(a, b contractRow) int {
			return xstrings.CompareIgnoreCase(a.assigneeName, b.assigneeName)
		},
		Update: func(r contractRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.assigneeName)
		},
	}, {
		ID:    contractsColStatus,
		Label: "Status",
		Width: 100,
		Sort: func(a, b contractRow) int {
			return strings.Compare(a.statusText, b.statusText)
		},
		Update: func(r contractRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).Set(r.status.DisplayRichText())
		},
	}, {
		ID:    contractsColIssuedAt,
		Label: "Date Issued",
		Width: columnWidthDateTime,
		Sort: func(a, b contractRow) int {
			return a.dateIssued.Compare(b.dateIssued)
		},
		Update: func(r contractRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.dateIssued.Format(app.DateTimeFormat))
		},
	}, {
		ID:    contractsColExpiresAt,
		Label: "Time Left",
		Width: 100,
		Sort: func(a, b contractRow) int {
			return a.dateExpired.Compare(b.dateExpired)
		},
		Update: func(r contractRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).Set(r.dateExpiredDisplay)
		},
	}})
	a := &contracts{
		forCorporation: forCorporation,
		columnSorter:   xwidget.NewColumnSorter(columns, contractsColIssuedAt, xwidget.SortDesc),
		footer:         newLabelWithTruncation(),
		u:              u,
	}
	a.ExtendBaseWidget(a)

	if app.IsMobile() {
		a.body = a.makeDataList()
	} else {
		a.body = xwidget.MakeDataTable(
			columns,
			&a.rowsFiltered,
			func() fyne.CanvasObject {
				x := xwidget.NewRichText()
				x.Truncation = fyne.TextTruncateClip
				return x
			},
			a.columnSorter,
			a.filterRowsAsync,
			func(column int, r contractRow) {
				if a.forCorporation {
					showCorporationContractWindow(a.u, r.corporationID, r.contractID)
				} else {
					showCharacterContractWindow(a.u, r.characterID, r.contractID)
				}
			},
		)
	}

	a.selectAssignee = kxwidget.NewFilterChipSelectWithSearch("Assignee", []string{}, func(string) {
		a.filterRowsAsync(-1)
	}, a.u.MainWindow())
	a.selectIssuer = kxwidget.NewFilterChipSelectWithSearch("Issuer", []string{}, func(string) {
		a.filterRowsAsync(-1)
	}, a.u.MainWindow())
	a.selectType = kxwidget.NewFilterChipSelect("Type", []string{}, func(string) {
		a.filterRowsAsync(-1)
	})

	a.selectStatus = kxwidget.NewFilterChipSelect("", []string{
		contractStatusAllActive,
		contractStatusOutstanding,
		contractStatusInProgress,
		contractStatusHasIssue,
		contractStatusHistory,
	}, func(string) {
		a.filterRowsAsync(-1)
	})
	a.selectStatus.Selected = contractStatusAllActive
	a.selectStatus.SortDisabled = true
	a.selectTag = kxwidget.NewFilterChipSelect("Tag", []string{}, func(string) {
		a.filterRowsAsync(-1)
	})
	a.sortButton = a.columnSorter.NewSortButton(func() {
		a.filterRowsAsync(-1)
	}, a.u.MainWindow())

	// Signals
	if a.forCorporation {
		a.u.Signals().CurrentCorporationExchanged.AddListener(func(ctx context.Context, c *app.Corporation) {
			a.corporation.Store(c)
			a.update(ctx)
		})
		a.u.Signals().CorporationSectionChanged.AddListener(func(ctx context.Context, arg app.CorporationSectionUpdated) {
			if corporationIDOrZero(a.corporation.Load()) != arg.CorporationID {
				return
			}
			if arg.Section != app.SectionCorporationContracts {
				return
			}
			a.update(ctx)
		})
	} else {
		a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
			if arg.Section == app.SectionCharacterContracts {
				a.update(ctx)
			}
		})
		a.u.Signals().CharacterAdded.AddListener(func(ctx context.Context, _ *app.Character) {
			a.update(ctx)
		})
		a.u.Signals().CharacterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
			a.update(ctx)
		})
		a.u.Signals().TagsChanged.AddListener(func(ctx context.Context, s struct{}) {
			a.update(ctx)
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
	if app.IsMobile() {
		filter.Add(a.sortButton)
	}
	c := container.NewBorder(
		container.NewVBox(container.NewHScroll(filter)),
		a.footer,
		nil,
		nil,
		a.body,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *contracts) makeDataList() *xwidget.StripedList {
	p := theme.Padding()
	l := xwidget.NewStripedList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			title := widget.NewLabelWithStyle("Template", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			type_ := widget.NewLabel("Template")
			status := xwidget.NewRichTextWithText("Template")
			issuer := widget.NewLabel("Template")
			assignee := widget.NewLabel("Template")
			dateExpired := xwidget.NewRichTextWithText("Template")
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
			box[2].(*xwidget.RichText).Set(r.status.DisplayRichText())

			main[2].(*widget.Label).SetText("From " + r.issuerName)
			assignee := "To "
			if r.assigneeName == "" {
				assignee += "..."
			} else {
				assignee += r.assigneeName
			}
			main[3].(*widget.Label).SetText(assignee)

			main[4].(*xwidget.RichText).Set(xwidget.InlineRichTextSegments(
				xwidget.RichTextSegmentsFromText("Expires "),
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

func (a *contracts) filterRowsAsync(sortCol int) {
	totalRows := len(a.rows)
	rows := slices.Clone(a.rows)
	issuer := a.selectIssuer.Selected
	assignee := a.selectAssignee.Selected
	type_ := a.selectType.Selected
	tag := a.selectTag.Selected
	sortCol, dir, doSort := a.columnSorter.CalcSort(sortCol)

	go func() {
		// filter
		rows = slices.DeleteFunc(rows, func(r contractRow) bool {
			switch a.selectStatus.Selected {
			case contractStatusAllActive:
				return !r.isActive
			case contractStatusOutstanding:
				return r.status != app.ContractStatusOutstanding
			case contractStatusInProgress:
				return r.status != app.ContractStatusInProgress
			case contractStatusHasIssue:
				return !r.hasIssue
			case contractStatusHistory:
				return !r.isHistory
			}
			return true
		})
		if issuer != "" {
			rows = slices.DeleteFunc(rows, func(r contractRow) bool {
				return r.issuerName != issuer
			})
		}
		if assignee != "" {
			rows = slices.DeleteFunc(rows, func(r contractRow) bool {
				return r.assigneeName != assignee
			})
		}
		if type_ != "" {
			rows = slices.DeleteFunc(rows, func(r contractRow) bool {
				return r.typeName != type_
			})
		}
		if tag != "" {
			rows = slices.DeleteFunc(rows, func(r contractRow) bool {
				return !r.tags.Contains(tag)
			})
		}
		a.columnSorter.SortRows(rows, sortCol, dir, doSort)
		// set data & refresh
		tagOptions := slices.Sorted(set.Union(xslices.Map(rows, func(r contractRow) set.Set[string] {
			return r.tags
		})...).All())
		issueOptions := xslices.Map(rows, func(r contractRow) string {
			return r.issuerName
		})
		assigneeOptions := xslices.Map(rows, func(r contractRow) string {
			return r.assigneeName
		})
		typeOptions := xslices.Map(rows, func(r contractRow) string {
			return r.typeName
		})

		footer := fmt.Sprintf("Showing %d / %d contracts", len(rows), totalRows)

		fyne.Do(func() {
			a.footer.Text = footer
			a.footer.Importance = widget.MediumImportance
			a.footer.Refresh()
			a.selectTag.SetOptions(tagOptions)
			a.selectIssuer.SetOptions(issueOptions)
			a.selectAssignee.SetOptions(assigneeOptions)
			a.selectType.SetOptions(typeOptions)
			a.rowsFiltered = rows
			a.body.Refresh()
		})
	}()
}

func (a *contracts) update(ctx context.Context) {
	var activeCount int
	var err error
	var rows []contractRow
	if a.forCorporation {
		rows, activeCount, err = a.fetchRowsCorporation(ctx)
	} else {
		rows, activeCount, err = a.fetchRowsOverview(ctx)
	}
	if err != nil {
		slog.Error("Failed to refresh contracts UI", "err", err)
		fyne.Do(func() {
			a.footer.Text = fmt.Sprintf("ERROR: %s", app.ErrorDisplay(err))
			a.footer.Importance = widget.DangerImportance
			a.footer.Refresh()
		})
		return
	}
	fyne.Do(func() {
		a.rows = rows
		a.filterRowsAsync(-1)
		if a.OnUpdate != nil {
			a.OnUpdate(activeCount)
		}
	})
}

func (a *contracts) fetchRowsCorporation(ctx context.Context) ([]contractRow, int, error) {
	corporationID := corporationIDOrZero(a.corporation.Load())
	if corporationID == 0 {
		return nil, 0, nil
	}
	oo, err := a.u.Corporation().ListCorporationContracts(ctx, corporationID)
	if err != nil {
		return nil, 0, err
	}
	var rows []contractRow
	var activeCount int
	for _, c := range oo {
		r := contractRow{
			name:       c.NameDisplay(),
			typeName:   c.Type.Display(),
			issuerName: c.IssuerEffective().Name,
			assigneeName: c.Assignee.StringFunc("", func(v *app.EveEntity) string {
				return v.Name
			}),
			statusText:    c.Status.Display(),
			status:        c.Status,
			dateIssued:    c.DateIssued,
			dateExpired:   c.DateExpired,
			isExpired:     c.IsExpired(),
			corporationID: c.CorporationID,
			contractID:    c.ContractID,
			isActive:      c.Status.IsActive(),
			isHistory:     c.Status.IsHistory(),
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
		r.dateExpiredDisplay = xwidget.RichTextSegmentsFromText(text, widget.RichTextStyle{
			ColorName: color,
		})
		rows = append(rows, r)
		if c.Status.IsActive() {
			activeCount++
		}
	}
	return rows, activeCount, nil
}

func (a *contracts) fetchRowsOverview(ctx context.Context) ([]contractRow, int, error) {
	oo, err := a.u.Character().ListAllContracts(ctx)
	if err != nil {
		return nil, 0, err
	}
	// Remove duplicate oo2 between the user's own characters
	oo2 := slices.CompactFunc(oo, func(a, b *app.CharacterContract) bool {
		return a.ContractID == b.ContractID &&
			a.Type == b.Type &&
			a.Issuer.ID == b.Issuer.ID &&
			optional.EqualFunc(a.Assignee, b.Assignee, func(x, y *app.EveEntity) bool { return x.ID == y.ID }) &&
			a.DateIssued.Equal(b.DateIssued)
	})
	var rows []contractRow
	var activeCount int
	for _, c := range oo2 {
		r := contractRow{
			name:       c.NameDisplay(),
			typeName:   c.Type.Display(),
			issuerName: c.IssuerEffective().Name,
			assigneeName: c.Assignee.StringFunc("", func(v *app.EveEntity) string {
				return v.Name
			}),
			statusText:  c.Status.Display(),
			status:      c.Status,
			dateIssued:  c.DateIssued,
			dateExpired: c.DateExpired,
			isExpired:   c.IsExpired(),
			characterID: c.CharacterID,
			contractID:  c.ContractID,
			isActive:    c.Status.IsActive(),
			isHistory:   c.Status.IsHistory(),
			hasIssue:    c.HasIssue(),
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
		r.dateExpiredDisplay = xwidget.RichTextSegmentsFromText(text, widget.RichTextStyle{
			ColorName: color,
		})
		tags, err := a.u.Character().ListTagsForCharacter(ctx, c.CharacterID)
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

// FIXME: Remove DB access from main go routine

// showCharacterContractWindow shows the details of a character contract in a window.
func showCharacterContractWindow(u         uiservices.UIServices, characterID, contractID int64) {
	ctx := context.Background()
	o, err := u.Character().GetContract(ctx, characterID, contractID)
	if err != nil {
		xdialog.ShowErrorAndLog("Failed to show contract", err, u.MainWindow())
		return
	}
	title := fmt.Sprintf("Contract #%d", contractID)
	characterName := u.StatusCache().CharacterName(characterID)
	w, created := u.GetOrCreateWindow(
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
	if v, ok := o.Assignee.Value(); ok {
		availability = container.NewBorder(
			nil,
			nil,
			availabilityLabel,
			nil,
			makeEveEntityActionLabel(v, u.InfoWindow().ShowEntity),
		)
	} else {
		availability = availabilityLabel
	}
	fi := []*widget.FormItem{
		widget.NewFormItem("Owner", makeCharacterActionLabel(
			characterID,
			characterName,
			u.InfoWindow().ShowEntity,
		)),
		widget.NewFormItem("Info by issuer", widget.NewLabel(o.Title.ValueOrFallback("-"))),
		widget.NewFormItem("Type", widget.NewLabel(o.Type.Display())),
		widget.NewFormItem("Issued By", makeEveEntityActionLabel(o.IssuerEffective(), u.InfoWindow().ShowEntity)),
		widget.NewFormItem("Availability", availability),
	}
	if app.IsDeveloperMode() {
		fi = append(fi, widget.NewFormItem("Contract ID", xwidget.NewTappableLabelWithClipboardCopy(fmt.Sprint(o.ContractID))))
	}
	if o.Type == app.ContractTypeCourier {
		fi = append(fi, widget.NewFormItem("Contractor", makeEveEntityActionLabel2(o.Acceptor, u.InfoWindow().ShowEntity)))
	}
	fi = append(fi, widget.NewFormItem("Status", xwidget.NewRichText(o.Status.DisplayRichText()...)))
	fi = append(fi, widget.NewFormItem("Location", makeLocationLabel2(o.StartLocation, u.InfoWindow().ShowLocation)))

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
		fi = slices.Concat(fi, []*widget.FormItem{
			{Text: "Complete In", Widget: widget.NewLabel(fmt.Sprintf("%s days", o.DaysToComplete.StringFunc("?", func(v int64) string {
				return fmt.Sprint(v)
			})))},
			{Text: "Volume", Widget: widget.NewLabel(fmt.Sprintf("%s m3", o.Volume.StringFunc("?", func(v float64) string {
				return fmt.Sprint(v)
			})))},
			{Text: "Reward", Widget: widget.NewLabel(o.Reward.StringFunc("-", formatISKAmount))},
			{Text: "Collateral", Widget: widget.NewLabel(o.Collateral.StringFunc("-", formatISKAmount))},
			{Text: "Destination", Widget: makeLocationLabel2(o.EndLocation, u.InfoWindow().ShowLocation)},
		})
	case app.ContractTypeItemExchange:
		if o.Price.ValueOrZero() > 0 {
			x := widget.NewLabel(o.Price.StringFunc("?", formatISKAmount))
			x.Importance = widget.DangerImportance
			fi = append(fi, widget.NewFormItem("Buyer Will Pay", x))
		} else {
			x := widget.NewLabel(o.Reward.StringFunc("?", formatISKAmount))
			x.Importance = widget.SuccessImportance
			fi = append(fi, widget.NewFormItem("Buyer Will Get", x))
		}
	case app.ContractTypeAuction:
		total, err := u.Character().CountContractBids(ctx, o.ID)
		if err != nil {
			xdialog.ShowErrorAndLog("Failed to show contract bids", err, u.MainWindow())
			return
		}
		var currentBid string
		if total == 0 {
			currentBid = "(None)"
		} else {
			top, err := u.Character().GetContractTopBid(ctx, o.ID)
			if err != nil {
				xdialog.ShowErrorAndLog("Failed to show contract top bid", err, u.MainWindow())
				return
			}
			currentBid = fmt.Sprintf("%s (%d bids so far)", formatISKAmount(float64(top.Amount)), total)
		}
		fi = slices.Concat(fi, []*widget.FormItem{
			{Text: "Starting Bid", Widget: widget.NewLabel(o.Price.StringFunc("?", formatISKAmount))},
			{Text: "Buyout Price", Widget: widget.NewLabel(o.Buyout.StringFunc("?", formatISKAmount))},
			{Text: "Current Bid", Widget: widget.NewLabel(currentBid)},
			{Text: "Expires", Widget: widget.NewLabel(makeContractExpiresString(o.DateExpired, o.IsExpired()))},
		})
	}

	makeItemsInfo := func(c *app.CharacterContract) (fyne.CanvasObject, error) {
		vb := container.NewVBox()
		items, err := u.Character().ListContractItems(ctx, c.ID)
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
					u.InfoWindow().ShowTypeWithCharacter(it.Type.ID, characterID)
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
			xdialog.ShowErrorAndLog("Failed to show contract items", err, u.MainWindow())
			return
		}
		main.Add(x)
	}
	xwindow.Set(xwindow.Params{
		Title:   subTitle,
		Content: main,
		Window:  w,
	})
	w.Show()
}

// FIXME: Remove DB access from main go routine

// showCorporationContractWindow shows the details of a corporation contract in a window.
func showCorporationContractWindow(u         uiservices.UIServices, corporationID, contractID int64) {
	ctx := context.Background()
	o, err := u.Corporation().GetContract(ctx, corporationID, contractID)
	if err != nil {
		xdialog.ShowErrorAndLog("Failed to show contract", err, u.MainWindow())
		return
	}
	title := fmt.Sprintf("Contract #%d", contractID)
	corporationName := u.StatusCache().CorporationName(corporationID)
	w, created := u.GetOrCreateWindow(
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
	if v, ok := o.Assignee.Value(); ok {
		availability = container.NewBorder(
			nil,
			nil,
			availabilityLabel,
			nil,
			makeEveEntityActionLabel(v, u.InfoWindow().ShowEntity),
		)
	} else {
		availability = availabilityLabel
	}
	fi := []*widget.FormItem{
		widget.NewFormItem("Owner", makeCharacterActionLabel(
			corporationID,
			corporationName,
			u.InfoWindow().ShowEntity,
		)),
		widget.NewFormItem("Info by issuer", widget.NewLabel(o.Title.ValueOrFallback("-"))),
		widget.NewFormItem("Type", widget.NewLabel(o.Type.Display())),
		widget.NewFormItem("Issued By", makeEveEntityActionLabel(o.IssuerEffective(), u.InfoWindow().ShowEntity)),
		widget.NewFormItem("Availability", availability),
	}
	if app.IsDeveloperMode() {
		fi = append(fi, widget.NewFormItem("Contract ID", xwidget.NewTappableLabelWithClipboardCopy(fmt.Sprint(o.ContractID))))
	}
	if o.Type == app.ContractTypeCourier {
		fi = append(fi, widget.NewFormItem("Contractor", makeEveEntityActionLabel2(o.Acceptor, u.InfoWindow().ShowEntity)))
	}
	fi = append(fi, widget.NewFormItem("Status", xwidget.NewRichText(o.Status.DisplayRichText()...)))
	fi = append(fi, widget.NewFormItem("Location", makeLocationLabel2(o.StartLocation, u.InfoWindow().ShowLocation)))

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
		fi = slices.Concat(fi, []*widget.FormItem{
			{Text: "Complete In", Widget: widget.NewLabel(fmt.Sprintf("%s days", o.DaysToComplete.StringFunc("?", func(v int64) string {
				return fmt.Sprint(v)
			})))},
			{Text: "Volume", Widget: widget.NewLabel(fmt.Sprintf("%s m3", o.Volume.StringFunc("?", func(v float64) string {
				return fmt.Sprint(v)
			})))},
			{Text: "Reward", Widget: widget.NewLabel(o.Reward.StringFunc("-", formatISKAmount))},
			{Text: "Collateral", Widget: widget.NewLabel(o.Collateral.StringFunc("-", formatISKAmount))},
			{Text: "Destination", Widget: makeLocationLabel2(o.EndLocation, u.InfoWindow().ShowLocation)},
		})
	case app.ContractTypeItemExchange:
		if o.Price.ValueOrZero() > 0 {
			x := widget.NewLabel(o.Price.StringFunc("?", formatISKAmount))
			x.Importance = widget.DangerImportance
			fi = append(fi, widget.NewFormItem("Buyer Will Pay", x))
		} else {
			x := widget.NewLabel(o.Reward.StringFunc("?", formatISKAmount))
			x.Importance = widget.SuccessImportance
			fi = append(fi, widget.NewFormItem("Buyer Will Get", x))
		}
	case app.ContractTypeAuction:
		total, err := u.Character().CountContractBids(ctx, o.ID)
		if err != nil {
			xdialog.ShowErrorAndLog("Failed to show contract bids", err, u.MainWindow())
			return
		}
		var currentBid string
		if total == 0 {
			currentBid = "(None)"
		} else {
			top, err := u.Character().GetContractTopBid(ctx, o.ID)
			if err != nil {
				xdialog.ShowErrorAndLog("Failed to show contract top bid", err, u.MainWindow())
				return
			}
			currentBid = fmt.Sprintf("%s (%d bids so far)", formatISKAmount(float64(top.Amount)), total)
		}
		fi = slices.Concat(fi, []*widget.FormItem{
			{Text: "Starting Bid", Widget: widget.NewLabel(o.Price.StringFunc("?", formatISKAmount))},
			{Text: "Buyout Price", Widget: widget.NewLabel(o.Buyout.StringFunc("?", formatISKAmount))},
			{Text: "Current Bid", Widget: widget.NewLabel(currentBid)},
			{Text: "Expires", Widget: widget.NewLabel(makeContractExpiresString(o.DateExpired, o.IsExpired()))},
		})
	}

	makeItemsInfo := func(c *app.CorporationContract) (fyne.CanvasObject, error) {
		vb := container.NewVBox()
		items, err := u.Corporation().ListContractItems(ctx, c.ID)
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
					u.InfoWindow().ShowType(it.Type.ID)
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
			xdialog.ShowErrorAndLog("Failed to show contract items", err, u.MainWindow())
			return
		}
		main.Add(x)
	}
	xwindow.Set(xwindow.Params{
		Title:   subTitle,
		Content: main,
		Window:  w,
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
