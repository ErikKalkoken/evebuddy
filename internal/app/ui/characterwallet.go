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
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type characterWallet struct {
	widget.BaseWidget

	onTopUpdate     func(top string)
	onBalanceUpdate func(balance float64)

	balance       *widget.Label
	character     atomic.Pointer[app.Character]
	journal       *walletJournal
	transactions  *walletTransactions
	loyaltyPoints *characterLoyaltyPoints
	u             *baseUI
}

func newCharacterWallet(u *baseUI) *characterWallet {
	a := &characterWallet{
		balance:       iwidget.NewLabelWithSelection(""),
		journal:       newCharacterWalletJournal(u),
		transactions:  newCharacterWalletTransaction(u),
		loyaltyPoints: newCharacterLoyaltyPoints(u),
		u:             u,
	}
	a.ExtendBaseWidget(a)
	a.u.currentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
		a.character.Store(c)
		a.update(ctx)
	})
	a.u.characterSectionChanged.AddListener(func(ctx context.Context, arg characterSectionUpdated) {
		if characterIDOrZero(a.character.Load()) != arg.characterID {
			return
		}
		if arg.section == app.SectionCharacterWalletBalance {
			a.updateBalance(ctx)
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
			container.NewTabItem("Loyalty Points", a.loyaltyPoints),
		),
	)
	return widget.NewSimpleRenderer(c)
}

func (a *characterWallet) update(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Go(func() {
		a.journal.update(ctx)
	})
	wg.Go(func() {
		a.transactions.update(ctx)
	})
	wg.Go(func() {
		a.loyaltyPoints.update(ctx)
	})
	wg.Go(func() {
		a.updateBalance(ctx)
	})
	wg.Wait()
}

func (a *characterWallet) updateBalance(ctx context.Context) {
	var err error
	var balance float64
	characterID := characterIDOrZero(a.character.Load())
	hasData := a.u.scs.HasCharacterSection(characterID, app.SectionCharacterWalletBalance)
	if hasData {
		c, err2 := a.u.cs.GetCharacter(ctx, characterID)
		if errors.Is(err2, app.ErrNotFound) {
			hasData = false
		} else if err2 != nil {
			slog.Error("Failed to update character wallet ballance UI", "characterID", characterID, "err", err2)
			err = err2
		} else {
			if v, ok := c.WalletBalance.Value(); ok {
				balance = v
			} else {
				hasData = false
			}
		}
	}
	t, i := a.u.makeTopText(characterID, hasData, err, func() (string, widget.Importance) {
		b1 := humanize.FormatFloat(app.FloatFormat, balance)
		b2 := ihumanize.NumberF(balance, 1)
		s := fmt.Sprintf("%s ISK (%s)", b1, b2)
		return s, widget.MediumImportance
	})
	fyne.Do(func() {
		if a.onTopUpdate != nil {
			a.onTopUpdate(t)
		}
		if a.onBalanceUpdate != nil {
			a.onBalanceUpdate(balance)
		}
		a.balance.Text = t
		a.balance.Importance = i
		a.balance.Refresh()
	})
}
