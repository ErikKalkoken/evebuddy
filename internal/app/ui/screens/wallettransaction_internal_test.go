package screens

import (
	"database/sql"
	"fmt"
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestWalletTransactionFilter_Match(t *testing.T) {
	r := walletTransactionRow{
		categoryName: "Ship",
		clientName:   "Bruce",
		isBuy:        true,
		locationName: "Jita IV - Moon 4",
		regionName:   "The Forge",
		typeName:     "Merlin",
	}
	sell := r
	sell.isBuy = false
	for _, tc := range []struct {
		name   string
		filter walletTransactionFilter
		row    walletTransactionRow
		want   bool
	}{
		{"no filter", walletTransactionFilter{}, r, true},
		{"buy matches", walletTransactionFilter{activity: marketTransactionActivityBuy}, r, true},
		{"buy but is sell", walletTransactionFilter{activity: marketTransactionActivityBuy}, sell, false},
		{"sell matches", walletTransactionFilter{activity: marketTransactionActivitySell}, sell, true},
		{"sell but is buy", walletTransactionFilter{activity: marketTransactionActivitySell}, r, false},
		{"category matches", walletTransactionFilter{category: "Ship"}, r, true},
		{"category differs", walletTransactionFilter{category: "Module"}, r, false},
		{"client differs", walletTransactionFilter{client: "Alice"}, r, false},
		{"location differs", walletTransactionFilter{location: "Amarr"}, r, false},
		{"region differs", walletTransactionFilter{region: "Domain"}, r, false},
		{"type differs", walletTransactionFilter{typeName: "Rifter"}, r, false},
		{"all match", walletTransactionFilter{activity: marketTransactionActivityBuy, category: "Ship", client: "Bruce", location: "Jita IV - Moon 4", region: "The Forge", typeName: "Merlin"}, r, true},
		{"one of many differs", walletTransactionFilter{category: "Ship", typeName: "Rifter"}, r, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.filter.match(tc.row))
		})
	}
}

func TestWalletTransactions_Filter(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	rows := []walletTransactionRow{
		{transactionID: 1, categoryName: "Ship", clientName: "Bruce", isBuy: true, typeName: "Merlin"},
		{transactionID: 2, categoryName: "Module", clientName: "Alice", isBuy: false, typeName: "Damage Control"},
	}
	for i := range rows {
		rows[i].setSearchTarget()
	}
	newWalletTransactions := func(t *testing.T, isMobile bool) *WalletTransactions {
		a := NewCharacterWalletTransaction(testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			IsMobile: isMobile,
			Storage:  st,
		}))
		a.rows = rows
		a.filterRowsAsync("")
		require.Len(t, a.rowsFiltered, 2)
		return a
	}
	t.Run("can filter on mobile", func(t *testing.T) {
		a := newWalletTransactions(t, true)
		a.filterChip.SetSelected(map[string]string{walletTransactionFilterActivity: marketTransactionActivitySell})
		if assert.Len(t, a.rowsFiltered, 1) {
			assert.EqualValues(t, 2, a.rowsFiltered[0].transactionID)
		}
	})
	t.Run("can filter on desktop", func(t *testing.T) {
		a := newWalletTransactions(t, false)
		a.activitySelect.SetSelected(marketTransactionActivitySell)
		if assert.Len(t, a.rowsFiltered, 1) {
			assert.EqualValues(t, 2, a.rowsFiltered[0].transactionID)
		}
	})
	t.Run("shows all filters on mobile", func(t *testing.T) {
		a := newWalletTransactions(t, true)
		assert.Equal(t, map[string]string{
			walletTransactionFilterActivity: "",
			walletTransactionFilterCategory: "",
			walletTransactionFilterClient:   "",
			walletTransactionFilterLocation: "",
			walletTransactionFilterRegion:   "",
			walletTransactionFilterType:     "",
		}, a.filterChip.Selected())
	})
	for _, isMobile := range []bool{true, false} {
		t.Run(fmt.Sprintf("can search client and type names mobile=%v", isMobile), func(t *testing.T) {
			for _, tc := range []struct {
				search string
				want   []int64
			}{
				{"bru", []int64{1}},
				{"DAMAGE", []int64{2}},
				{"merlin", []int64{1}},
				{"e", []int64{1, 2}}, // too short to search
				{"xyz", []int64{}},
			} {
				a := newWalletTransactions(t, isMobile)
				a.searchEntry.SetText(tc.search)
				got := xslices.Map(a.rowsFiltered, func(r walletTransactionRow) int64 {
					return r.transactionID
				})
				assert.ElementsMatch(t, tc.want, got, tc.search)
			}
		})
	}
}

func TestWalletTransactions_KeepsStatusWithoutCharacter(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	a := NewCharacterWalletTransaction(testdouble.NewUIFake(testdouble.UIParams{
		App:     test.NewTempApp(t),
		Storage: st,
	}))
	var pending []func() // run background work later, like the real app
	orig := runAsync
	runAsync = func(f func()) { pending = append(pending, f) }
	t.Cleanup(func() { runAsync = orig })
	a.filterRowsAsync("") // a filter still running from before

	a.updateCharacter(t.Context())
	for _, f := range pending {
		f()
	}

	assert.Equal(t, "No character", a.footerLabel.Text)
	assert.Empty(t, a.rowsFiltered)
}

