package screens

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
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
	acceptor        optional.Optional[*app.EveEntity]
	acceptorIsMine  bool // accepted by one of the user's characters
	assignee        optional.Optional[*app.EveEntity]
	assigneeName    string
	availability    app.ContractAvailability
	buyout          optional.Optional[float64]
	category        app.ContractCategory
	collateral      optional.Optional[float64]
	contractID      int64
	contractType    app.ContractType
	dateAccepted    optional.Optional[time.Time]
	dateCompleted   optional.Optional[time.Time]
	dateExpired     time.Time
	dateIssued      time.Time
	daysToComplete  optional.Optional[int64]
	endLocation     optional.Optional[*app.EveLocationShort]
	isCorporation   bool
	isExpired       bool
	isOverdue       bool
	deadline        optional.Optional[time.Time] // the deadline relevant for the current status
	deadlineLabel   string                       // describes the deadline, e.g. "Expires"
	timeLeftDisplay []widget.RichTextSegment
	issuer          *app.EveEntity
	issuerIsMine    bool // issued by one of the user's characters
	issuerName      string
	name            string
	objectID        int64
	ownerID         int64
	ownerName       string
	price           optional.Optional[float64]
	reward          optional.Optional[float64]
	searchTarget    string
	startLocation   optional.Optional[*app.EveLocationShort]
	status          app.ContractStatus
	statusDisplay   []widget.RichTextSegment
	statusText      string
	tags            set.Set[string]
	title           string
	typeName        string
	volume          optional.Optional[float64]
}

func newContractRowForCharacter(o *app.CharacterContract, characterName func(int64) string, myCharacterIDs set.Set[int64]) contractRow {
	assigneeName := o.Assignee.StringFunc("", func(v *app.EveEntity) string {
		return v.Name
	})
	r := contractRow{
		acceptor:       o.Acceptor,
		assignee:       o.Assignee,
		assigneeName:   assigneeName,
		availability:   o.Availability,
		buyout:         o.Buyout,
		collateral:     o.Collateral,
		contractID:     o.ContractID,
		contractType:   o.Type,
		dateAccepted:   o.DateAccepted,
		dateCompleted:  o.DateCompleted,
		dateExpired:    o.DateExpired,
		dateIssued:     o.DateIssued,
		daysToComplete: o.DaysToComplete,
		endLocation:    o.EndLocation,
		isCorporation:  false,
		issuer:         o.IssuerEffective(),
		issuerName:     o.IssuerEffective().Name,
		name:           o.NameDisplay(),
		objectID:       o.ID,
		ownerID:        o.CharacterID,
		ownerName:      characterName(o.CharacterID),
		price:          o.Price,
		reward:         o.Reward,
		searchTarget:   makeSearchTarget(o.Items, o.Title),
		startLocation:  o.StartLocation,
		status:         o.Status,
		title:          o.Title.ValueOrFallback("-"),
		typeName:       o.Type.Display(),
		volume:         o.Volume,
	}
	r.issuerIsMine = myCharacterIDs.Contains(o.Issuer.ID)
	r.acceptorIsMine = isMyEntity(o.Acceptor, myCharacterIDs)
	r.updateTimeDependent(time.Now())
	return r
}

