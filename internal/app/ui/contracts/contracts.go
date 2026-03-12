// Package contracts provides widgets for building the contracts UI.
package contracts

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
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
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

type baseUI interface {
	Character() *characterservice.CharacterService
	Corporation() *corporationservice.CorporationService
	ErrorDisplay(err error) string
	EVEImage() ui.EVEImageService
	EVEUniverse() *eveuniverseservice.EVEUniverseService
	GetOrCreateWindow(id string, titles ...string) (window fyne.Window, created bool)
	InfoViewer() ui.InfoViewer
	IsDeveloperMode() bool
	IsMobile() bool
	MainWindow() fyne.Window
	Signals() *app.Signals
}

type contractRow struct {
	acceptor           optional.Optional[*app.EveEntity]
	assignee           optional.Optional[*app.EveEntity]
	assigneeName       string
	availability       app.ContractAvailability
	buyout             optional.Optional[float64]
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
	hasIssue           bool
	isActive           bool
	isCorporation      bool
	isExpired          bool // TODO: make dynamic
	isHistory          bool
	issuer             *app.EveEntity
	issuerName         string
	name               string
	objectID           int64
	ownerID            int64
	ownerName          string
	price              optional.Optional[float64]
	reward             optional.Optional[float64]
	startLocation      optional.Optional[*app.EveLocationShort]
	status             app.ContractStatus
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
		hasIssue:       o.HasIssue(),
		isActive:       o.Status.IsActive(),
		isCorporation:  false,
		isExpired:      o.IsExpired(),
		isHistory:      o.Status.IsHistory(),
		issuer:         o.IssuerEffective(),
		issuerName:     o.IssuerEffective().Name,
		name:           o.NameDisplay(),
		objectID:       o.ID,
		ownerID:        o.CharacterID,
		ownerName:      characterName(o.CharacterID),
		price:          o.Price,
		reward:         o.Reward,
		startLocation:  o.StartLocation,
		status:         o.Status,
		statusText:     o.Status.Display(),
		title:          o.Title.ValueOrFallback("-"),
		typeName:       o.Type.Display(),
		volume:         o.Volume,
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
	return r
}

func newContractRowForCorporation(o *app.CorporationContract, corporation *app.Corporation) contractRow {
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
		hasIssue:       o.HasIssue(),
		isActive:       o.Status.IsActive(),
		isCorporation:  true,
		isExpired:      o.IsExpired(),
		isHistory:      o.Status.IsHistory(),
		issuer:         o.IssuerEffective(),
		issuerName:     o.IssuerEffective().Name,
		name:           o.NameDisplay(),
		objectID:       o.ID,
		ownerID:        corporation.ID,
		ownerName:      corporation.NameOrZero(),
		price:          o.Price,
		reward:         o.Reward,
		startLocation:  o.StartLocation,
		status:         o.Status,
		statusText:     o.Status.Display(),
		title:          o.Title.ValueOrFallback("-"),
		typeName:       o.Type.Display(),
		volume:         o.Volume,
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
	return r
}

// Contracts is a UI element for showing Contracts.
// It either shows all character Contracts or the Contracts for a corporation.
type Contracts struct {
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
	u              baseUI
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

func NewContractsForCorporation(u baseUI) *Contracts {
	return newContracts(u, true)
}

func NewContractsForCharacters(u baseUI) *Contracts {
	return newContracts(u, false)
}

func newContracts(u baseUI, forCorporation bool) *Contracts {
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
		Width: ui.ColumnWidthDateTime,
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
	a := &Contracts{
		forCorporation: forCorporation,
		columnSorter:   xwidget.NewColumnSorter(columns, contractsColIssuedAt, xwidget.SortDesc),
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
			a.Update(ctx)
		})
		a.u.Signals().CorporationSectionChanged.AddListener(func(ctx context.Context, arg app.CorporationSectionUpdated) {
			if a.corporation.Load().IDOrZero() != arg.CorporationID {
				return
			}
			if arg.Section != app.SectionCorporationContracts {
				return
			}
			a.Update(ctx)
		})
	} else {
		a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
			if arg.Section == app.SectionCharacterContracts {
				a.Update(ctx)
			}
		})
		a.u.Signals().CharacterAdded.AddListener(func(ctx context.Context, _ *app.Character) {
			a.Update(ctx)
		})
		a.u.Signals().CharacterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
			a.Update(ctx)
		})
		a.u.Signals().TagsChanged.AddListener(func(ctx context.Context, _ struct{}) {
			a.Update(ctx)
		})
	}
	return a
}

func (a *Contracts) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHBox(
		a.selectType,
		a.selectIssuer,
		a.selectAssignee,
		a.selectStatus,
	)
	if !a.forCorporation {
		filter.Add(a.selectTag)
	}
	if a.u.IsMobile() {
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
			ShowCorporationContract(a.u, r)
		} else {
			ShowCharacterContract(a.u, r)
		}
	}
	return l
}

func (a *Contracts) filterRowsAsync(sortCol int) {
	totalRows := len(a.rows)
	rows := slices.Clone(a.rows)
	issuer := a.selectIssuer.Selected
	assignee := a.selectAssignee.Selected
	et := a.selectType.Selected
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
		if et != "" {
			rows = slices.DeleteFunc(rows, func(r contractRow) bool {
				return r.typeName != et
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

func (a *Contracts) Update(ctx context.Context) {
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
			a.footer.Text = fmt.Sprintf("ERROR: %s", a.u.ErrorDisplay(err))
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

func (a *Contracts) fetchRowsCorporation(ctx context.Context) ([]contractRow, int, error) {
	corporation := a.corporation.Load()
	if corporation == nil {
		return nil, 0, nil
	}
	oo, err := a.u.Corporation().ListCorporationContracts(ctx, corporation.ID)
	if err != nil {
		return nil, 0, err
	}
	var rows []contractRow
	var activeCount int
	for _, c := range oo {
		r := newContractRowForCorporation(c, corporation)
		rows = append(rows, r)
		if c.Status.IsActive() {
			activeCount++
		}
	}
	return rows, activeCount, nil
}

func (a *Contracts) fetchRowsOverview(ctx context.Context) ([]contractRow, int, error) {
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
	characters, err := a.u.Character().CharacterNames(ctx)
	if err != nil {
		return nil, 0, err
	}
	var rows []contractRow
	var activeCount int
	for _, c := range oo2 {
		r := newContractRowForCharacter(c, func(id int64) string {
			return characters[id]
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
