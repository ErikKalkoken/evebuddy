package screens

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

// CorporationWallets shows the wallets of the current corporation.
type CorporationWallets struct {
	widget.BaseWidget

	// OnBalanceUpdate is called on the main thread with the total balance after each update.
	OnBalanceUpdate func(total optional.Optional[float64])

	divisions    *CorporationWalletDivisions
	journal      *WalletJournal
	transactions *WalletTransactions
}

func NewCorporationWallets(u baseUI) *CorporationWallets {
	a := &CorporationWallets{
		divisions:    NewCorporationWalletDivisions(u),
		journal:      NewCorporationWalletJournal(u),
		transactions: NewCorporationWalletTransactions(u),
	}
	a.ExtendBaseWidget(a)
	a.divisions.onBalanceUpdate = func(balance optional.Optional[float64]) {
		if a.OnBalanceUpdate != nil {
			a.OnBalanceUpdate(balance)
		}
	}
	return a
}

func (a *CorporationWallets) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewAppTabs(
		ui.NewTabItem("Transactions", a.journal),
		ui.NewTabItem("Market", a.transactions),
		ui.NewTabItem("Divisions", a.divisions),
	)
	return widget.NewSimpleRenderer(c)
}

// corporationWalletLabel returns the display label for a wallet, e.g. "[1] Master Wallet".
func corporationWalletLabel(d app.Division, name string) string {
	if name == "" {
		name = d.DefaultWalletName()
	}
	return fmt.Sprintf("[%d] %s", d, name)
}

func defaultCorporationWalletNames() map[app.Division]string {
	m := make(map[app.Division]string)
	for _, d := range app.Divisions {
		m[d] = d.DefaultWalletName()
	}
	return m
}
