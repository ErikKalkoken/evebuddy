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

	"github.com/ErikKalkoken/evebuddy/internal/app"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
)

type corporationWallet struct {
	widget.BaseWidget

	onUpdate func(balance string)

	balance      *widget.Label
	division     app.Division
	journal      *corporationWalletJournal
	name         *widget.Label
	transactions *corporationWalletTransactions
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
	go a.journal.update()
	go a.transactions.update()
	go a.updateBalance()
	go a.updateName()
}

func (a *corporationWallet) updateBalance() {
	var err error
	var balance float64
	corporationID := a.u.currentCorporationID()
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
	t, i := a.u.makeTopTextCorporation(corporationID, hasData, err, func() (string, widget.Importance) {
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
	if a.onUpdate != nil {
		a.onUpdate(s)
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
	corporationID := a.u.currentCorporationID()
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
		fyne.Do(func() {
			a.name.Hide()
		})
	}
	fyne.Do(func() {
		a.name.SetText(name)
		a.name.Show()
	})
}
