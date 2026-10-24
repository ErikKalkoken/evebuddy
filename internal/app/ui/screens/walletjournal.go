package screens

import (
	"cmp"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/filedialog"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"

	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xsync"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

const (
	walletJournalDirectionInflow  = "Inflow"
	walletJournalDirectionOutflow = "Outflow"
)

func walletJournalDirectionOptions() []string {
	return []string{walletJournalDirectionInflow, walletJournalDirectionOutflow}
}

// Names of the wallet journal filters, used as labels on desktop and as option names on mobile.
const (
	walletJournalFilterDirection = "Direction"
	walletJournalFilterType      = "Type"
)

// walletJournalFilter is the selected value of each wallet journal filter. Empty means not filtered.
type walletJournalFilter struct {
	direction string
	refType   string
}

// match reports whether row r passes all filters.
func (f walletJournalFilter) match(r walletJournalRow) bool {
	amount := r.amount.ValueOrZero()
	switch {
	case f.direction == walletJournalDirectionInflow && amount <= 0,
		f.direction == walletJournalDirectionOutflow && amount >= 0,
		f.refType != "" && r.refTypeDisplay != f.refType:
		return false
	}
	return true
}

type walletJournalRow struct {
	amount           optional.Optional[float64]
	amountDisplay    []widget.RichTextSegment
	amountFormatted  string
	balance          optional.Optional[float64]
	balanceFormatted string
	date             time.Time
	dateFormatted    string
	description      string
	division         app.Division
	firstParty       optional.Optional[*app.EveEntity]
	forCorporation   bool
	ownerID          int64
	ownerName        string
	reason           optional.Optional[string]
	refID            int64
	refType          string
	refTypeDisplay   string
	searchTarget     string
	secondParty      optional.Optional[*app.EveEntity]
	taxReceiver      optional.Optional[*app.EveEntity]
}

// setDisplayFields sets the fields derived from the row's data.
func (r *walletJournalRow) setDisplayFields() {
	formatISK := func(v float64) string {
		return humanize.FormatFloat(ui.FloatFormatISK, v)
	}
	r.amountFormatted = r.amount.StringFunc("?", formatISK)
	r.amountDisplay = xwidget.RichTextSegmentsFromText(
		r.amountFormatted,
		widget.RichTextStyle{
			Alignment: fyne.TextAlignTrailing,
			ColorName: colorISKAmount(r.amount),
		},
	)
	r.balanceFormatted = r.balance.StringFunc("?", formatISK)
	r.dateFormatted = r.date.Format(app.DateTimeFormat)
	r.setSearchTarget()
}

// setSearchTarget sets the text the search entry matches against.
func (r *walletJournalRow) setSearchTarget() {
	r.searchTarget = strings.ToLower(r.description + "\n" + r.reason.ValueOrZero()) // separator prevents matches across texts
}

func (r walletJournalRow) descriptionWithReason() string {
	if r.reason.IsEmpty() {
		return r.description
	}
	return fmt.Sprintf("[r] %s", r.description)
}

// WalletJournal is a widget for showing wallet journals for both characters and corporations.
type WalletJournal struct {
	widget.BaseWidget

	onBalanceUpdate func(balance optional.Optional[float64])

	balanceLabel     *widget.Label
	main             fyne.CanvasObject
	character        atomic.Pointer[app.Character]
	columnSorter     *xwidget.ColumnSorter[walletJournalRow]
	corporation      atomic.Pointer[app.Corporation]
	directionSelect  *kxwidget.FilterChipSelect // select chips only on desktop
	divisionID       atomic.Int64
	divisionSelect   *DivisionSelect
	errorLabel       *widget.Label
	filterChip       *xwidget.FilterChipCompact // only on mobile
	filterRun        xsync.LatestRun
	footerLabel      *widget.Label
	forCorporation   bool
	moreButton       *xwidget.ContextMenuButton
	rows             []walletJournalRow
	rowsFiltered     []walletJournalRow
	searchEntry      *xwidget.SearchEntry
	sortChip         *kxwidget.SortChip
	typeSelect       *kxwidget.FilterChipSelect
	u                baseUI
	updateRunBalance xsync.LatestRun
	updateRunJournal xsync.LatestRun
	updateRunNames   xsync.LatestRun
}

func NewCharacterWalletJournal(u baseUI) *WalletJournal {
	return newWalletJournal(u, false)
}

func NewCorporationWalletJournal(u baseUI) *WalletJournal {
	return newWalletJournal(u, true)
}

func newWalletJournal(u baseUI, forCorporation bool) *WalletJournal {
	columns := xwidget.NewDataColumns([]xwidget.DataColumn[walletJournalRow]{{
		Label: "Date",
		Width: 150,
		Sort: func(a, b walletJournalRow) int {
			return a.date.Compare(b.date)
		},
		Update: func(r walletJournalRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.dateFormatted)
		},
	}, {
		Label: "Type",
		Width: 150,
		Sort: func(a, b walletJournalRow) int {
			return strings.Compare(a.refType, b.refType)
		},
		Update: func(r walletJournalRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.refTypeDisplay)
		},
	}, {
		Label: "Amount",
		Width: 200,
		Sort: func(a, b walletJournalRow) int {
			return optional.Compare(a.amount, b.amount)
		},
		Update: func(r walletJournalRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).Set(r.amountDisplay)
		},
	}, {
		Label: "Balance",
		Width: 200,
		Update: func(r walletJournalRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.balance.StringFunc("?", func(v float64) string {
				return humanize.FormatFloat(ui.FloatFormatISK, v)
			}), widget.RichTextStyle{
				Alignment: fyne.TextAlignTrailing,
			},
			)
		},
	}, {
		Label: "Description",
		Width: 450,
		Update: func(r walletJournalRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.descriptionWithReason())
		},
	}})
	a := &WalletJournal{
		balanceLabel:   ui.NewLabelWithTruncation(""),
		columnSorter:   xwidget.NewColumnSorter(columns, "Date", xwidget.SortDesc),
		errorLabel:     ui.NewLabelWithWrapping(""),
		footerLabel:    ui.NewLabelWithTruncation(""),
		forCorporation: forCorporation,
		u:              u,
	}
	a.ExtendBaseWidget(a)

	a.balanceLabel.TextStyle.Bold = true

	a.errorLabel.Importance = widget.DangerImportance
	a.errorLabel.Hide()

	a.moreButton = xwidget.NewContextMenuButtonWithIcon(
		"",
		theme.MoreHorizontalIcon(),
		fyne.NewMenu("", fyne.NewMenuItem("Export transactions as CSV", a.exportAsCSV)),
	)
	a.moreButton.Importance = widget.LowImportance

	a.divisionSelect = NewDivisionSelect(func(d app.Division) {
		if a.divisionID.Load() == d.ID() {
			return
		}
		a.divisionID.Store(d.ID())
		a.clearData()
		ctx := context.Background()
		runAsync(func() {
			a.updateBalanceCorporation(ctx)
		})
		runAsync(func() {
			a.updateRowsCorporation(ctx)
		})
	})
	a.divisionID.Store(app.Division1.ID())

	if a.u.IsMobile() {
		a.main = a.makeDataList()
	} else {
		a.main = xwidget.MakeDataTable(
			columns,
			&a.rowsFiltered,
			func() fyne.CanvasObject {
				x := xwidget.NewRichText()
				x.Truncation = fyne.TextTruncateClip
				return x
			},
			a.columnSorter,
			a.filterRowsAsync,
			func(_ int, r walletJournalRow) {
				if a.forCorporation {
					ShowCorporationWalletJournalEntryWindowAsync(a.u, r.ownerID, r.ownerName, r.division, r.refID)
				} else {
					ShowCharacterWalletJournalEntryWindowAsync(a.u, r.ownerID, r.ownerName, r.refID)
				}
			},
		)
	}
	placeholder := "Search descriptions"
	if a.u.IsMobile() {
		placeholder = "Search" // shares the row with the chips
	}
	a.searchEntry = xwidget.NewSearchEntry(placeholder, func(_ string) {
		a.filterRowsAsync("")
	})

	if a.u.IsMobile() {
		a.filterChip = xwidget.NewFilterChipCompact(nil, func(map[string]string) {
			a.filterRowsAsync("")
		})
	} else {
		a.directionSelect = kxwidget.NewFilterChipSelect(walletJournalFilterDirection, walletJournalDirectionOptions(), func(string) {
			a.filterRowsAsync("")
		})
		a.directionSelect.SortDisabled = true
		a.typeSelect = kxwidget.NewFilterChipSelectWithSearch(walletJournalFilterType, []string{}, func(string) {
			a.filterRowsAsync("")
		}, a.u.MainWindow())
	}
	a.sortChip = a.columnSorter.NewSortChip(func() {
		a.filterRowsAsync("")
	})

	// signals
	if a.forCorporation {
		a.u.Signals().CurrentCorporationExchanged.AddListener(
			func(ctx context.Context, c *app.Corporation) {
				a.corporation.Store(c)
				a.divisionID.Store(app.Division1.ID())
				fyne.Do(func() {
					a.clearData()
					a.divisionSelect.SetDivisionSilent(app.Division1)
					a.resetFilters()
				})
				var wg sync.WaitGroup
				wg.Go(func() {
					a.updateBalanceCorporation(ctx)
				})
				wg.Go(func() {
					a.updateWalletNames(ctx)
				})
				wg.Go(func() {
					a.updateRowsCorporation(ctx)
				})
				wg.Wait()
			},
		)
		a.u.Signals().CorporationSectionChanged.AddListener(func(ctx context.Context, arg app.CorporationSectionUpdated) {
			if a.corporation.Load().IDOrZero() != arg.CorporationID {
				return
			}
			if arg.Section == app.CorporationSectionWalletJournal(app.Division(a.divisionID.Load())) {
				a.updateRowsCorporation(ctx)
			}
			if arg.Section == app.SectionCorporationDivisions {
				a.updateWalletNames(ctx)
			}
			if arg.Section == app.SectionCorporationWalletBalances {
				a.updateBalanceCorporation(ctx)
			}
		})
	} else {
		a.u.Signals().CurrentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
			a.character.Store(c)
			fyne.Do(func() {
				a.resetFilters()
			})
			var wg sync.WaitGroup
			wg.Go(func() {
				a.updateBalanceCharacter(ctx)
			})
			wg.Go(func() {
				a.updateRowsCharacter(ctx)
			})
			wg.Wait()
		})
		a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
			if a.character.Load().IDOrZero() != arg.CharacterID {
				return
			}
			switch arg.Section {
			case app.SectionCharacterWalletBalance:
				a.updateBalanceCharacter(ctx)
			case app.SectionCharacterWalletJournal:
				a.updateRowsCharacter(ctx)
			}
		})
	}
	return a
}