func TestWalletTransactions_ErrorDisplay(t *testing.T) {
	newScreen := func(t *testing.T) (*WalletTransactions, *sql.DB) {
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		a := NewCharacterWalletTransaction(u)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletTransactions,
		})
		a.character.Store(c)
		return a, db
	}
	t.Run("shows error when update fails and keeps it on filtering", func(t *testing.T) {
		a, db := newScreen(t)
		if _, err := db.Exec("DROP TABLE character_wallet_transactions"); err != nil { // let the next update fail
			t.Fatal(err)
		}

		a.updateCharacter(t.Context())
		assert.True(t, a.errorLabel.Visible())
		assert.Contains(t, a.errorLabel.Text, "ERROR:")

		a.filterRowsAsync("")
		assert.True(t, a.errorLabel.Visible())
	})
	t.Run("hides error after successful update", func(t *testing.T) {
		a, _ := newScreen(t)
		a.errorLabel.SetText("ERROR: failed")
		a.errorLabel.Show()

		a.updateCharacter(t.Context())
		assert.False(t, a.errorLabel.Visible())
	})
}

func TestCharacterWalletTransactions(t *testing.T) {
	t.Run("shows buy and sell transactions of exchanged character", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterWalletTransaction(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCharacterFull()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletTransactions,
		})
		buy := factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{
			CharacterID: c.ID,
			IsBuy:       true,
		})
		sell := factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{
			CharacterID: c.ID,
		})
		// when
		a.u.Signals().CurrentCharacterExchanged.Emit(t.Context(), c)
		// then
		got := make(map[int64]widget.Importance)
		for _, r := range a.rows {
			got[r.transactionID] = r.totalImportance
			assert.NotEmpty(t, r.regionName)
		}
		want := map[int64]widget.Importance{
			buy.TransactionID:  widget.DangerImportance,
			sell.TransactionID: widget.SuccessImportance,
		}
		xassert.Equal(t, want, got)
	})
	t.Run("updates when transactions of current character changed", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterWalletTransaction(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCharacterFull()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletTransactions,
		})
		factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{CharacterID: c.ID})
		a.character.Store(c)
		// when
		a.u.Signals().CharacterSectionChanged.Emit(t.Context(), app.CharacterSectionUpdated{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletTransactions,
		})
		// then
		assert.Len(t, a.rows, 1)
	})
	t.Run("ignores change of other character", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterWalletTransaction(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCharacterFull()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletTransactions,
		})
		factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{CharacterID: c.ID})
		a.character.Store(c)
		other := factory.CreateCharacterFull()
		// when
		a.u.Signals().CharacterSectionChanged.Emit(t.Context(), app.CharacterSectionUpdated{
			CharacterID: other.ID,
			Section:     app.SectionCharacterWalletTransactions,
		})
		// then
		assert.Empty(t, a.rows)
	})
}

