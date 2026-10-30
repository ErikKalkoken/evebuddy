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
	"github.com/ErikKalkoken/evebuddy/internal/xsync"
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
		return r.categories.ContainsAny(app.ContractCategoriesOpen.All())
	}
	return r.categories.ContainsFunc(func(c app.ContractCategory) bool {
		return c.Display() == f.status
	})
}

type contractRow struct {
	acceptor        optional.Optional[*app.EveEntity]
	acceptorIsMine  bool // accepted by one of the user's characters
	assignee        optional.Optional[*app.EveEntity]
	assigneeName    string
	availability    app.ContractAvailability
	buyout          optional.Optional[float64]
	categories      set.Set[app.ContractCategory]
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
	deadlineKind    contractDeadlineKind
	timeLeftDisplay []widget.RichTextSegment
	timeLeftText    string
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
	tags            set.Set[string]
	title           string
	typeName        string
	volume          optional.Optional[float64]
}

// newContractRow returns a new row for a contract of a character or corporation.
func newContractRow(o *app.Contract, ownerID int64, ownerName string, isCorporation bool, myCharacterIDs, myCorporationIDs set.Set[int64]) contractRow {
	assigneeName := o.Assignee.StringFunc("", func(v *app.EveEntity) string {
		return v.Name
	})
	if assigneeName == "" && o.Availability == app.ContractAvailabilityPublic {
		assigneeName = "[Public]" // brackets so it can't be mistaken for a character name
	}
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
		isCorporation:  isCorporation,
		issuer:         o.IssuerEffective(),
		issuerName:     o.IssuerEffective().Name,
		name:           o.NameDisplay(),
		objectID:       o.ID,
		ownerID:        ownerID,
		ownerName:      ownerName,
		price:          o.Price,
		reward:         o.Reward,
		searchTarget:   makeSearchTarget(o.Items, o.Title),
		startLocation:  o.StartLocation,
		status:         o.Status,
		title:          o.Title.ValueOrFallback("-"),
		typeName:       o.Type.Display(),
		volume:         o.Volume,
	}
	r.init(o.IssuerEffective().ID, o.Acceptor, myCharacterIDs, myCorporationIDs, time.Now())
	return r
}

func makeSearchTarget(items []string, title optional.Optional[string]) string {
	token := make([]string, 0, len(items)+1)
	for _, it := range items {
		token = append(token, strings.ToLower(it))
	}
	if v, ok := title.Value(); ok {
		token = append(token, strings.ToLower(v))
	}
	return strings.Join(token, "~")
}

// init sets the derived fields of a new row. It must be called once after the row is created.
// Contracts issued or accepted for one of the user's corporations count as the user's.
func (r *contractRow) init(issuerID int64, acceptor optional.Optional[*app.EveEntity], myCharacterIDs, myCorporationIDs set.Set[int64], now time.Time) {
	myIDs := set.Union(myCharacterIDs, myCorporationIDs)
	r.issuerIsMine = myIDs.Contains(issuerID)
	r.acceptorIsMine = optional.Map(acceptor, false, func(v *app.EveEntity) bool {
		return myIDs.Contains(v.ID)
	})
	r.deadline, r.deadlineKind = contractDeadline(r.status, r.dateExpired, r.dateAccepted, r.daysToComplete)
	r.statusDisplay = makeContractStatusDisplay(r.status)
	r.setTimeDependent(now)
}

// owner returns the character or corporation which owns the contract.
func (r *contractRow) owner() *app.EveEntity {
	category := app.EveEntityCharacter
	if r.isCorporation {
		category = app.EveEntityCorporation
	}
	return &app.EveEntity{ID: r.ownerID, Name: r.ownerName, Category: category}
}

// setTimeDependent sets the fields which change with time.
func (r *contractRow) setTimeDependent(now time.Time) {
	r.categories, r.isExpired, r.isOverdue = app.ContractStateAt(r.status, r.dateExpired, r.dateAccepted, r.daysToComplete, r.issuerIsMine, r.acceptorIsMine, now)
	var isPassed bool
	r.timeLeftText, isPassed = contractTimeLeftText(r.deadline, r.deadlineKind.passedText(), now)
	r.timeLeftDisplay = makeTimeLeftDisplay(r.timeLeftText, isPassed)
}

