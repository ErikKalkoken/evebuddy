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

type CharacterWallet struct {
	widget.BaseWidget

	OnTopUpdate     func(top string)
	OnBalanceUpdate func(balance optional.Optional[float64])

	balance       *widget.Label
	character     atomic.Pointer[app.Character]
	journal       *WalletJournal
	transactions  *WalletTransactions
	loyaltyPoints *CharacterLoyaltyPoints
	u             baseUI
}

func NewCharacterWallet(u baseUI) *CharacterWallet {
	a := &CharacterWallet{
		balance:       xwidget.NewLabelWithSelection(""),
		journal:       NewCharacterWalletJournal(u),
		transactions:  NewCharacterWalletTransaction(u),
		loyaltyPoints: NewCharacterLoyaltyPoints(u),
		u:             u,
	}
	a.ExtendBaseWidget(a)
	a.u.Signals().CurrentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
		a.character.Store(c)
		a.Update(ctx)
	})
	a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		if a.character.Load().IDOrZero() != arg.CharacterID {
			return
		}
		if arg.Section == app.SectionCharacterWalletBalance {
			a.UpdateBalance(ctx)
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

func (a *CharacterWallet) Update(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Go(func() {
		a.journal.Update(ctx)
	})
	wg.Go(func() {
		a.transactions.Update(ctx)
	})
	wg.Go(func() {
		a.loyaltyPoints.Update(ctx)
	})
	wg.Go(func() {
		a.UpdateBalance(ctx)
	})
	wg.Wait()
}

func (a *CharacterWallet) UpdateBalance(ctx context.Context) {
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
	characterID := a.character.Load().IDOrZero()
	if characterID == 0 {
		reset()
		setBalance("", widget.MediumImportance)
		return
	}

	hasData := a.u.StatusCache().HasCharacterSection(characterID, app.SectionCharacterWalletBalance)
	if !hasData {
		reset()
		setBalance("No data", widget.WarningImportance)
		return
	}

	c, err := a.u.Character().GetCharacter(ctx, characterID)
	if errors.Is(err, app.ErrNotFound) {
		reset()
		setBalance("No data", widget.WarningImportance)
		return
	}
	if err != nil {
		slog.Error("Failed to update character wallet ballance UI", "characterID", characterID, "err", err)
		reset()
		setBalance("Error: "+a.u.ErrorDisplay(err), widget.DangerImportance)
		return
	}
	balance, ok := c.WalletBalance.Value()
	if !ok {
		reset()
		setBalance("No data", widget.WarningImportance)
		return
	}

	s := fmt.Sprintf("%s ISK", humanize.FormatFloat(app.FloatFormat, balance))
	if balance > 1000 {
		s += fmt.Sprintf(" (%s)", ihumanize.NumberF(balance, 1))
	}
	setBalance(s, widget.MediumImportance)
	fyne.Do(func() {
		if a.OnTopUpdate != nil {
			a.OnTopUpdate(s)
		}
		if a.OnBalanceUpdate != nil {
			a.OnBalanceUpdate(optional.New(balance))
		}
	})
}
