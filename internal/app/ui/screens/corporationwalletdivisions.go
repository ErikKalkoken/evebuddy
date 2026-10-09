package screens

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type corporationWalletRow struct {
	balance  float64
	division app.Division
	isTotal  bool
	name     string
}

func (r corporationWalletRow) balanceDisplay() string {
	return humanize.FormatFloat(ui.FloatFormatISK, r.balance)
}

func (r corporationWalletRow) divisionDisplay() string {
	if r.isTotal {
		return ""
	}
	return fmt.Sprint(r.division)
}

// label returns the name with the wallet number, e.g. "Master Wallet [1]".
func (r corporationWalletRow) label() string {
	if r.isTotal {
		return r.name
	}
	return corporationWalletLabel(r.division, r.name)
}

// CorporationWalletDivisions shows the balances of all wallets of the current corporation.
type CorporationWalletDivisions struct {
	widget.BaseWidget

	body         fyne.CanvasObject
	columnSorter *xwidget.ColumnSorter[corporationWalletRow]
	corporation  atomic.Pointer[app.Corporation]
	rows         []corporationWalletRow
	rowsSorted   []corporationWalletRow // sorted rows plus total row
	status       *widget.Label
	u            baseUI
	updateRun    latestRun
}

func NewCorporationWalletDivisions(u baseUI) *CorporationWalletDivisions {
	columns := xwidget.NewDataColumns([]xwidget.DataColumn[corporationWalletRow]{{
		Label: "#",
		Width: 50,
		Sort: func(a, b corporationWalletRow) int {
			return cmp.Compare(a.division, b.division)
		},
		Update: func(r corporationWalletRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.divisionDisplay(), widget.RichTextStyle{
				Alignment: fyne.TextAlignTrailing,
			})
		},
	}, {
		Label: "Name",
		Width: 250,
		Sort: func(a, b corporationWalletRow) int {
			return strings.Compare(a.name, b.name)
		},
		Update: func(r corporationWalletRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.name, widget.RichTextStyle{
				TextStyle: fyne.TextStyle{Bold: r.isTotal},
			})
		},
	}, {
		Label: "Balance",
		Width: 200,
		Sort: func(a, b corporationWalletRow) int {
			return cmp.Compare(a.balance, b.balance)
		},
		Update: func(r corporationWalletRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.balanceDisplay(), widget.RichTextStyle{
				Alignment: fyne.TextAlignTrailing,
				TextStyle: fyne.TextStyle{Bold: r.isTotal},
			})
		},
	}})
	a := &CorporationWalletDivisions{
		columnSorter: xwidget.NewColumnSorter(columns, "#", xwidget.SortAsc),
		status:       widget.NewLabel(""),
		u:            u,
	}
	a.ExtendBaseWidget(a)
	a.status.Hide()
	if a.u.IsMobile() {
		a.body = a.makeList()
	} else {
		a.body = xwidget.MakeDataTable(
			columns,
			&a.rowsSorted,
			func() fyne.CanvasObject {
				x := xwidget.NewRichText()
				x.Truncation = fyne.TextTruncateClip
				return x
			},
			a.columnSorter,
			a.sortRows,
			nil,
		)
	}

	a.u.Signals().CurrentCorporationExchanged.AddListener(func(ctx context.Context, c *app.Corporation) {
		a.corporation.Store(c)
		a.Update(ctx)
	})
	a.u.Signals().CorporationSectionChanged.AddListener(func(ctx context.Context, arg app.CorporationSectionUpdated) {
		if a.corporation.Load().IDOrZero() != arg.CorporationID {
			return
		}
		switch arg.Section {
		case app.SectionCorporationWalletBalances, app.SectionCorporationDivisions:
			a.Update(ctx)
		}
	})
	return a
}

func (a *CorporationWalletDivisions) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(a.status, nil, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *CorporationWalletDivisions) makeList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.rowsSorted)
		},
		func() fyne.CanvasObject {
			balance := widget.NewLabel("")
			balance.Alignment = fyne.TextAlignTrailing
			return container.NewBorder(nil, nil, nil, balance, ui.NewLabelWithTruncation(""))
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.rowsSorted) {
				return
			}
			r := a.rowsSorted[id]
			c := co.(*fyne.Container)
			for i, s := range []string{r.label(), r.balanceDisplay()} {
				l := c.Objects[i].(*widget.Label)
				l.Text = s
				l.TextStyle.Bold = r.isTotal
				l.Refresh()
			}
		},
	)
	l.OnSelected = func(_ widget.ListItemID) {
		l.UnselectAll()
	}
	return l
}

// Update reloads the wallet names and balances.
func (a *CorporationWalletDivisions) Update(ctx context.Context) {
	isLatest := a.updateRun.start()
	data := fetchCorporationWalletsData(ctx, a.u, a.corporation.Load().IDOrZero())
	if ctx.Err() != nil {
		return
	}
	fyne.Do(func() {
		if !isLatest() {
			return
		}
		a.set(data)
	})
}

func (a *CorporationWalletDivisions) set(data corporationWalletsData) {
	var rows []corporationWalletRow
	if data.status == "" && data.balances != nil {
		for _, d := range app.Divisions {
			name := data.names[d]
			if name == "" {
				name = d.DefaultWalletName()
			}
			rows = append(rows, corporationWalletRow{
				balance:  data.balances[d],
				division: d,
				name:     name,
			})
		}
	}
	a.rows = rows
	if data.status != "" {
		a.status.Text, a.status.Importance = data.status, data.importance
		a.status.Refresh()
		a.status.Show()
	} else {
		a.status.Hide()
	}
	a.sortRows("")
}

// sortRows sorts the rows by column sortCol and appends the total row.
// An empty sortCol keeps the current sort.
func (a *CorporationWalletDivisions) sortRows(sortCol string) {
	rows := slices.Clone(a.rows)
	sortCol, dir, doSort := a.columnSorter.CalcSort(sortCol)
	a.columnSorter.SortRows(rows, sortCol, dir, doSort)
	if len(rows) > 0 {
		var total float64
		for _, r := range rows {
			total += r.balance
		}
		rows = append(rows, corporationWalletRow{balance: total, isTotal: true, name: "Total"})
	}
	a.rowsSorted = rows
	a.body.Refresh()
}
