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

type characterWallet struct {
	widget.BaseWidget

	OnUpdate func(balance string)

	balance      *widget.Label
	journal      *characterWalletJournal
	transactions *characterWalletTransactions
	u            *baseUI
}

func newCharacterWallet(u *baseUI) *characterWallet {
	a := &characterWallet{
		balance:      widget.NewLabel(""),
		journal:      newCharacterWalletJournal(u),
		transactions: newCharacterWalletTransaction(u),
		u:            u,
	}
	a.ExtendBaseWidget(a)
	return a
}

func (a *characterWallet) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		a.balance,
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

func (a *characterWallet) update() {
	go a.journal.update()
	go a.transactions.update()
	go a.updateBalance()
}

func (a *characterWallet) updateBalance() {
	var err error
	var balance float64
	characterID := a.u.currentCharacterID()
	hasData := a.u.scs.HasCharacterSection(characterID, app.SectionCharacterWalletBalance)
	if hasData {
		c, err2 := a.u.cs.GetCharacter(context.Background(), characterID)
		if errors.Is(err2, app.ErrNotFound) {
			hasData = false
		} else if err2 != nil {
			slog.Error("Failed to update character wallet ballance UI", "characterID", characterID, "err", err2)
			err = err2
		} else {
			if c.WalletBalance.IsEmpty() {
				hasData = false
			} else {
				balance = c.WalletBalance.ValueOrZero()
			}
		}
	}
	t, i := a.u.makeTopText(characterID, hasData, err, func() (string, widget.Importance) {
		b1 := humanize.FormatFloat(app.FloatFormat, balance)
		b2 := ihumanize.Number(balance, 2)
		s := fmt.Sprintf("Balance: %s ISK (%s)", b1, b2)
		if a.OnUpdate != nil {
			a.OnUpdate(b2)
		}
		return s, widget.MediumImportance
	})
	fyne.Do(func() {
		a.balance.Text = t
		a.balance.Importance = i
		a.balance.Refresh()
	})
}
