package ui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
)

type corporationWallet struct {
	widget.BaseWidget

	onBalanceUpdate func(balance string)
	onNameUpdate    func(name string)

	balance      *widget.Label
	corporation  *app.Corporation
	division     app.Division
	journal      *walletJournal
	name         *widget.Label
	transactions *walletTransactions
	u            *baseUI
}

func newCorporationWallet(u *baseUI, division app.Division) *corporationWallet {
	a := &corporationWallet{
		balance:      widget.NewLabel(""),
		division:     division,
		journal:      newCorporationWalletJournal(u, division),
		name:         widget.NewLabel(""),
		transactions: newCorporationWalletTransactions(u, division),
		u:            u,
	}
	a.name.TextStyle.Italic = true
	a.ExtendBaseWidget(a)
	a.u.corporationExchanged.AddListener(
		func(_ context.Context, c *app.Corporation) {
			a.corporation = c
		},
	)
	a.u.corporationSectionChanged.AddListener(func(_ context.Context, arg corporationSectionUpdated) {
		if corporationIDOrZero(a.corporation) != arg.corporationID {
			return
		}
		switch arg.section {
		case app.SectionCorporationWalletBalances:
			a.updateBalance()
		case app.SectionCorporationDivisions:
			a.updateName()
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

func (a *corporationWallet) update() {
	g := new(errgroup.Group)
	g.Go(func() error {
		a.journal.update()
		return nil
	})
	g.Go(func() error {
		a.transactions.update()
		return nil
	})
	g.Go(func() error {
		a.updateBalance()
		return nil
	})
	g.Go(func() error {
		a.updateName()
		return nil
	})
	g.Wait()
}

func (a *corporationWallet) updateBalance() {
	var err error
	var balance float64
	corporationID := corporationIDOrZero(a.corporation)
	hasData := a.u.scs.HasCorporationSection(corporationID, app.SectionCorporationWalletBalances)
	if hasData {
		b, err2 := a.u.rs.GetWalletBalance(context.Background(), corporationID, a.division)
		if errors.Is(err2, app.ErrNotFound) {
			hasData = false
		} else if err2 != nil {
			slog.Error("Failed to update corp wallet ballance UI", "corporationID", corporationID, "err", err2)
			err = err2
		} else {
			balance = b
		}
	}
	t, i := a.u.makeTopText(corporationID, hasData, err, func() (string, widget.Importance) {
		b1 := humanize.FormatFloat(app.FloatFormat, balance)
		b2 := ihumanize.Number(balance, 1)
		s := fmt.Sprintf("Balance: %s ISK (%s)", b1, b2)

		return s, widget.MediumImportance
	})
	var s string
	if !hasData || err != nil {
		s = ""
	} else {
		s = ihumanize.Number(balance, 1)
	}
	if a.onBalanceUpdate != nil {
		a.onBalanceUpdate(s)
	}
	fyne.Do(func() {
		a.balance.Text = t
		a.balance.Importance = i
		a.balance.Refresh()
	})
}

func (a *corporationWallet) updateName() {
	var err error
	var name string
	corporationID := corporationIDOrZero(a.corporation)
	hasData := a.u.scs.HasCorporationSection(corporationID, app.SectionCorporationDivisions)
	if hasData {
		n, err2 := a.u.rs.GetWalletName(context.Background(), corporationID, a.division)
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
	if a.onNameUpdate != nil {
		a.onNameUpdate(name)
	}

}
