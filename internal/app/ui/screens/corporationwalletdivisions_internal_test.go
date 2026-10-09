package screens

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestCorporationWalletRow(t *testing.T) {
	r := corporationWalletRow{balance: 1234.5, division: app.Division2, name: "Ops"}
	assert.Equal(t, "1,234.50", r.balanceDisplay())
	assert.Equal(t, "2", r.divisionDisplay())
	assert.Equal(t, "Ops [2]", r.label())

	total := corporationWalletRow{balance: 1, isTotal: true, name: "Total"}
	assert.Equal(t, "", total.divisionDisplay())
	assert.Equal(t, "Total", total.label())
}

func TestCorporationWalletDivisions(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	newDivisions := func(t *testing.T, isMobile bool) *CorporationWalletDivisions {
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			IsMobile: isMobile,
			Storage:  st,
		})
		return NewCorporationWalletDivisions(u)
	}
	labels := func(rows []corporationWalletRow) []string {
		return xslices.Map(rows, func(r corporationWalletRow) string {
			return r.label()
		})
	}

	for _, isMobile := range []bool{true, false} {
		t.Run(fmt.Sprintf("can render mobile=%v", isMobile), func(t *testing.T) {
			a := newDivisions(t, isMobile)
			test.WidgetRenderer(a)
		})
	}
	t.Run("uses table on desktop and list on mobile", func(t *testing.T) {
		assert.IsType(t, &widget.Table{}, newDivisions(t, false).body)
		assert.IsType(t, &widget.List{}, newDivisions(t, true).body)
	})
	t.Run("shows empty when no corporation", func(t *testing.T) {
		a := newDivisions(t, false)
		assert.Empty(t, a.rowsSorted)
		assert.False(t, a.status.Visible())
	})
	for _, isMobile := range []bool{true, false} {
		t.Run(fmt.Sprintf("shows balances with total row last mobile=%v", isMobile), func(t *testing.T) {
			testutil.MustTruncateTables(db)
			a := newDivisions(t, isMobile)

			a.u.Signals().CurrentCorporationExchanged.Emit(t.Context(), createCorporationWithWalletBalances(factory))

			if assert.Len(t, a.rowsSorted, 8) {
				assert.Equal(t, corporationWalletRow{balance: 1000, division: app.Division1, name: "Master Wallet"}, a.rowsSorted[0])
				assert.Equal(t, corporationWalletRow{balance: 234.5, division: app.Division2, name: "Ops"}, a.rowsSorted[1])
				assert.Equal(t, corporationWalletRow{balance: 1234.5, isTotal: true, name: "Total"}, a.rowsSorted[7])
			}
			assert.False(t, a.status.Visible())
		})
	}
	t.Run("keeps total row last when sorting", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		a := newDivisions(t, false)
		a.u.Signals().CurrentCorporationExchanged.Emit(t.Context(), createCorporationWithWalletBalances(factory))

		a.sortRows("Balance") // ascending

		if assert.Len(t, a.rowsSorted, 8) {
			assert.Equal(t, "Ops [2]", a.rowsSorted[5].label())
			assert.Equal(t, "Master Wallet [1]", a.rowsSorted[6].label())
			assert.True(t, a.rowsSorted[7].isTotal)
		}

		a.sortRows("Balance") // descending

		if assert.Len(t, a.rowsSorted, 8) {
			assert.Equal(t, []string{"Master Wallet [1]", "Ops [2]"}, labels(a.rowsSorted[:2]))
			assert.True(t, a.rowsSorted[7].isTotal)
		}
	})
	t.Run("keeps sort when data is reloaded", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		a := newDivisions(t, false)
		c := createCorporationWithWalletBalances(factory)
		a.u.Signals().CurrentCorporationExchanged.Emit(t.Context(), c)
		a.sortRows("Balance")
		a.sortRows("Balance") // descending

		a.u.Signals().CorporationSectionChanged.Emit(t.Context(), app.CorporationSectionUpdated{
			CorporationID: c.ID,
			Section:       app.SectionCorporationWalletBalances,
		})

		assert.Equal(t, []string{"Master Wallet [1]", "Ops [2]"}, labels(a.rowsSorted[:2]))
	})
	t.Run("updates when wallet is renamed", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		a := newDivisions(t, false)
		c := createCorporationWithWalletBalances(factory)
		a.u.Signals().CurrentCorporationExchanged.Emit(t.Context(), c)
		factory.CreateCorporationWalletName(storage.UpdateOrCreateCorporationWalletNameParams{
			CorporationID: c.ID,
			DivisionID:    2,
			Name:          "Logistics",
		})

		a.u.Signals().CorporationSectionChanged.Emit(t.Context(), app.CorporationSectionUpdated{
			CorporationID: c.ID,
			Section:       app.SectionCorporationDivisions,
		})

		if assert.Len(t, a.rowsSorted, 8) {
			assert.Equal(t, "Logistics [2]", a.rowsSorted[1].label())
		}
	})
	t.Run("shows no data when balances are missing", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		a := newDivisions(t, false)

		a.u.Signals().CurrentCorporationExchanged.Emit(t.Context(), factory.CreateCorporation())

		assert.Equal(t, "No data", a.status.Text)
		assert.True(t, a.status.Visible())
		assert.Empty(t, a.rowsSorted)
	})
}
