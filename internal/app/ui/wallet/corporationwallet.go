package wallet

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type CorporationWallet struct {
	widget.BaseWidget

	OnBalanceUpdate func(balance optional.Optional[float64])
	NnNameUpdate    func(name string)
	OnTopUpdate     func(top string)

	balance      *widget.Label
	corporation  atomic.Pointer[app.Corporation]
	division     app.Division
	journal      *WalletJournal
	name         *widget.Label
	transactions *WalletTransactions
	u            ui
}

func NewCorporationWallet(u ui, division app.Division) *CorporationWallet {
	a := &CorporationWallet{
		balance:      xwidget.NewLabelWithSelection(""),
		division:     division,
		journal:      NewCorporationWalletJournal(u, division),
		name:         widget.NewLabel(""),
		transactions: NewCorporationWalletTransactions(u, division),
		u:            u,
	}
	a.name.TextStyle.Italic = true
	a.ExtendBaseWidget(a)
	a.u.Signals().CurrentCorporationExchanged.AddListener(func(ctx context.Context, c *app.Corporation) {
		a.corporation.Store(c)
		a.Update(ctx)
	})
	a.u.Signals().CorporationSectionChanged.AddListener(func(ctx context.Context, arg app.CorporationSectionUpdated) {
		if a.corporation.Load().IDOrZero() != arg.CorporationID {
			return
		}
		switch arg.Section {
		case app.SectionCorporationWalletBalances:
			a.updateBalance(ctx)
		case app.SectionCorporationDivisions:
			a.updateName(ctx)
		}
	})
	return a
}

func (a *CorporationWallet) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		container.NewHBox(a.balance, a.name),
		nil,
		nil,
		nil,
		container.NewAppTabs(
			container.NewTabItem("Transactions", a.journal),
			container.NewTabItem("Market Transactions", a.transactions),
		),
	)
	return widget.NewSimpleRenderer(c)
}

func (a *CorporationWallet) Update(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Go(func() {
		a.journal.Update(ctx)
	})
	wg.Go(func() {
		a.transactions.Update(ctx)
	})
	wg.Go(func() {
		a.updateBalance(ctx)
	})
	wg.Go(func() {
		a.updateName(ctx)
	})
	wg.Wait()
}

func (a *CorporationWallet) updateBalance(ctx context.Context) {
	reset := func() {
		fyne.Do(func() {
			if a.OnBalanceUpdate != nil {
				a.OnBalanceUpdate(optional.Optional[float64]{})
			}
			if a.OnTopUpdate != nil {
				a.OnTopUpdate("")
			}
		})
	}
	setBalance := func(s string, i widget.Importance) {
		fyne.Do(func() {
			a.balance.Text, a.balance.Importance = s, i
			a.balance.Refresh()
		})
	}
	corporationID := a.corporation.Load().IDOrZero()
	if corporationID == 0 {
		reset()
		setBalance("", widget.MediumImportance)
		return
	}
	hasData := a.u.StatusCache().HasCorporationSection(corporationID, app.SectionCorporationWalletBalances)
	if !hasData {
		reset()
		setBalance("No data", widget.WarningImportance)
		return
	}
	balance, err := a.u.Corporation().GetWalletBalance(ctx, corporationID, a.division)
	if errors.Is(err, app.ErrNotFound) {
		reset()
		setBalance("No data", widget.WarningImportance)
		return
	}
	if err != nil {
		slog.Error("Failed to update corp wallet ballance UI", "corporationID", corporationID, "err", err)
		reset()
		setBalance("Error: "+a.u.ErrorDisplay(err), widget.DangerImportance)
		return
	}
	s := fmt.Sprintf("%s ISK", humanize.FormatFloat(app.FloatFormat, balance))
	if balance > 1000 {
		s += fmt.Sprintf(" (%s)", ihumanize.NumberF(balance, 1))
	}
	setBalance(s, widget.MediumImportance)
	fyne.Do(func() {
		if a.OnBalanceUpdate != nil {
			a.OnBalanceUpdate(optional.New(balance))
		}
		if a.OnTopUpdate != nil {
			a.OnTopUpdate(s)
		}
	})
}

func (a *CorporationWallet) updateName(ctx context.Context) {
	var err error
	var name string
	corporationID := a.corporation.Load().IDOrZero()
	hasData := a.u.StatusCache().HasCorporationSection(corporationID, app.SectionCorporationDivisions)
	if hasData {
		n, err2 := a.u.Corporation().GetWalletName(ctx, corporationID, a.division)
		if errors.Is(err2, app.ErrNotFound) {
			hasData = false
		} else if err2 != nil {
			slog.Error("Failed to update corp wallet name UI", "corporationID", corporationID, "err", err2)
			err = err2
		} else {
			name = n
		}
	}
	if !hasData || name == "" || err != nil {
		name = a.division.DefaultWalletName()
		fyne.Do(func() {
			a.name.Hide()
		})
	} else {
		fyne.Do(func() {
			a.name.SetText(a.division.DefaultWalletName())
			a.name.Show()
		})
	}
	fyne.Do(func() {
		if a.NnNameUpdate != nil {
			a.NnNameUpdate(name)
		}
	})
}