func (a *WalletJournal) CreateRenderer() fyne.WidgetRenderer {
	var top fyne.CanvasObject
	if a.u.IsMobile() {
		var top2 fyne.CanvasObject
		if a.forCorporation {
			top2 = container.NewVBox(a.divisionSelect, a.balanceLabel)
		} else {
			top2 = a.balanceLabel
		}
		top = container.NewBorder(
			top2,
			nil,
			nil,
			container.NewHBox(a.filterChip, a.sortChip, a.moreButton),
			a.searchEntry,
		)
	} else {
		var top2 fyne.CanvasObject
		if a.forCorporation {
			a.balanceLabel.Alignment = fyne.TextAlignTrailing
			top2 = container.NewBorder(nil, nil, a.divisionSelect, nil, a.balanceLabel)
		} else {
			top2 = a.balanceLabel
		}
		filter := container.NewHBox(a.directionSelect, a.typeSelect)
		top = container.NewBorder(
			top2,
			nil,
			filter,
			a.moreButton,
			a.searchEntry,
		)
	}
	c := container.NewBorder(
		top,
		container.NewVBox(a.errorLabel, a.footerLabel),
		nil,
		nil,
		a.main,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *WalletJournal) clearData() {
	clearLabel := func(l *widget.Label) {
		l.Text = ""
		l.Importance = widget.MediumImportance
		l.Refresh()
	}
	clearLabel(a.balanceLabel)
	clearLabel(a.footerLabel)
	a.rows = nil
	a.rowsFiltered = nil
	a.main.Refresh()
}

// currentFilter returns the selected filters: from the compact chip on mobile
// and from the filter chip on desktop.
func (a *WalletJournal) currentFilter() walletJournalFilter {
	if a.filterChip != nil {
		s := a.filterChip.Selected()
		return walletJournalFilter{
			direction: s[walletJournalFilterDirection],
			refType:   s[walletJournalFilterType],
		}
	}
	return walletJournalFilter{
		direction: a.directionSelect.Selected,
		refType:   a.typeSelect.Selected,
	}
}

// resetFilters clears search and filters without filtering again.
func (a *WalletJournal) resetFilters() {
	a.searchEntry.ClearSilent()
	if a.filterChip != nil {
		a.filterChip.ResetSilent()
		return
	}
	clearSelectsSilent(a.directionSelect, a.typeSelect)
}

func (a *WalletJournal) makeDataList() *xwidget.StripedList {
	p := theme.Padding()
	l := xwidget.NewStripedList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			date := widget.NewLabel("Template")
			date.Truncation = fyne.TextTruncateClip
			balance := widget.NewLabel("Template")
			balance.Alignment = fyne.TextAlignTrailing
			refType := widget.NewLabel("Template")
			refType.Truncation = fyne.TextTruncateClip
			value := widget.NewLabel("Template")
			value.Alignment = fyne.TextAlignTrailing
			description := widget.NewLabel("Template")
			description.Truncation = fyne.TextTruncateClip
			return container.New(layout.NewCustomPaddedVBoxLayout(-p),
				container.NewBorder(nil, nil, nil, value, date),
				container.NewBorder(nil, nil, nil, balance, refType),
				description,
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id < 0 || id >= len(a.rowsFiltered) {
				return
			}
			r := a.rowsFiltered[id]
			c := co.(*fyne.Container).Objects

			b0 := c[0].(*fyne.Container).Objects
			b0[0].(*widget.Label).SetText(r.dateFormatted)
			amount := b0[1].(*widget.Label)
			amount.Text = r.amountFormatted
			amount.Importance = importanceISKAmount(r.amount)
			amount.Refresh()

			b1 := c[1].(*fyne.Container).Objects
			b1[0].(*widget.Label).SetText(r.refTypeDisplay)
			b1[1].(*widget.Label).SetText(r.balanceFormatted)

			c[2].(*widget.Label).SetText(r.description)
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll()
		if id < 0 || id >= len(a.rowsFiltered) {
			return
		}
		r := a.rowsFiltered[id]
		if a.forCorporation {
			ShowCorporationWalletJournalEntryWindowAsync(a.u, r.ownerID, r.ownerName, r.division, r.refID)
		} else {
			ShowCharacterWalletJournalEntryWindowAsync(a.u, r.ownerID, r.ownerName, r.refID)
		}
	}
	l.HideSeparators = true
	return l
}

func (a *WalletJournal) exportAsCSV() {
	var ids []int64
	if a.forCorporation {
		ids = append(ids, a.corporation.Load().IDOrZero())
		ids = append(ids, int64(a.divisionID.Load()))
	} else {
		ids = append(ids, a.character.Load().IDOrZero())
	}
	filename := filedialog.MakeFilename("walletjournal", ".csv", ids...)
	exportRowsAsCSV(a.u, "walletjournal", filename, a.rowsFiltered, func(w io.Writer, rows []walletJournalRow) error {
		return writeJournalRowsToCSV(w, rows)
	})
}

func (a *WalletJournal) filterRowsAsync(sortCol string) {
	isLatest := a.filterRun.Start()
	totalRows := len(a.rows)
	rows := slices.Clone(a.rows)
	filter := a.currentFilter()
	search := strings.ToLower(a.searchEntry.Text)
	sortCol, dir, doSort := a.columnSorter.CalcSort(sortCol)

	runAsync(func() {
		rows = slices.DeleteFunc(rows, func(r walletJournalRow) bool {
			return !filter.match(r)
		})
		if len(search) > 1 {
			rows = slices.DeleteFunc(rows, func(r walletJournalRow) bool {
				return !strings.Contains(r.searchTarget, search)
			})
		}
		a.columnSorter.SortRows(rows, sortCol, dir, doSort)
		typeOptions := xslices.Map(rows, func(r walletJournalRow) string {
			return r.refTypeDisplay
		})
		footer := fmt.Sprintf("Showing %s / %s entries", ihumanize.Comma(len(rows)), ihumanize.Comma(totalRows))

		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.footerLabel.Text = footer
			a.footerLabel.Importance = widget.MediumImportance
			a.footerLabel.Refresh()
			if a.filterChip != nil {
				a.filterChip.SetOptions(
					xwidget.NewFilterOptionMultiChoiceOrdered(walletJournalFilterDirection, walletJournalDirectionOptions()),
					xwidget.NewFilterOptionMultiChoiceWithSearch(walletJournalFilterType, typeOptions),
				)
			} else {
				a.typeSelect.SetOptions(typeOptions)
			}
			a.rowsFiltered = rows
			a.main.Refresh()
		})
	})
}

