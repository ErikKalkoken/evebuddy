package screens

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

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// Status choices which are not a single category.
const (
	contractsStatusAll     = "All contracts"
	contractsStatusAllOpen = "All open contracts"
)

// Names of the contract filters, used as labels on desktop and as option names on mobile.
const (
	contractsFilterAssignee = "Assignee"
	contractsFilterIssuer   = "Issuer"
	contractsFilterTag      = "Tag"
	contractsFilterType     = "Type"
)

// contractsStatusOptions returns the status choices in UI order.
func contractsStatusOptions() []string {
	options := []string{contractsStatusAllOpen}
	for _, c := range app.ContractCategories() {
		options = append(options, c.Display())
	}
	return append(options, contractsStatusAll)
}

// contractsFilter is the selected value of each contract filter. Empty means not filtered.
type contractsFilter struct {
	assignee string
	issuer   string
	status   string
	tag      string
	typeName string
}

// match reports whether row r passes all filters.
func (f contractsFilter) match(r contractRow) bool {
	switch {
	case f.assignee != "" && r.assigneeName != f.assignee,
		f.issuer != "" && r.issuerName != f.issuer,
		f.tag != "" && !r.tags.Contains(f.tag),
		f.typeName != "" && r.typeName != f.typeName:
		return false
	}
	switch f.status {
	case "", contractsStatusAll:
		return true
	case contractsStatusAllOpen:
		return r.category.IsOpen()
	}
	return r.category.Display() == f.status
}

type contractRow struct {
	acceptor           optional.Optional[*app.EveEntity]
	assignee           optional.Optional[*app.EveEntity]
	assigneeName       string
	availability       app.ContractAvailability
	buyout             optional.Optional[float64]
	category           app.ContractCategory
	collateral         optional.Optional[float64]
	contractID         int64
	contractType       app.ContractType
	dateAccepted       optional.Optional[time.Time]
	dateCompleted      optional.Optional[time.Time]
	dateExpired        time.Time
	dateExpiredDisplay []widget.RichTextSegment
	dateIssued         time.Time
	daysToComplete     optional.Optional[int64]
	endLocation        optional.Optional[*app.EveLocationShort]
	isCorporation      bool
	isExpired          bool // TODO: make dynamic
	issuer             *app.EveEntity
	issuerName         string
	name               string
	objectID           int64
	ownerID            int64
	ownerName          string
	price              optional.Optional[float64]
	reward             optional.Optional[float64]
	searchTarget       string
	startLocation      optional.Optional[*app.EveLocationShort]
	status             app.ContractStatus
	statusDisplay      []widget.RichTextSegment
	statusText         string
	tags               set.Set[string]
	title              string
	typeName           string
	volume             optional.Optional[float64]
}

func newContractRowForCharacter(o *app.CharacterContract, characterName func(int64) string) contractRow {
	assigneeName := o.Assignee.StringFunc("", func(v *app.EveEntity) string {
		return v.Name
	})
	category := o.Category()
	return contractRow{
		acceptor:           o.Acceptor,
		assignee:           o.Assignee,
		assigneeName:       assigneeName,
		availability:       o.Availability,
		buyout:             o.Buyout,
		category:           category,
		collateral:         o.Collateral,
		contractID:         o.ContractID,
		contractType:       o.Type,
		dateAccepted:       o.DateAccepted,
		dateCompleted:      o.DateCompleted,
		dateExpired:        o.DateExpired,
		dateExpiredDisplay: makeDateExpiredDisplay(o.IsExpired(), o.DateExpired),
		dateIssued:         o.DateIssued,
		daysToComplete:     o.DaysToComplete,
		endLocation:        o.EndLocation,
		isCorporation:      false,
		isExpired:          o.IsExpired(),
		issuer:             o.IssuerEffective(),
		issuerName:         o.IssuerEffective().Name,
		name:               o.NameDisplay(),
		objectID:           o.ID,
		ownerID:            o.CharacterID,
		ownerName:          characterName(o.CharacterID),
		price:              o.Price,
		reward:             o.Reward,
		searchTarget:       makeSearchTarget(o.Items, o.Title),
		startLocation:      o.StartLocation,
		status:             o.Status,
		statusDisplay:      makeContractStatusDisplay(o.Status, category),
		statusText:         contractStatusText(o.Status, category),
		title:              o.Title.ValueOrFallback("-"),
		typeName:           o.Type.Display(),
		volume:             o.Volume,
	}
}