// updateTimeDependent recomputes the fields which change with time
// and reports whether any of them changed.
func (r *contractRow) updateTimeDependent(now time.Time) bool {
	categories, isExpired, isOverdue, timeLeft := r.categories, r.isExpired, r.isOverdue, r.timeLeftText
	r.setTimeDependent(now)
	return !r.categories.Equal(categories) || r.isExpired != isExpired || r.isOverdue != isOverdue || r.timeLeftText != timeLeft
}

// contractsOutstandingCount returns how many rows are outstanding.
func contractsOutstandingCount(rows []contractRow) int {
	var n int
	for _, r := range rows {
		if r.categories.Contains(app.ContractCategoryOutstanding) {
			n++
		}
	}
	return n
}

// makeContractStatusDisplay returns the status text in the color of the status.
func makeContractStatusDisplay(status app.ContractStatus) []widget.RichTextSegment {
	return xwidget.RichTextSegmentsFromText(status.Display(), widget.RichTextStyle{
		ColorName: status.Color(),
	})
}

// contractDeadlineKind is the kind of deadline relevant for a contract.
type contractDeadlineKind uint8

const (
	contractDeadlineNone     contractDeadlineKind = iota
	contractDeadlineExpiry                        // contract must be accepted before it expires
	contractDeadlineDelivery                      // accepted contract must be completed in time
)

// label returns a short description of the deadline.
func (k contractDeadlineKind) label() string {
	switch k {
	case contractDeadlineExpiry:
		return "Expires"
	case contractDeadlineDelivery:
		return "Due"
	}
	return ""
}

// passedText returns the text to show once the deadline has passed.
func (k contractDeadlineKind) passedText() string {
	switch k {
	case contractDeadlineExpiry:
		return "EXPIRED"
	case contractDeadlineDelivery:
		return "OVERDUE"
	}
	return ""
}

// contractDeadline returns the deadline relevant for a status and its kind:
// the expiry for outstanding contracts and the delivery deadline for accepted couriers.
// Closed contracts have no deadline. The expiry no longer applies once a contract is accepted.
func contractDeadline(status app.ContractStatus, dateExpired time.Time, dateAccepted optional.Optional[time.Time], daysToComplete optional.Optional[int64]) (optional.Optional[time.Time], contractDeadlineKind) {
	switch status {
	case app.ContractStatusOutstanding:
		return optional.New(dateExpired), contractDeadlineExpiry
	case app.ContractStatusInProgress:
		if v, ok := app.ContractDeliveryDeadline(dateAccepted, daysToComplete); ok {
			return optional.New(v), contractDeadlineDelivery
		}
	}
	return optional.Optional[time.Time]{}, contractDeadlineNone
}

// contractTimeLeftText returns the time left until the deadline
// or the passed text when it is over. It also reports whether the deadline has passed.
func contractTimeLeftText(deadline optional.Optional[time.Time], passed string, now time.Time) (string, bool) {
	v, ok := deadline.Value()
	if !ok {
		return "", false
	}
	if v.Before(now) {
		return passed, true
	}
	return ihumanize.Duration(v.Sub(now)), false
}

// makeTimeLeftDisplay returns the time left text, in red when the deadline has passed.
func makeTimeLeftDisplay(text string, isPassed bool) []widget.RichTextSegment {
	if !isPassed {
		return xwidget.RichTextSegmentsFromText(text)
	}
	return xwidget.RichTextSegmentsFromText(text, widget.RichTextStyle{
		ColorName: theme.ColorNameError,
	})
}