func (a *WalletJournal) updateBalanceCharacter(ctx context.Context) {
	a.updateBalance(ctx, a.character.Load().IDOrZero(), a.fetchBalanceCharacter)
}

func (a *WalletJournal) updateBalanceCorporation(ctx context.Context) {
	a.updateBalance(ctx, a.corporation.Load().IDOrZero(), a.fetchBalanceCorporation)
}

func (a *WalletJournal) updateBalance(ctx context.Context, ownerID int64, fetch func(context.Context, int64) (optional.Optional[float64], error)) {
	isLatest := a.updateRunBalance.Start()
	setBalance := func(balance optional.Optional[float64], s string, i widget.Importance) {
		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.balanceLabel.Text, a.balanceLabel.Importance = s, i
			a.balanceLabel.Refresh()
			if a.onBalanceUpdate != nil {
				a.onBalanceUpdate(balance)
			}
		})
	}
	if ownerID == 0 {
		setBalance(optional.Optional[float64]{}, "", widget.MediumImportance)
		return
	}
	balance, err := fetch(ctx, ownerID)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		slog.Error("Failed to update wallet balance UI", "forCorporation", a.forCorporation, "ownerID", ownerID, "err", err)
		setBalance(optional.Optional[float64]{}, "ERROR: "+a.u.ErrorDisplay(err), widget.DangerImportance)
		return
	}
	v, ok := balance.Value()
	if !ok {
		setBalance(balance, "No data", widget.WarningImportance)
		return
	}
	setBalance(balance, humanize.FormatFloat(ui.FloatFormatISK, v)+" ISK", widget.MediumImportance)
}

