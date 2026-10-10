package screens

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sync/atomic"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// corporationWalletsData is the loaded state for a [CorporationWallets] screen.
type corporationWalletsData struct {
	balances   map[app.Division]float64 // empty when there is no data
	importance widget.Importance
	names      map[app.Division]string
	status     string                     // shown instead of balances when not empty
	total      optional.Optional[float64] // empty when there is no data
}

// CorporationWallets shows the wallets of the current corporation.
type CorporationWallets struct {
	widget.BaseWidget

	// OnBalanceUpdate is called on the main thread with the total balance after each update.
	OnBalanceUpdate func(total optional.Optional[float64])

	balanceLabel      *widget.Label
	corporation       atomic.Pointer[app.Corporation]
	data              corporationWalletsData
	divisions         *CorporationWalletDivisions
	journals          map[app.Division]*WalletJournal
	journalStack      *fyne.Container
	selected          app.Division
	walletSelect      *widget.Select
	transactions      map[app.Division]*WalletTransactions
	transactionsStack *fyne.Container
	u                 baseUI
	updateRun         latestRun
}

func NewCorporationWallets(u baseUI) *CorporationWallets {
	a := &CorporationWallets{
		balanceLabel: xwidget.NewLabelWithSelection(""),
		divisions:    NewCorporationWalletDivisions(u),
		journals:     make(map[app.Division]*WalletJournal),
		selected:     app.Division1,
		transactions: make(map[app.Division]*WalletTransactions),
		u:            u,
	}
	a.ExtendBaseWidget(a)

	a.balanceLabel.TextStyle.Bold = true
	a.journalStack = container.NewStack()
	a.transactionsStack = container.NewStack()
	for _, d := range app.Divisions {
		a.journals[d] = NewCorporationWalletJournal(u, d)
		a.transactions[d] = NewCorporationWalletTransactions(u, d)
		a.journalStack.Add(a.journals[d])
		a.transactionsStack.Add(a.transactions[d])
	}

	a.data = corporationWalletsData{names: defaultCorporationWalletNames()}
	a.walletSelect = widget.NewSelect(nil, func(s string) {
		for d, n := range a.data.names {
			if corporationWalletLabel(d, n) == s {
				a.selectDivision(d)
				return
			}
		}
	})
	a.applyData(a.data)
	a.selectDivision(app.Division1)

	a.u.Signals().CurrentCorporationExchanged.AddListener(func(ctx context.Context, c *app.Corporation) {
		a.corporation.Store(c)
		fyne.Do(func() {
			a.selectDivision(app.Division1)
		})
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

func (a *CorporationWallets) CreateRenderer() fyne.WidgetRenderer {
	var header fyne.CanvasObject
	if a.u.IsMobile() {
		header = container.NewVBox(a.walletSelect, a.balanceLabel)
	} else {
		a.balanceLabel.Alignment = fyne.TextAlignTrailing
		header = container.NewBorder(nil, nil, a.walletSelect, a.balanceLabel)
	}
	c := container.NewBorder(
		header,
		nil,
		nil,
		nil,
		container.NewAppTabs(
			ui.NewTabItem("Transactions", a.journalStack),
			ui.NewTabItem("Market", a.transactionsStack),
			ui.NewTabItem("Divisions", a.divisions),
		),
	)
	return widget.NewSimpleRenderer(c)
}

// Update reloads wallet names and balances.
// The journals, transactions and divisions update themselves.
func (a *CorporationWallets) Update(ctx context.Context) {
	isLatest := a.updateRun.start()
	data := fetchCorporationWalletsData(ctx, a.u, a.corporation.Load().IDOrZero())
	if ctx.Err() != nil {
		return
	}
	fyne.Do(func() {
		if !isLatest() {
			return
		}
		a.applyData(data)
		if a.OnBalanceUpdate != nil {
			a.OnBalanceUpdate(data.total)
		}
	})
}

// fetchCorporationWalletsData loads wallet names and balances for a corporation.
func fetchCorporationWalletsData(ctx context.Context, u baseUI, corporationID int64) corporationWalletsData {
	if corporationID == 0 {
		return corporationWalletsData{names: defaultCorporationWalletNames()}
	}
	data := corporationWalletsData{
		names: u.Corporation().ListWalletNames(ctx, corporationID),
	}
	setError := func(err error) corporationWalletsData {
		if ctx.Err() != nil {
			return data
		}
		slog.Error("Failed to update corp wallets UI", "corporationID", corporationID, "err", err)
		data.status, data.importance = "Error: "+u.ErrorDisplay(err), widget.DangerImportance
		return data
	}
	hasRole, err := u.Corporation().PermittedSection(ctx, corporationID, app.SectionCorporationWalletBalances)
	if err != nil {
		return setError(err)
	}
	if !hasRole {
		data.status, data.importance = "No permission", widget.WarningImportance
		return data
	}
	hasData, err := u.Corporation().HasSection(ctx, corporationID, app.SectionCorporationWalletBalances)
	if err != nil {
		return setError(err)
	}
	if !hasData {
		data.status, data.importance = "No data", widget.WarningImportance
		return data
	}
	oo, err := u.Corporation().ListWalletBalances(ctx, corporationID)
	if err != nil {
		return setError(err)
	}
	if len(oo) == 0 {
		data.status, data.importance = "No data", widget.WarningImportance
		return data
	}
	data.balances = make(map[app.Division]float64)
	for _, o := range oo {
		data.balances[app.Division(o.DivisionID)] = o.Balance
		data.total = optional.SumNonEmpty(data.total, optional.New(o.Balance))
	}
	return data
}

func (a *CorporationWallets) applyData(data corporationWalletsData) {
	a.data = data

	var options []string
	for _, d := range app.Divisions {
		options = append(options, corporationWalletLabel(d, data.names[d]))
	}
	// Select sizes itself by the placeholder, which is never shown since a wallet is always selected.
	a.walletSelect.PlaceHolder = slices.MaxFunc(options, func(x, y string) int {
		return cmp.Compare(utf8.RuneCountInString(x), utf8.RuneCountInString(y))
	})
	a.walletSelect.SetOptions(options)
	a.refreshSelected()
}

// selectDivision shows the wallet for division d.
func (a *CorporationWallets) selectDivision(d app.Division) {
	a.selected = d
	for _, x := range app.Divisions {
		if x == d {
			a.journals[x].Show()
			a.transactions[x].Show()
		} else {
			a.journals[x].Hide()
			a.transactions[x].Hide()
		}
	}
	a.journalStack.Refresh()
	a.transactionsStack.Refresh()
	a.refreshSelected()
}

// refreshSelected updates the selector label and balance for the selected division.
func (a *CorporationWallets) refreshSelected() {
	// Setting the field directly avoids triggering OnChanged.
	a.walletSelect.Selected = corporationWalletLabel(a.selected, a.data.names[a.selected])
	a.walletSelect.Refresh()

	switch {
	case a.corporation.Load() == nil:
		a.balanceLabel.Text, a.balanceLabel.Importance = "", widget.MediumImportance
	case a.data.status != "":
		a.balanceLabel.Text, a.balanceLabel.Importance = a.data.status, a.data.importance
	default:
		a.balanceLabel.Text = humanize.FormatFloat(ui.FloatFormatISK, a.data.balances[a.selected]) + " ISK"
		a.balanceLabel.Importance = widget.MediumImportance
	}
	a.balanceLabel.Refresh()
}

// corporationWalletLabel returns the display label for a wallet, e.g. "Master Wallet [1]".
func corporationWalletLabel(d app.Division, name string) string {
	if name == "" {
		name = d.DefaultWalletName()
	}
	return fmt.Sprintf("%s [%d]", name, d)
}

func defaultCorporationWalletNames() map[app.Division]string {
	m := make(map[app.Division]string)
	for _, d := range app.Divisions {
		m[d] = d.DefaultWalletName()
	}
	return m
}