func newContractRowForCorporation(o *app.CorporationContract, corporation *app.Corporation, myCharacterIDs set.Set[int64]) contractRow {
	assigneeName := o.Assignee.StringFunc("", func(v *app.EveEntity) string {
		return v.Name
	})
	r := contractRow{
		acceptor:       o.Acceptor,
		assignee:       o.Assignee,
		assigneeName:   assigneeName,
		availability:   o.Availability,
		buyout:         o.Buyout,
		collateral:     o.Collateral,
		contractID:     o.ContractID,
		contractType:   o.Type,
		dateAccepted:   o.DateAccepted,
		dateCompleted:  o.DateCompleted,
		dateExpired:    o.DateExpired,
		dateIssued:     o.DateIssued,
		daysToComplete: o.DaysToComplete,
		endLocation:    o.EndLocation,
		isCorporation:  true,
		issuer:         o.IssuerEffective(),
		issuerName:     o.IssuerEffective().Name,
		name:           o.NameDisplay(),
		objectID:       o.ID,
		ownerID:        corporation.ID,
		ownerName:      corporation.NameOrZero(),
		price:          o.Price,
		reward:         o.Reward,
		searchTarget:   makeSearchTarget(o.Items, o.Title),
		startLocation:  o.StartLocation,
		status:         o.Status,
		title:          o.Title.ValueOrFallback("-"),
		typeName:       o.Type.Display(),
		volume:         o.Volume,
	}
	r.issuerIsMine = myCharacterIDs.Contains(o.Issuer.ID)
	r.acceptorIsMine = isMyEntity(o.Acceptor, myCharacterIDs)
	r.updateTimeDependent(time.Now())
	return r
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

// isMyEntity reports whether an entity is one of the user's characters.
func isMyEntity(o optional.Optional[*app.EveEntity], myCharacterIDs set.Set[int64]) bool {
	v, ok := o.Value()
	return ok && myCharacterIDs.Contains(v.ID)
}

// updateTimeDependent recomputes the fields which change with time
// and reports whether any of them changed.
func (r *contractRow) updateTimeDependent(now time.Time) bool {
	category := app.ContractCategoryAt(r.status, r.dateExpired, r.dateAccepted, r.daysToComplete, r.issuerIsMine, r.acceptorIsMine, now)
	isExpired := app.ContractIsExpiredAt(r.dateExpired, now)
	isOverdue := app.ContractIsOverdueAt(r.dateAccepted, r.daysToComplete, now)
	if category == r.category && isExpired == r.isExpired && isOverdue == r.isOverdue {
		return false
	}
	r.category = category
	r.isExpired = isExpired
	r.isOverdue = isOverdue
	r.statusText = contractStatusText(r.status, isExpired, isOverdue)
	r.statusDisplay = makeContractStatusDisplay(r.statusText, category)
	var passed string
	r.deadline, r.deadlineLabel, passed = contractDeadline(r.status, r.dateExpired, r.dateAccepted, r.daysToComplete)
	r.timeLeftDisplay = makeTimeLeftDisplay(r.deadline, passed, now)
	return true
}

// contractsAttentionCount returns how many rows require attention.
func contractsAttentionCount(rows []contractRow) int {
	var n int
	for _, r := range rows {
		if r.category == app.ContractCategoryRequiresAttention {
			n++
		}
	}
	return n
}

// contractStatusText returns the status and notes when an open contract
// has passed its deadline.
func contractStatusText(status app.ContractStatus, isExpired, isOverdue bool) string {
	s := status.Display()
	switch {
	case status == app.ContractStatusOutstanding && isExpired:
		return s + " (expired)"
	case status == app.ContractStatusInProgress && isOverdue:
		return s + " (overdue)"
	}
	return s
}

// makeContractStatusDisplay returns the status text in the color of its category.
func makeContractStatusDisplay(text string, category app.ContractCategory) []widget.RichTextSegment {
	return xwidget.RichTextSegmentsFromText(text, widget.RichTextStyle{
		ColorName: category.Color(),
	})
}

// contractDeadline returns the deadline relevant for a status, a label for it
// and the text to show once it has passed: the expiry for outstanding contracts
// and the delivery deadline for accepted couriers.
// Finished and other closed contracts have no deadline.
func contractDeadline(status app.ContractStatus, dateExpired time.Time, dateAccepted optional.Optional[time.Time], daysToComplete optional.Optional[int64]) (deadline optional.Optional[time.Time], label, passed string) {
	switch status {
	case app.ContractStatusOutstanding:
		return optional.New(dateExpired), "Expires", "EXPIRED"
	case app.ContractStatusInProgress:
		if v, ok := app.ContractDeliveryDeadline(dateAccepted, daysToComplete); ok {
			return optional.New(v), "Due", "OVERDUE"
		}
		return optional.New(dateExpired), "Expires", "EXPIRED"
	}
	return optional.Optional[time.Time]{}, "", ""
}

// makeTimeLeftDisplay returns the time left until the deadline
// or the passed text when it is over.
func makeTimeLeftDisplay(deadline optional.Optional[time.Time], passed string, now time.Time) []widget.RichTextSegment {
	v, ok := deadline.Value()
	if !ok {
		return xwidget.RichTextSegmentsFromText("")
	}
	if v.Before(now) {
		return xwidget.RichTextSegmentsFromText(passed, widget.RichTextStyle{
			ColorName: theme.ColorNameError,
		})
	}
	return xwidget.RichTextSegmentsFromText(ihumanize.RelTime(v))
}

// compareContractDeadlines compares two deadlines. Missing deadlines sort last.
func compareContractDeadlines(a, b optional.Optional[time.Time]) int {
	x, okA := a.Value()
	y, okB := b.Value()
	switch {
	case !okA && !okB:
		return 0
	case !okA:
		return 1
	case !okB:
		return -1
	}
	return x.Compare(y)
}

// Contracts is a UI element for showing Contracts.
// It either shows all character Contracts or the Contracts for a corporation.
type Contracts struct {
	widget.BaseWidget

	OnUpdate func(attention int)

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
			return compareContractDeadlines(a.deadline, b.deadline)
		},
		Update: func(r contractRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).Set(r.timeLeftDisplay)
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
		a.u.Signals().AppInit.AddListener(func(ctx context.Context, _ struct{}) {
			a.update(ctx)
		})
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
	a.u.Signals().RefreshTickerExpired.AddListener(func(_ context.Context, _ struct{}) {
		fyne.Do(func() {
			a.refreshTimeDependent(time.Now())
		})
	})
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

			if r.deadlineLabel == "" {
				main[4].(*xwidget.RichText).Set(xwidget.RichTextSegmentsFromText(""))
			} else {
				main[4].(*xwidget.RichText).Set(xwidget.InlineRichTextSegments(
					xwidget.RichTextSegmentsFromText(r.deadlineLabel+" "),
					r.timeLeftDisplay,
				))
			}
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
	attentionCount := contractsAttentionCount(rows)
	fyne.Do(func() {
		a.rows = rows
		a.filterRowsAsync("")
		if a.OnUpdate != nil {
			a.OnUpdate(attentionCount)
		}
	})
}

// refreshTimeDependent updates rows which crossed a deadline,
// so that list and badge stay current without new data. Must run on the main thread.
func (a *Contracts) refreshTimeDependent(now time.Time) {
	var changed bool
	for i := range a.rows {
		if a.rows[i].updateTimeDependent(now) {
			changed = true
		}
	}
	if !changed {
		return
	}
	a.filterRowsAsync("")
	if a.OnUpdate != nil {
		a.OnUpdate(contractsAttentionCount(a.rows))
	}
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
	characters, err := a.u.Character().CharacterNames(ctx)
	if err != nil {
		return nil, err
	}
	myCharacterIDs := set.Collect(maps.Keys(characters))
	var rows []contractRow
	for _, c := range oo {
		r := newContractRowForCorporation(c, corporation, myCharacterIDs)
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
	myCharacterIDs := set.Collect(maps.Keys(characters))
	var rows []contractRow
	tagsByCharacter := make(map[int64]set.Set[string])
	for _, c := range oo2 {
		r := newContractRowForCharacter(c, func(id int64) string {
			return characters[id]
		}, myCharacterIDs)
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