func (a *WalletJournal) fetchBalanceCharacter(ctx context.Context, characterID int64) (optional.Optional[float64], error) {
	hasData, err := a.u.Character().HasSection(ctx, characterID, app.SectionCharacterWalletBalance)
	if err != nil || !hasData {
		return optional.Optional[float64]{}, err
	}
	c, err := a.u.Character().GetCharacter(ctx, characterID)
	if errors.Is(err, app.ErrNotFound) {
		return optional.Optional[float64]{}, nil
	}
	if err != nil {
		return optional.Optional[float64]{}, err
	}
	return c.WalletBalance, nil
}

func (a *WalletJournal) fetchBalanceCorporation(ctx context.Context, corporationID int64) (optional.Optional[float64], error) {
	hasData, err := a.u.Corporation().HasSection(ctx, corporationID, app.SectionCorporationWalletBalances)
	if err != nil || !hasData {
		return optional.Optional[float64]{}, err
	}
	balance, err := a.u.Corporation().GetWalletBalance(ctx, corporationID, app.Division(a.divisionID.Load()))
	if errors.Is(err, app.ErrNotFound) {
		return optional.Optional[float64]{}, nil
	}
	if err != nil {
		return optional.Optional[float64]{}, err
	}
	return optional.New(balance), nil
}

