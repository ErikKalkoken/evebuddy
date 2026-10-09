package screens

import (
	"fmt"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
)

func TestCorporationWalletLabel(t *testing.T) {
	assert.Equal(t, "Ops [3]", corporationWalletLabel(app.Division3, "Ops"))
	assert.Equal(t, "Master Wallet [1]", corporationWalletLabel(app.Division1, ""))
}

func TestCorporationWallets(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	newCorporationWallets := func(t *testing.T, isMobile bool) *CorporationWallets {
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			IsMobile: isMobile,
			Storage:  st,
		})
		return NewCorporationWallets(u)
	}
	assertShows := func(t *testing.T, a *CorporationWallets, want app.Division) {
		t.Helper()
		for _, d := range app.Divisions {
			assert.Equal(t, d == want, a.journals[d].Visible(), "journal %d", d)
			assert.Equal(t, d == want, a.transactions[d].Visible(), "transactions %d", d)
		}
	}

	for _, isMobile := range []bool{true, false} {
		t.Run(fmt.Sprintf("can render mobile=%v", isMobile), func(t *testing.T) {
			a := newCorporationWallets(t, isMobile)
			test.WidgetRenderer(a)
		})
	}
	t.Run("sizes selector for longest label", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		a := newCorporationWallets(t, false)
		c := createCorporationWithWalletBalances(factory)
		factory.CreateCorporationWalletName(storage.UpdateOrCreateCorporationWalletNameParams{
			CorporationID: c.ID,
			DivisionID:    3,
			Name:          "Alliance Logistics and Fuel Reserve",
		})

		a.u.Signals().CurrentCorporationExchanged.Emit(t.Context(), c)

		assert.Equal(t, "Alliance Logistics and Fuel Reserve [3]", a.walletSelect.PlaceHolder)
	})
	t.Run("shows master wallet by default", func(t *testing.T) {
		a := newCorporationWallets(t, false)
		assert.Equal(t, app.Division1, a.selected)
		assert.Equal(t, "Master Wallet [1]", a.walletSelect.Selected)
		assertShows(t, a, app.Division1)
	})
	t.Run("can switch wallet", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		a := newCorporationWallets(t, false)
		c := createCorporationWithWalletBalances(factory)
		a.u.Signals().CurrentCorporationExchanged.Emit(t.Context(), c)

		a.walletSelect.SetSelected("Ops [2]")

		assert.Equal(t, app.Division2, a.selected)
		assertShows(t, a, app.Division2)
		assert.Equal(t, "234.50 ISK", a.balanceLabel.Text)
	})
	t.Run("shows balances and labels", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		a := newCorporationWallets(t, false)
		c := createCorporationWithWalletBalances(factory)

		a.u.Signals().CurrentCorporationExchanged.Emit(t.Context(), c)

		assert.Equal(t, "1,000.00 ISK", a.balanceLabel.Text)
		assert.Contains(t, a.walletSelect.Options, "Ops [2]")
	})
	t.Run("resets to master wallet when corporation changes", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		a := newCorporationWallets(t, false)
		a.u.Signals().CurrentCorporationExchanged.Emit(t.Context(), createCorporationWithWalletBalances(factory))
		a.walletSelect.SetSelected("Ops [2]")

		a.u.Signals().CurrentCorporationExchanged.Emit(t.Context(), createCorporationWithWalletBalances(factory))

		assert.Equal(t, app.Division1, a.selected)
		assert.Equal(t, "Master Wallet [1]", a.walletSelect.Selected)
		assertShows(t, a, app.Division1)
	})
	t.Run("keeps selection when wallet is renamed", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		a := newCorporationWallets(t, false)
		c := createCorporationWithWalletBalances(factory)
		a.u.Signals().CurrentCorporationExchanged.Emit(t.Context(), c)
		a.walletSelect.SetSelected("Ops [2]")
		factory.CreateCorporationWalletName(storage.UpdateOrCreateCorporationWalletNameParams{
			CorporationID: c.ID,
			DivisionID:    2,
			Name:          "Logistics",
		})

		a.u.Signals().CorporationSectionChanged.Emit(t.Context(), app.CorporationSectionUpdated{
			CorporationID: c.ID,
			Section:       app.SectionCorporationDivisions,
		})

		assert.Equal(t, app.Division2, a.selected)
		assert.Equal(t, "Logistics [2]", a.walletSelect.Selected)
		assertShows(t, a, app.Division2)
	})
	t.Run("shows no data when balances are missing", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		a := newCorporationWallets(t, false)

		a.u.Signals().CurrentCorporationExchanged.Emit(t.Context(), factory.CreateCorporation())

		assert.Equal(t, "No data", a.balanceLabel.Text)
	})
}

// createCorporationWithWalletBalances returns a corporation with balances for wallets 1 and 2,
// where wallet 2 is named "Ops".
func createCorporationWithWalletBalances(factory testutil.Factory) *app.Corporation {
	c := factory.CreateCorporation()
	factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletBalances)
	factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
		CorporationID: c.ID,
		Section:       app.SectionCorporationWalletBalances,
		CompletedAt:   time.Now(),
	})
	factory.CreateCorporationWalletBalance(storage.UpdateOrCreateCorporationWalletBalanceParams{
		CorporationID: c.ID,
		DivisionID:    1,
		Balance:       1000,
	})
	factory.CreateCorporationWalletBalance(storage.UpdateOrCreateCorporationWalletBalanceParams{
		CorporationID: c.ID,
		DivisionID:    2,
		Balance:       234.5,
	})
	factory.CreateCorporationWalletName(storage.UpdateOrCreateCorporationWalletNameParams{
		CorporationID: c.ID,
		DivisionID:    2,
		Name:          "Ops",
	})
	return c
}
