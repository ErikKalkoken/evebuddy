package ui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type characterWallet struct {
	widget.BaseWidget

	onTopUpdate     func(top string)
	onBalanceUpdate func(balance float64)

	balance      *widget.Label
	character    atomic.Pointer[app.Character]
	journal      *walletJournal
	transactions *walletTransactions
	u            *baseUI
}

func newCharacterWallet(u *baseUI) *characterWallet {
	a := &characterWallet{
		balance:      iwidget.NewLabelWithSelection(""),
		journal:      newCharacterWalletJournal(u),
		transactions: newCharacterWalletTransaction(u),
		u:            u,
	}
	a.ExtendBaseWidget(a)
	a.u.currentCharacterExchanged.AddListener(func(_ context.Context, c *app.Character) {
		a.character.Store(c)
		a.update()
	})
	a.u.characterSectionChanged.AddListener(func(_ context.Context, arg characterSectionUpdated) {
		if characterIDOrZero(a.character.Load()) != arg.characterID {
			return
		}
		if arg.section == app.SectionCharacterWalletBalance {
			a.updateBalance()
		}
	})
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
	characterID := characterIDOrZero(a.character.Load())
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
		b2 := ihumanize.NumberF(balance, 1)
		s := fmt.Sprintf("%s ISK (%s)", b1, b2)
		return s, widget.MediumImportance
	})
	if a.onBalanceUpdate != nil {
		a.onBalanceUpdate(balance)
	}
	if a.onTopUpdate != nil {
		a.onTopUpdate(t)
	}
	fyne.Do(func() {
		a.balance.Text = t
		a.balance.Importance = i
		a.balance.Refresh()
	})
}