func (a *WalletJournal) updateRowsCharacter(ctx context.Context) {
	c := a.character.Load()
	a.updateRows(ctx, c != nil, func(ctx context.Context) ([]walletJournalRow, bool, error) {
		return a.fetchRowsCharacter(ctx, c)
	})
}

func (a *WalletJournal) updateRowsCorporation(ctx context.Context) {
	c := a.corporation.Load()
	division := app.Division(a.divisionID.Load())
	a.updateRows(ctx, c != nil, func(ctx context.Context) ([]walletJournalRow, bool, error) {
		return a.fetchRowsCorporation(ctx, c, division)
	})
}

func (a *WalletJournal) updateRows(ctx context.Context, hasOwner bool, fetch func(context.Context) ([]walletJournalRow, bool, error)) {
	isLatest := a.updateRunJournal.Start()
	setInfo := func(s string, i widget.Importance) {
		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.footerLabel.Text, a.footerLabel.Importance = s, i
			a.footerLabel.Refresh()
		})
	}
	setError := func(err error) {
		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.errorLabel.SetText("ERROR: " + a.u.ErrorDisplay(err))
			a.errorLabel.Show()
			a.Refresh()
		})
	}
	reset := func() {
		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.filterRun.Discard()
			a.errorLabel.Hide()
			xslices.Clear(&a.rows)
			xslices.Clear(&a.rowsFiltered)
			a.footerLabel.SetText("")
			a.main.Refresh()
		})
	}
	if !hasOwner {
		reset()
		var s string
		if a.forCorporation {
			s = "No corporation"
		} else {
			s = "No character"
		}
		setInfo(s, widget.LowImportance)
		return
	}
	rows, hasData, err := fetch(ctx)
	if err != nil {
		reset()
		setError(err)
		return
	}
	if !hasData {
		reset()
		setInfo("No data", widget.WarningImportance)
		return
	}
	fyne.Do(func() {
		if !isLatest() {
			return
		}
		a.errorLabel.Hide()
		a.rows = rows
		a.filterRowsAsync("")
	})
}

