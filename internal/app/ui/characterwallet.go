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
	"github.com/ErikKalkoken/evebuddy/internal/app/uiservices"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type CharacterWallet struct {
	widget.BaseWidget

	onTopUpdate     func(top string)
	onBalanceUpdate func(balance optional.Optional[float64])

	balance       *widget.Label
	character     atomic.Pointer[app.Character]
	journal       *walletJournal
	transactions  *walletTransactions
	loyaltyPoints *characterLoyaltyPoints
	u         uiservices.UIServices
}

func NewCharacterWallet(u         uiservices.UIServices) *CharacterWallet {
	a := &CharacterWallet{
		balance:       xwidget.NewLabelWithSelection(""),
		journal:       newCharacterWalletJournal(u),
		transactions:  newCharacterWalletTransaction(u),
		loyaltyPoints: newCharacterLoyaltyPoints(u),
		u:             u,
	}
	a.ExtendBaseWidget(a)
	a.u.Signals().CurrentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
		a.character.Store(c)
		a.update(ctx)
	})
	a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		if characterIDOrZero(a.character.Load()) != arg.CharacterID {
			return
		}
		if arg.Section == app.SectionCharacterWalletBalance {
			a.updateBalance(ctx)
		}
	})
	return a
}

func (a *CharacterWallet) CreateRenderer() fyne.WidgetRenderer {
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

func (a *CharacterWallet) update(ctx context.Context) {
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

func (a *CharacterWallet) updateBalance(ctx context.Context) {
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
	characterID := characterIDOrZero(a.character.Load())
	if characterID == 0 {
		clear()
		setBalance("", widget.MediumImportance)
		return
	}

	hasData := a.u.StatusCache().HasCharacterSection(characterID, app.SectionCharacterWalletBalance)
	if !hasData {
		clear()
		setBalance("No data", widget.WarningImportance)
		return
	}

	c, err := a.u.Character().GetCharacter(ctx, characterID)
	if errors.Is(err, app.ErrNotFound) {
		clear()
		setBalance("No data", widget.WarningImportance)
		return
	}
	if err != nil {
		slog.Error("Failed to update character wallet ballance UI", "characterID", characterID, "err", err)
		clear()
		setBalance("Error: "+app.ErrorDisplay(err), widget.DangerImportance)
		return
	}
	balance, ok := c.WalletBalance.Value()
	if !ok {
		clear()
		setBalance("No data", widget.WarningImportance)
		return
	}

	s := fmt.Sprintf("%s ISK", humanize.FormatFloat(app.FloatFormat, balance))
	if balance > 1000 {
		s += fmt.Sprintf(" (%s)", ihumanize.NumberF(balance, 1))
	}
	setBalance(s, widget.MediumImportance)
	fyne.Do(func() {
		if a.onTopUpdate != nil {
			a.onTopUpdate(s)
		}
		if a.onBalanceUpdate != nil {
			a.onBalanceUpdate(optional.New(balance))
		}
	})
}
