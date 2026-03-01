package ui

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
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type corporationWallet struct {
	widget.BaseWidget

	onBalanceUpdate func(balance optional.Optional[float64])
	onNameUpdate    func(name string)
	onTopUpdate     func(top string)

	balance      *widget.Label
	corporation  atomic.Pointer[app.Corporation]
	division     app.Division
	journal      *walletJournal
	name         *widget.Label
	transactions *walletTransactions
	u            *baseUI
}

func newCorporationWallet(u *baseUI, division app.Division) *corporationWallet {
	a := &corporationWallet{
		balance:      iwidget.NewLabelWithSelection(""),
		division:     division,
		journal:      newCorporationWalletJournal(u, division),
		name:         widget.NewLabel(""),
		transactions: newCorporationWalletTransactions(u, division),
		u:            u,
	}
	a.name.TextStyle.Italic = true
	a.ExtendBaseWidget(a)
	a.u.currentCorporationExchanged.AddListener(func(ctx context.Context, c *app.Corporation) {
		a.corporation.Store(c)
		a.update(ctx)
	})
	a.u.corporationSectionChanged.AddListener(func(ctx context.Context, arg corporationSectionUpdated) {
		if corporationIDOrZero(a.corporation.Load()) != arg.corporationID {
			return
		}
		switch arg.section {
		case app.SectionCorporationWalletBalances:
			a.updateBalance(ctx)
		case app.SectionCorporationDivisions:
			a.updateName(ctx)
		}
	})
	return a
}

func (a *corporationWallet) CreateRenderer() fyne.WidgetRenderer {
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

func (a *corporationWallet) update(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Go(func() {
		a.journal.update(ctx)
	})
	wg.Go(func() {
		a.transactions.update(ctx)
	})
	wg.Go(func() {
		a.updateBalance(ctx)
	})
	wg.Go(func() {
		a.updateName(ctx)
	})
	wg.Wait()
}

func (a *corporationWallet) updateBalance(ctx context.Context) {
	clear := func() {
		fyne.Do(func() {
			if a.onBalanceUpdate != nil {
				a.onBalanceUpdate(optional.Optional[float64]{})
			}
			if a.onTopUpdate != nil {
				a.onTopUpdate("")
			}
		})
	}
	setBalance := func(s string, i widget.Importance) {
		fyne.Do(func() {
			a.balance.Text, a.balance.Importance = s, i
			a.balance.Refresh()
		})
	}
	corporationID := corporationIDOrZero(a.corporation.Load())
	if corporationID == 0 {
		clear()
		setBalance("", widget.MediumImportance)
		return
	}
	hasData := a.u.scs.HasCorporationSection(corporationID, app.SectionCorporationWalletBalances)
	if !hasData {
		clear()
		setBalance("No data", widget.WarningImportance)
		return
	}
	balance, err := a.u.rs.GetWalletBalance(ctx, corporationID, a.division)
	if errors.Is(err, app.ErrNotFound) {
		clear()
		setBalance("No data", widget.WarningImportance)
		return
	}
	if err != nil {
		slog.Error("Failed to update corp wallet ballance UI", "corporationID", corporationID, "err", err)
		clear()
		setBalance("Error: "+a.u.humanizeError(err), widget.DangerImportance)
		return
	}
	s := fmt.Sprintf("%s ISK", humanize.FormatFloat(app.FloatFormat, balance))
	if balance > 1000 {
		s += fmt.Sprintf(" (%s)", ihumanize.NumberF(balance, 1))
	}
	setBalance(s, widget.MediumImportance)
	fyne.Do(func() {
		if a.onBalanceUpdate != nil {
			a.onBalanceUpdate(optional.New(balance))
		}
		if a.onTopUpdate != nil {
			a.onTopUpdate(s)
		}
	})
}

func (a *corporationWallet) updateName(ctx context.Context) {
	var err error
	var name string
	corporationID := corporationIDOrZero(a.corporation.Load())
	hasData := a.u.scs.HasCorporationSection(corporationID, app.SectionCorporationDivisions)
	if hasData {
		n, err2 := a.u.rs.GetWalletName(ctx, corporationID, a.division)
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
		if a.onNameUpdate != nil {
			a.onNameUpdate(name)
		}
	})
}