func (a *WalletJournal) fetchRowsCharacter(ctx context.Context, character *app.Character) ([]walletJournalRow, bool, error) {
	hasData, err := a.u.Character().HasSection(ctx, character.ID, app.SectionCharacterWalletJournal)
	if err != nil || !hasData {
		return nil, false, err
	}
	oo, err := a.u.Character().ListWalletJournalEntries(ctx, character.ID)
	if err != nil {
		return nil, false, err
	}
	rows := make([]walletJournalRow, 0, len(oo))
	for _, o := range oo {
		r := walletJournalRow{
			amount:         o.Amount,
			balance:        o.Balance,
			ownerID:        character.ID,
			ownerName:      character.NameOrZero(),
			date:           o.Date,
			description:    o.Description,
			reason:         o.Reason,
			refID:          o.RefID,
			refType:        o.RefType,
			refTypeDisplay: o.RefTypeDisplay(),
			firstParty:     o.FirstParty,
			secondParty:    o.SecondParty,
			taxReceiver:    o.TaxReceiver,
		}
		r.setDisplayFields()
		rows = append(rows, r)
	}
	return rows, true, nil
}

func (a *WalletJournal) fetchRowsCorporation(ctx context.Context, corporation *app.Corporation, division app.Division) ([]walletJournalRow, bool, error) {
	hasData, err := a.u.Corporation().HasSection(ctx, corporation.ID, app.CorporationSectionWalletJournal(division))
	if err != nil || !hasData {
		return nil, false, err
	}
	oo, err := a.u.Corporation().ListWalletJournalEntries(ctx, corporation.ID, division)
	if err != nil {
		return nil, false, err
	}
	rows := make([]walletJournalRow, 0, len(oo))
	for _, o := range oo {
		r := walletJournalRow{
			amount:         o.Amount,
			balance:        o.Balance,
			ownerID:        corporation.ID,
			ownerName:      corporation.NameOrZero(),
			forCorporation: true,
			division:       division,
			date:           o.Date,
			description:    o.Description,
			reason:         o.Reason,
			refID:          o.RefID,
			refType:        o.RefType,
			refTypeDisplay: o.RefTypeDisplay(),
			firstParty:     o.FirstParty,
			secondParty:    o.SecondParty,
			taxReceiver:    o.TaxReceiver,
		}
		r.setDisplayFields()
		rows = append(rows, r)
	}
	return rows, true, nil
}

func (a *WalletJournal) updateWalletNames(ctx context.Context) {
	isLatest := a.updateRunNames.Start()
	c := a.corporation.Load()
	if c == nil {
		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.divisionSelect.SetNames(nil)
		})
		return
	}
	names := a.u.Corporation().ListWalletNames(ctx, c.ID)
	fyne.Do(func() {
		if !isLatest() {
			return
		}
		a.divisionSelect.SetNames(names)
	})
}