func TestCorporationWalletTransactions(t *testing.T) {
	t.Run("shows transactions of all divisions when corporation is exchanged", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWalletTransactions(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCorporation()
		for _, d := range app.Divisions {
			factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
				CorporationID: c.ID,
				Section:       app.CorporationSectionWalletTransactions(d),
			})
		}
		buy := factory.CreateCorporationWalletTransaction(storage.CreateCorporationWalletTransactionParams{
			CorporationID: c.ID,
			Division:      1,
			IsBuy:         true,
		})
		sell := factory.CreateCorporationWalletTransaction(storage.CreateCorporationWalletTransactionParams{
			CorporationID: c.ID,
			Division:      2,
		})
		// when
		a.u.Signals().CurrentCorporationExchanged.Emit(t.Context(), c)
		// then
		gotImportance := make(map[int64]widget.Importance)
		gotDivision := make(map[int64]app.Division)
		for _, r := range a.rows {
			gotImportance[r.transactionID] = r.totalImportance
			gotDivision[r.transactionID] = r.division
		}
		xassert.Equal(t, map[int64]widget.Importance{
			buy.TransactionID:  widget.DangerImportance,
			sell.TransactionID: widget.SuccessImportance,
		}, gotImportance)
		xassert.Equal(t, map[int64]app.Division{
			buy.TransactionID:  app.Division1,
			sell.TransactionID: app.Division2,
		}, gotDivision)
	})
	t.Run("shows transactions when only one division has completed", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWalletTransactions(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCorporation()
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationWalletTransactions1,
		})
		o := factory.CreateCorporationWalletTransaction(storage.CreateCorporationWalletTransactionParams{
			CorporationID: c.ID,
			Division:      1,
		})
		// when
		a.u.Signals().CurrentCorporationExchanged.Emit(t.Context(), c)
		// then
		require.Len(t, a.rows, 1)
		xassert.Equal(t, o.TransactionID, a.rows[0].transactionID)
	})
	t.Run("shows no corporation when corporation is exchanged to none", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWalletTransactions(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		a.rows = []walletTransactionRow{{transactionID: 1}}
		// when
		a.u.Signals().CurrentCorporationExchanged.Emit(t.Context(), nil)
		// then
		xassert.Equal(t, "No corporation", a.footerLabel.Text)
		xassert.Equal(t, widget.LowImportance, a.footerLabel.Importance)
		assert.Empty(t, a.rows)
	})
	t.Run("shows no data when section was never updated", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWalletTransactions(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCorporation()
		a.rows = []walletTransactionRow{{transactionID: 1}}
		// when
		a.u.Signals().CurrentCorporationExchanged.Emit(t.Context(), c)
		// then
		xassert.Equal(t, "No data", a.footerLabel.Text)
		xassert.Equal(t, widget.WarningImportance, a.footerLabel.Importance)
		assert.Empty(t, a.rows)
	})
	t.Run("shows error when section status can not be loaded", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWalletTransactions(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCorporation()
		if _, err := db.Exec("DROP TABLE corporation_section_status"); err != nil {
			t.Fatal(err)
		}
		// when
		a.u.Signals().CurrentCorporationExchanged.Emit(t.Context(), c)
		// then
		assert.True(t, a.errorLabel.Visible())
		assert.Contains(t, a.errorLabel.Text, "ERROR:")
		assert.Empty(t, a.rows)
	})
	t.Run("shows error when transactions can not be loaded", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWalletTransactions(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCorporation()
		for _, d := range app.Divisions {
			factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
				CorporationID: c.ID,
				Section:       app.CorporationSectionWalletTransactions(d),
			})
		}
		if _, err := db.Exec("DROP TABLE corporation_wallet_transactions"); err != nil {
			t.Fatal(err)
		}
		// when
		a.u.Signals().CurrentCorporationExchanged.Emit(t.Context(), c)
		// then
		assert.True(t, a.errorLabel.Visible())
		assert.Contains(t, a.errorLabel.Text, "ERROR:")
		assert.Empty(t, a.rows)
	})
	t.Run("updates when transactions of division changed", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWalletTransactions(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCorporation()
		for _, d := range app.Divisions {
			factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
				CorporationID: c.ID,
				Section:       app.CorporationSectionWalletTransactions(d),
			})
		}
		factory.CreateCorporationWalletTransaction(storage.CreateCorporationWalletTransactionParams{
			CorporationID: c.ID,
			Division:      1,
		})
		a.corporation.Store(c)
		// when
		a.u.Signals().CorporationSectionChanged.Emit(t.Context(), app.CorporationSectionUpdated{
			CorporationID: c.ID,
			Section:       app.SectionCorporationWalletTransactions1,
		})
		// then
		assert.Len(t, a.rows, 1)
	})
	t.Run("ignores change of other section", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWalletTransactions(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCorporation()
		for _, d := range app.Divisions {
			factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
				CorporationID: c.ID,
				Section:       app.CorporationSectionWalletTransactions(d),
			})
		}
		factory.CreateCorporationWalletTransaction(storage.CreateCorporationWalletTransactionParams{
			CorporationID: c.ID,
			Division:      1,
		})
		a.corporation.Store(c)
		// when
		a.u.Signals().CorporationSectionChanged.Emit(t.Context(), app.CorporationSectionUpdated{
			CorporationID: c.ID,
			Section:       app.SectionCorporationWalletJournal1,
		})
		// then
		assert.Empty(t, a.rows)
	})
	t.Run("ignores change of other corporation", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWalletTransactions(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCorporation()
		for _, d := range app.Divisions {
			factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
				CorporationID: c.ID,
				Section:       app.CorporationSectionWalletTransactions(d),
			})
		}
		factory.CreateCorporationWalletTransaction(storage.CreateCorporationWalletTransactionParams{
			CorporationID: c.ID,
			Division:      1,
		})
		a.corporation.Store(c)
		other := factory.CreateCorporation()
		// when
		a.u.Signals().CorporationSectionChanged.Emit(t.Context(), app.CorporationSectionUpdated{
			CorporationID: other.ID,
			Section:       app.SectionCorporationWalletTransactions1,
		})
		// then
		assert.Empty(t, a.rows)
	})
	t.Run("updates division names when divisions changed", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWalletTransactions(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCorporation()
		for _, d := range app.Divisions {
			factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
				CorporationID: c.ID,
				Section:       app.CorporationSectionWalletTransactions(d),
			})
		}
		factory.CreateCorporationWalletTransaction(storage.CreateCorporationWalletTransactionParams{
			CorporationID: c.ID,
			Division:      2,
		})
		a.u.Signals().CurrentCorporationExchanged.Emit(t.Context(), c)
		factory.CreateCorporationWalletName(storage.UpdateOrCreateCorporationWalletNameParams{
			CorporationID: c.ID,
			Division:      2,
			Name:          "Ops",
		})
		// when
		a.u.Signals().CorporationSectionChanged.Emit(t.Context(), app.CorporationSectionUpdated{
			CorporationID: c.ID,
			Section:       app.SectionCorporationDivisions,
		})
		// then
		require.Len(t, a.rows, 1)
		xassert.Equal(t, "[2] Ops", a.rows[0].divisionName)
	})
}
