package screens

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type CharacterWallet struct {
	widget.BaseWidget

	OnBalanceUpdate func(balance optional.Optional[float64])

	journal       *WalletJournal
	transactions  *WalletTransactions
	loyaltyPoints *CharacterLoyaltyPoints
}

func NewCharacterWallet(u baseUI) *CharacterWallet {
	a := &CharacterWallet{
		journal:       NewCharacterWalletJournal(u),
		transactions:  NewCharacterWalletTransaction(u),
		loyaltyPoints: NewCharacterLoyaltyPoints(u),
	}
	a.ExtendBaseWidget(a)
	a.journal.onBalanceUpdate = func(balance optional.Optional[float64]) {
		if a.OnBalanceUpdate != nil {
			a.OnBalanceUpdate(balance)
		}
	}
	return a
}

func (a *CharacterWallet) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewAppTabs(
		ui.NewTabItem("Transactions", a.journal),
		ui.NewTabItem("Market", a.transactions),
		ui.NewTabItem("Loyalty Points", a.loyaltyPoints),
	)
	return widget.NewSimpleRenderer(c)
}