func newContractRowForCorporation(o *app.CorporationContract, corporation *app.Corporation) contractRow {
	assigneeName := o.Assignee.StringFunc("", func(v *app.EveEntity) string {
		return v.Name
	})
	category := o.Category()
	return contractRow{
		acceptor:           o.Acceptor,
		assignee:           o.Assignee,
		assigneeName:       assigneeName,
		availability:       o.Availability,
		buyout:             o.Buyout,
		category:           category,
		collateral:         o.Collateral,
		contractID:         o.ContractID,
		contractType:       o.Type,
		dateAccepted:       o.DateAccepted,
		dateCompleted:      o.DateCompleted,
		dateExpired:        o.DateExpired,
		dateExpiredDisplay: makeDateExpiredDisplay(o.IsExpired(), o.DateExpired),
		dateIssued:         o.DateIssued,
		daysToComplete:     o.DaysToComplete,
		endLocation:        o.EndLocation,
		isCorporation:      true,
		isExpired:          o.IsExpired(),
		issuer:             o.IssuerEffective(),
		issuerName:         o.IssuerEffective().Name,
		name:               o.NameDisplay(),
		objectID:           o.ID,
		ownerID:            corporation.ID,
		ownerName:          corporation.NameOrZero(),
		price:              o.Price,
		reward:             o.Reward,
		searchTarget:       makeSearchTarget(o.Items, o.Title),
		startLocation:      o.StartLocation,
		status:             o.Status,
		statusDisplay:      makeContractStatusDisplay(o.Status, category),
		statusText:         contractStatusText(o.Status, category),
		title:              o.Title.ValueOrFallback("-"),
		typeName:           o.Type.Display(),
		volume:             o.Volume,
	}
}

func makeSearchTarget(items []string, title optional.Optional[string]) string {
	var token []string
	for _, it := range items {
		token = append(token, strings.ToLower(it))
	}
	if v, ok := title.Value(); ok {
		token = append(token, strings.ToLower(v))
	}
	return strings.Join(token, "~")
}

// contractStatusText returns the status and explains why it requires attention
// when the status alone does not.
func contractStatusText(status app.ContractStatus, category app.ContractCategory) string {
	s := status.Display()
	if category != app.ContractCategoryRequiresAttention {
		return s
	}
	switch status {
	case app.ContractStatusOutstanding:
		return s + " (expired)"
	case app.ContractStatusInProgress:
		return s + " (overdue)"
	}
	return s
}

// makeContractStatusDisplay returns the status in the color of its category.
func makeContractStatusDisplay(status app.ContractStatus, category app.ContractCategory) []widget.RichTextSegment {
	return xwidget.RichTextSegmentsFromText(contractStatusText(status, category), widget.RichTextStyle{
		ColorName: category.Color(),
	})
}

func makeDateExpiredDisplay(isExpired bool, dateExpired time.Time) []widget.RichTextSegment {
	var text string
	var color fyne.ThemeColorName
	if isExpired {
		text = "EXPIRED"
		color = theme.ColorNameError
	} else {
		text = ihumanize.RelTime(dateExpired)
		color = theme.ColorNameForeground
	}
	return xwidget.RichTextSegmentsFromText(text, widget.RichTextStyle{
		ColorName: color,
	})
}

// Contracts is a UI element for showing Contracts.
// It either shows all character Contracts or the Contracts for a corporation.
type Contracts struct {
	widget.BaseWidget

	OnUpdate func(open int)

	body           fyne.CanvasObject
	columnSorter   *xwidget.ColumnSorter[contractRow]
	corporation    atomic.Pointer[app.Corporation]
	filterChip     *xwidget.FilterChipCompact // only on mobile
	filterRun      latestRun
	footer         *widget.Label
	forCorporation bool // reports whether it runs in corporation mode
	rows           []contractRow
	rowsFiltered   []contractRow
	searchEntry    *xwidget.SearchEntry
	selectAssignee *kxwidget.FilterChipSelect // select chips only on desktop
	selectIssuer   *kxwidget.FilterChipSelect
	selectStatus   *widget.Select
	selectTag      *kxwidget.FilterChipSelect
	selectType     *kxwidget.FilterChipSelect
	sortChip       *kxwidget.SortChip
	u              baseUI
}