// compareContractDeadlines compares two deadlines.
// Missing deadlines sort last in ascending order (and first in descending order).
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

	// OnUpdate reports the outstanding contracts; empty when the update failed.
	OnUpdate func(outstanding optional.Optional[int])

	body           fyne.CanvasObject
	columnSorter   *xwidget.ColumnSorter[contractRow]
	corporation    atomic.Pointer[app.Corporation]
	filterChip     *xwidget.FilterChipCompact // only on mobile
	filterRun      xsync.LatestRun
	updateRun      xsync.LatestRun
	footerLabel    *widget.Label
	errorLabel     *widget.Label
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
			return strings.Compare(a.status.String(), b.status.String())
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
		footerLabel:    ui.NewLabelWithTruncation(""),
		errorLabel:     ui.NewLabelWithTruncation(""),
		u:              u,
	}
	a.ExtendBaseWidget(a)

	a.errorLabel.Importance = widget.DangerImportance
	a.errorLabel.Hide()

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
				ShowContract(a.u, r)
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
		// Which contracts require attention depends on the user's characters.
		a.u.Signals().CharacterAdded.AddListener(func(ctx context.Context, _ *app.Character) {
			a.update(ctx)
		})
		a.u.Signals().CharacterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
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
	// Which contracts require attention depends on the corporations of the user's characters.
	a.u.Signals().CorporationsChanged.AddListener(func(ctx context.Context, _ struct{}) {
		a.update(ctx)
	})
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
		container.NewVBox(a.errorLabel, a.footerLabel),
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

			if r.deadlineKind == contractDeadlineNone {
				main[4].(*xwidget.RichText).Set(xwidget.RichTextSegmentsFromText(""))
			} else {
				main[4].(*xwidget.RichText).Set(xwidget.RichTextSegmentsInlined(
					xwidget.RichTextSegmentsFromText(r.deadlineKind.label()+" "),
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
		ShowContract(a.u, a.rowsFiltered[id])
	}
	return l
}

func (a *Contracts) filterRowsAsync(sortCol string) {
	isLatest := a.filterRun.Start()
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
			a.footerLabel.Text = footer
			a.footerLabel.Importance = widget.MediumImportance
			a.footerLabel.Refresh()
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
	isLatest := a.updateRun.Start()
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
			if !isLatest() {
				return
			}
			xslices.Clear(&a.rows)
			a.filterRowsAsync("")
			if a.OnUpdate != nil {
				a.OnUpdate(optional.Optional[int]{})
			}
			a.errorLabel.SetText(fmt.Sprintf("ERROR: %s", a.u.ErrorDisplay(err)))
			a.errorLabel.Show()
			a.Refresh()
		})
		return
	}
	outstandingCount := contractsOutstandingCount(rows)
	fyne.Do(func() {
		if !isLatest() {
			return
		}
		a.errorLabel.Hide()
		a.rows = rows
		a.filterRowsAsync("")
		if a.OnUpdate != nil {
			a.OnUpdate(optional.New(outstandingCount))
		}
	})
}

// refreshTimeDependent updates the time dependent fields of all rows,
// so that the list stays current without new data. Must run on the main thread.
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
	myCharacterIDs, err := a.u.Character().ListCharacterIDs(ctx)
	if err != nil {
		return nil, err
	}
	myCorporationIDs, err := a.u.Character().ListCharacterCorporationIDs(ctx)
	if err != nil {
		return nil, err
	}
	rows := make([]contractRow, len(oo))
	for i, c := range oo {
		r := newContractRow(&c.Contract, corporation.ID, corporation.NameOrZero(), true, myCharacterIDs, myCorporationIDs)
		rows[i] = r
	}
	return rows, nil
}

func (a *Contracts) fetchRowsOverview(ctx context.Context) ([]contractRow, error) {
	oo, err := a.u.Character().ListAllContracts(ctx)
	if err != nil {
		return nil, err
	}
	characters, err := a.u.Character().CharacterNames(ctx)
	if err != nil {
		return nil, err
	}
	myCharacterIDs := set.Collect(maps.Keys(characters))
	myCorporationIDs, err := a.u.Character().ListCharacterCorporationIDs(ctx)
	if err != nil {
		return nil, err
	}
	rows := make([]contractRow, len(oo))
	tagsByCharacter := make(map[int64]set.Set[string])
	for i, c := range oo {
		r := newContractRow(&c.Contract, c.CharacterID, characters[c.CharacterID], false, myCharacterIDs, myCorporationIDs)
		tags, ok := tagsByCharacter[c.CharacterID]
		if !ok {
			tags, err = a.u.Character().ListTagsForCharacter(ctx, c.CharacterID)
			if err != nil {
				return nil, err
			}
			tagsByCharacter[c.CharacterID] = tags
		}
		r.tags = tags
		rows[i] = r
	}
	return rows, nil
}