func writeJournalRowsToCSV(w io.Writer, rows []walletJournalRow) error {
	cw := csv.NewWriter(w)
	header := []string{
		"ID",
		"Date",
		"Type",
		"Amount",
		"Balance",
		"Description",
		"First Party ID",
		"First Party Name",
		"Second Party ID",
		"Second Party Name",
		"Tax Receiver ID",
		"Tax Receiver Name",
		"Reason",
	}
	if err := cw.Write(header); err != nil {
		return err
	}
	for _, r := range rows {
		var amount string
		if v, ok := r.amount.Value(); ok {
			amount = strconv.FormatFloat(v, 'f', -1, 64)
		}
		var balance string
		if v, ok := r.balance.Value(); ok {
			balance = strconv.FormatFloat(v, 'f', -1, 64)
		}
		var firstPartyID, secondPartyID, taxReceiverID int64
		var firstPartyName, secondPartyName, taxReceiverName string
		f := func(o optional.Optional[*app.EveEntity]) (int64, string) {
			v, ok := o.Value()
			if !ok {
				return 0, ""
			}
			return v.ID, v.Name
		}
		firstPartyID, firstPartyName = f(r.firstParty)
		secondPartyID, secondPartyName = f(r.secondParty)
		taxReceiverID, taxReceiverName = f(r.taxReceiver)
		record := []string{
			strconv.FormatInt(r.refID, 10),
			r.date.Format(time.RFC3339),
			r.refType,
			amount,
			balance,
			r.description,
			strconv.FormatInt(firstPartyID, 10),
			firstPartyName,
			strconv.FormatInt(secondPartyID, 10),
			secondPartyName,
			strconv.FormatInt(taxReceiverID, 10),
			taxReceiverName,
			r.reason.ValueOrZero(),
		}
		if err := cw.Write(record); err != nil {
			return err
		}

	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		return err
	}
	return nil
}

// DivisionSelect is a select widget for corporation wallet divisions.
// A division is always selected; the default is the master wallet.
type DivisionSelect struct {
	widget.Select

	names            map[app.Division]string
	selectedDivision app.Division
}

// NewDivisionSelect returns a new DivisionSelect with the default wallet names.
// changed is called with the division the user selects and may be nil.
func NewDivisionSelect(changed func(d app.Division)) *DivisionSelect {
	const defaultDivision = app.Division1
	w := &DivisionSelect{
		selectedDivision: defaultDivision,
	}
	w.ExtendBaseWidget(w)
	w.OnChanged = func(s string) {
		for _, d := range app.Divisions {
			if corporationWalletLabel(d, w.names[d]) == s {
				w.selectedDivision = d
				if changed != nil {
					changed(d)
				}
				break
			}
		}
	}
	w.setNames(nil)
	return w
}

// SetDivisionSilent selects division d without calling changed.
func (w *DivisionSelect) SetDivisionSilent(d app.Division) {
	w.setDivisionSilent(d)
	w.Refresh()
}

func (w *DivisionSelect) setDivisionSilent(d app.Division) {
	w.Selected = corporationWalletLabel(d, w.names[d])
	w.selectedDivision = d
}

// SetNames sets the wallet names, keeping the selected division.
// Divisions missing from names are shown with their default name.
// Passing in a nil map resets all names to their defaults.
func (w *DivisionSelect) SetNames(names map[app.Division]string) {
	w.setNames(names)
	w.Refresh()
}

func (w *DivisionSelect) setNames(names map[app.Division]string) {
	w.names = names

	var options []string
	for _, d := range app.Divisions {
		options = append(options, corporationWalletLabel(d, names[d]))
	}
	// Select sizes itself by the placeholder, which is never shown since a wallet is always selected.
	w.PlaceHolder = slices.MaxFunc(options, func(x, y string) int {
		return cmp.Compare(utf8.RuneCountInString(x), utf8.RuneCountInString(y))
	})
	w.Options = options
	w.setDivisionSilent(w.selectedDivision)
}

func colorISKAmount(amount optional.Optional[float64]) fyne.ThemeColorName {
	var color fyne.ThemeColorName
	if v, ok := amount.Value(); !ok {
		color = theme.ColorNameDisabled
	} else if v < 0 {
		color = theme.ColorNameError
	} else if v > 0 {
		color = theme.ColorNameSuccess
	} else {
		color = theme.ColorNameForeground
	}
	return color
}

func importanceISKAmount(amount optional.Optional[float64]) widget.Importance {
	if v, ok := amount.Value(); !ok {
		return widget.LowImportance
	} else if v > 0 {
		return widget.SuccessImportance
	} else if v < 0 {
		return widget.DangerImportance
	}
	return widget.MediumImportance
}