func NewContractsForCorporation(u baseUI) *Contracts {
	return newContracts(u, true)
}

func NewContractsForCharacters(u baseUI) *Contracts {
	return newContracts(u, false)
}

func newContracts(u baseUI, forCorporation bool) *Contracts {
	columns := xwidget.NewDataColumns([]xwidget.DataColumn[contractRow]{{
		Label: "Contract",
		Width: 300,
		Sort: func(a, b contractRow) int {
			return strings.Compare(a.name, b.name)
		},
		Update: func(r contractRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.name)
		},
	}, {
		Label: "Type",
		Width: 120,
		Sort: func(a, b contractRow) int {
			return strings.Compare(a.typeName, b.typeName)
		},
		Update: func(r contractRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.typeName)
		},
	}, {
		Label: "From",
		Width: 150,
		Sort: func(a, b contractRow) int {
			return xstrings.CompareIgnoreCase(a.issuerName, b.issuerName)
		},
		Update: func(r contractRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.issuerName)
		},
	}, {
		Label: "To",
		Width: 150,
		Sort: func(a, b contractRow) int {
			return xstrings.CompareIgnoreCase(a.assigneeName, b.assigneeName)
		},
		Update: func(r contractRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.assigneeName)
		},
	}, {
		Label: "Status",
		Width: 180,
		Sort: func(a, b contractRow) int {
			return strings.Compare(a.statusText, b.statusText)
		},
		Update: func(r contractRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).Set(r.statusDisplay)
		},
	}, {
		Label: "Date Issued",
		Width: ui.ColumnWidthDateTime,
		Sort: func(a, b contractRow) int {
			return a.dateIssued.Compare(b.dateIssued)
		},
		Update: func(r contractRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.dateIssued.Format(app.DateTimeFormat))
		},
	}, {
		Label: "Time Left",
		Width: 100,
		Sort: func(a, b contractRow) int {
			return a.dateExpired.Compare(b.dateExpired)
		},
		Update: func(r contractRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).Set(r.dateExpiredDisplay)
		},
	}, {
		Label: "Description",
		Width: 300,
		Sort: func(a, b contractRow) int {
			return strings.Compare(a.title, b.title)
		},
		Update: func(r contractRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.title)
		},
	}})
	a := &Contracts{
		forCorporation: forCorporation,
		columnSorter:   xwidget.NewColumnSorter(columns, "Date Issued", xwidget.SortDesc),
		footer:         ui.NewLabelWithTruncation(""),
		u:              u,
	}
	a.ExtendBaseWidget(a)

	if a.u.IsMobile() {
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
			func(_ int, r contractRow) {
				if a.forCorporation {
					ShowCorporationContract(a.u, r)
				} else {
					ShowCharacterContract(a.u, r)
				}
			},
		)
	}

	placeholder := "Search items and descriptions"
	if a.u.IsMobile() {
		placeholder = "Search" // shares the row with the chips
	}
	a.searchEntry = xwidget.NewSearchEntry(placeholder, func(_ string) {
		a.filterRowsAsync("")
	})

	a.selectStatus = widget.NewSelect(contractsStatusOptions(), func(string) {
		a.filterRowsAsync("")
	})
	a.selectStatus.Selected = contractsStatusAllOpen
	a.selectStatus.PlaceHolder = app.ContractCategoryRequiresAttention.Display() // widest option, so the select fits all choices

	if a.u.IsMobile() {
		a.filterChip = xwidget.NewFilterChipCompact(nil, func(map[string]string) {
			a.filterRowsAsync("")
		})
	} else {
		makeSelectWithSearch := func(label string) *kxwidget.FilterChipSelect {
			return kxwidget.NewFilterChipSelectWithSearch(label, []string{}, func(string) {
				a.filterRowsAsync("")
			}, a.u.MainWindow())
		}
		makeSelect := func(label string) *kxwidget.FilterChipSelect {
			return kxwidget.NewFilterChipSelect(label, []string{}, func(string) {
				a.filterRowsAsync("")
			})
		}
		a.selectAssignee = makeSelectWithSearch(contractsFilterAssignee)
		a.selectIssuer = makeSelectWithSearch(contractsFilterIssuer)
		a.selectType = makeSelect(contractsFilterType)
		a.selectTag = makeSelect(contractsFilterTag)
	}

	a.sortChip = a.columnSorter.NewSortChip(func() {
		a.filterRowsAsync("")
	})

	// Signals
	a.u.Signals().AppInit.AddListener(func(ctx context.Context, _ struct{}) {
		a.update(ctx)
	})

	if a.forCorporation {
		a.u.Signals().CurrentCorporationExchanged.AddListener(func(ctx context.Context, c *app.Corporation) {
			a.corporation.Store(c)
			fyne.Do(func() {
				a.searchEntry.ClearSilent()
				a.selectStatus.Selected = contractsStatusAllOpen
				a.selectStatus.Refresh()
				if a.filterChip != nil {
					a.filterChip.ResetSilent()
					return
				}
				clearSelectsSilent(a.selectAssignee, a.selectIssuer, a.selectTag, a.selectType)
			})
			a.update(ctx)
		})
		a.u.Signals().CorporationSectionChanged.AddListener(func(ctx context.Context, arg app.CorporationSectionUpdated) {
			if a.corporation.Load().IDOrZero() != arg.CorporationID {
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
		a.u.Signals().TagsChanged.AddListener(func(ctx context.Context, _ struct{}) {
			a.update(ctx)
		})
	}
	return a
}

func (a *Contracts) CreateRenderer() fyne.WidgetRenderer {
	var topBox *fyne.Container
	if a.u.IsMobile() {
		topBox = container.NewVBox(
			a.selectStatus,
			container.NewBorder(nil, nil, nil, container.NewHBox(a.filterChip, a.sortChip), a.searchEntry),
		)
	} else {
		filter := container.NewHBox(
			a.selectStatus,
			a.selectType,
			a.selectIssuer,
			a.selectAssignee,
		)
		if !a.forCorporation {
			filter.Add(a.selectTag)
		}
		topBox = container.NewBorder(nil, nil, filter, nil, a.searchEntry)
	}

	c := container.NewBorder(
		topBox,
		a.footer,
		nil,
		nil,
		a.body,
	)
	return widget.NewSimpleRenderer(c)
}

// currentFilter returns the selected filters: from the compact chip on mobile
// and from the filter chips on desktop. Status always comes from its select.
func (a *Contracts) currentFilter() contractsFilter {
	if a.filterChip != nil {
		s := a.filterChip.Selected()
		return contractsFilter{
			assignee: s[contractsFilterAssignee],
			issuer:   s[contractsFilterIssuer],
			status:   a.selectStatus.Selected,
			tag:      s[contractsFilterTag],
			typeName: s[contractsFilterType],
		}
	}
	return contractsFilter{
		assignee: a.selectAssignee.Selected,
		issuer:   a.selectIssuer.Selected,
		status:   a.selectStatus.Selected,
		tag:      a.selectTag.Selected,
		typeName: a.selectType.Selected,
	}
}

func (a *Contracts) makeDataList() *xwidget.StripedList {
	p := theme.Padding()
	l := xwidget.NewStripedList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			title := widget.NewLabelWithStyle("Template", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			et := widget.NewLabel("Template")
			status := xwidget.NewRichTextWithText("Template")
			issuer := widget.NewLabel("Template")
			assignee := widget.NewLabel("Template")
			dateExpired := xwidget.NewRichTextWithText("Template")
			return container.New(layout.NewCustomPaddedVBoxLayout(-p),
				title,
				container.NewHBox(et, layout.NewSpacer(), status),
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
			box[2].(*xwidget.RichText).Set(r.statusDisplay)

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
			ShowCorporationContract(a.u, r)
		} else {
			ShowCharacterContract(a.u, r)
		}
	}
	return l
}

func (a *Contracts) filterRowsAsync(sortCol string) {
	isLatest := a.filterRun.start()
	totalRows := len(a.rows)
	rows := slices.Clone(a.rows)
	filter := a.currentFilter()
	search := strings.ToLower(a.searchEntry.Text)
	sortCol, dir, doSort := a.columnSorter.CalcSort(sortCol)

	runAsync(func() {
		rows = slices.DeleteFunc(rows, func(r contractRow) bool {
			return !filter.match(r)
		})
		if len(search) > 1 {
			rows = slices.DeleteFunc(rows, func(r contractRow) bool {
				return !strings.Contains(r.searchTarget, search)
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
			if !isLatest() {
				return
			}
			a.footer.Text = footer
			a.footer.Importance = widget.MediumImportance
			a.footer.Refresh()
			if a.filterChip != nil {
				options := []xwidget.FilterOption{
					xwidget.NewFilterOptionMultiChoice(contractsFilterType, typeOptions),
					xwidget.NewFilterOptionMultiChoiceWithSearch(contractsFilterIssuer, issueOptions),
					xwidget.NewFilterOptionMultiChoiceWithSearch(contractsFilterAssignee, assigneeOptions),
				}
				if !a.forCorporation {
					options = append(options, xwidget.NewFilterOptionMultiChoice(contractsFilterTag, tagOptions))
				}
				a.filterChip.SetOptions(options...)
			} else {
				a.selectTag.SetOptions(tagOptions)
				a.selectIssuer.SetOptions(issueOptions)
				a.selectAssignee.SetOptions(assigneeOptions)
				a.selectType.SetOptions(typeOptions)
			}
			a.rowsFiltered = rows
			a.body.Refresh()
		})
	})
}

func (a *Contracts) update(ctx context.Context) {
	var err error
	var rows []contractRow
	if a.forCorporation {
		rows, err = a.fetchRowsCorporation(ctx)
	} else {
		rows, err = a.fetchRowsOverview(ctx)
	}
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		slog.Error("Failed to refresh contracts UI", "err", err)
		fyne.Do(func() {
			a.footer.Text = fmt.Sprintf("ERROR: %s", a.u.ErrorDisplay(err))
			a.footer.Importance = widget.DangerImportance
			a.footer.Refresh()
		})
		return
	}
	var openCount int
	for _, r := range rows {
		if r.category.IsOpen() {
			openCount++
		}
	}
	fyne.Do(func() {
		a.rows = rows
		a.filterRowsAsync("")
		if a.OnUpdate != nil {
			a.OnUpdate(openCount)
		}
	})
}

func (a *Contracts) fetchRowsCorporation(ctx context.Context) ([]contractRow, error) {
	corporation := a.corporation.Load()
	if corporation == nil {
		return nil, nil
	}
	oo, err := a.u.Corporation().ListCorporationContracts(ctx, corporation.ID)
	if err != nil {
		return nil, err
	}
	var rows []contractRow
	for _, c := range oo {
		r := newContractRowForCorporation(c, corporation)
		rows = append(rows, r)
	}
	return rows, nil
}

func (a *Contracts) fetchRowsOverview(ctx context.Context) ([]contractRow, error) {
	oo, err := a.u.Character().ListAllContracts(ctx)
	if err != nil {
		return nil, err
	}
	// Remove duplicate oo2 between the user's own characters
	oo2 := slices.CompactFunc(oo, func(a, b *app.CharacterContract) bool {
		return a.ContractID == b.ContractID &&
			a.Type == b.Type &&
			a.Issuer.ID == b.Issuer.ID &&
			optional.EqualFunc(a.Assignee, b.Assignee, func(x, y *app.EveEntity) bool { return x.ID == y.ID }) &&
			a.DateIssued.Equal(b.DateIssued)
	})
	characters, err := a.u.Character().CharacterNames(ctx)
	if err != nil {
		return nil, err
	}
	var rows []contractRow
	tagsByCharacter := make(map[int64]set.Set[string])
	for _, c := range oo2 {
		r := newContractRowForCharacter(c, func(id int64) string {
			return characters[id]
		})
		tags, ok := tagsByCharacter[c.CharacterID]
		if !ok {
			tags, err = a.u.Character().ListTagsForCharacter(ctx, c.CharacterID)
			if err != nil {
				return nil, err
			}
			tagsByCharacter[c.CharacterID] = tags
		}
		r.tags = tags
		rows = append(rows, r)
	}
	return rows, nil
}
