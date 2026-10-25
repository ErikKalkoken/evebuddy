package screens

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestWalletJournalFilter_Match(t *testing.T) {
	inflow := walletJournalRow{amount: optional.New(100.0), refTypeDisplay: "Bounty"}
	outflow := walletJournalRow{amount: optional.New(-50.0), refTypeDisplay: "Bounty"}
	unknown := walletJournalRow{refTypeDisplay: "Bounty"}
	for _, tc := range []struct {
		name   string
		filter walletJournalFilter
		row    walletJournalRow
		want   bool
	}{
		{"no filter", walletJournalFilter{}, inflow, true},
		{"type matches", walletJournalFilter{refType: "Bounty"}, inflow, true},
		{"type differs", walletJournalFilter{refType: "Market Transaction"}, inflow, false},
		{"inflow matches", walletJournalFilter{direction: walletJournalDirectionInflow}, inflow, true},
		{"inflow but outflow", walletJournalFilter{direction: walletJournalDirectionInflow}, outflow, false},
		{"outflow matches", walletJournalFilter{direction: walletJournalDirectionOutflow}, outflow, true},
		{"outflow but inflow", walletJournalFilter{direction: walletJournalDirectionOutflow}, inflow, false},
		{"inflow but amount unknown", walletJournalFilter{direction: walletJournalDirectionInflow}, unknown, false},
		{"outflow but amount unknown", walletJournalFilter{direction: walletJournalDirectionOutflow}, unknown, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.filter.match(tc.row))
		})
	}
}

func TestWalletJournalRow_SetSearchTarget(t *testing.T) {
	r := walletJournalRow{description: "Alpha deposited cash", reason: optional.New("For Fuel")}
	r.setSearchTarget()
	assert.Equal(t, "alpha deposited cash\nfor fuel", r.searchTarget)
}

func TestWalletJournal_Filter(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	rows := []walletJournalRow{
		{refID: 1, amount: optional.New(100.0), refTypeDisplay: "Bounty", searchTarget: "bounty prizes"},
		{refID: 2, amount: optional.New(-50.0), refTypeDisplay: "Market Transaction", searchTarget: "market escrow"},
	}
	newWalletJournal := func(t *testing.T, isMobile bool) *WalletJournal {
		a := NewCharacterWalletJournal(testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			IsMobile: isMobile,
			Storage:  st,
		}))
		a.rows = rows
		a.filterRowsAsync("")
		require.Len(t, a.rowsFiltered, 2)
		return a
	}
	refIDs := func(a *WalletJournal) []int64 {
		return xslices.Map(a.rowsFiltered, func(r walletJournalRow) int64 {
			return r.refID
		})
	}
	t.Run("can filter on mobile", func(t *testing.T) {
		a := newWalletJournal(t, true)
		a.filterChip.SetSelected(map[string]string{walletJournalFilterType: "Bounty"})
		assert.ElementsMatch(t, []int64{1}, refIDs(a))
	})
	t.Run("can filter on desktop", func(t *testing.T) {
		a := newWalletJournal(t, false)
		a.typeSelect.SetSelected("Bounty")
		assert.ElementsMatch(t, []int64{1}, refIDs(a))
	})
	t.Run("can filter direction on mobile", func(t *testing.T) {
		a := newWalletJournal(t, true)
		a.filterChip.SetSelected(map[string]string{walletJournalFilterDirection: walletJournalDirectionOutflow})
		assert.ElementsMatch(t, []int64{2}, refIDs(a))
	})
	t.Run("can filter direction on desktop", func(t *testing.T) {
		a := newWalletJournal(t, false)
		a.directionSelect.SetSelected(walletJournalDirectionOutflow)
		assert.ElementsMatch(t, []int64{2}, refIDs(a))
	})
	t.Run("can search", func(t *testing.T) {
		a := newWalletJournal(t, true)
		a.searchEntry.SetText("escrow")
		assert.ElementsMatch(t, []int64{2}, refIDs(a))
	})
}

func TestWalletJournal_KeepsStatusWithoutCharacter(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	a := NewCharacterWalletJournal(testdouble.NewUIFake(testdouble.UIParams{
		App:     test.NewTempApp(t),
		Storage: st,
	}))
	var pending []func() // run background work later, like the real app
	orig := runAsync
	runAsync = func(f func()) { pending = append(pending, f) }
	t.Cleanup(func() { runAsync = orig })
	a.filterRowsAsync("") // a filter still running from before

	a.updateRowsCharacter(t.Context())
	for _, f := range pending {
		f()
	}

	assert.Equal(t, "No character", a.footerLabel.Text)
	assert.Empty(t, a.rowsFiltered)
}

func TestWalletJournal_ErrorDisplay(t *testing.T) {
	newScreen := func(t *testing.T) (*WalletJournal, *sql.DB) {
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		a := NewCharacterWalletJournal(u)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletJournal,
		})
		a.character.Store(c)
		return a, db
	}
	t.Run("shows error when update fails and keeps it on filtering", func(t *testing.T) {
		a, db := newScreen(t)
		if _, err := db.Exec("DROP TABLE character_wallet_journal_entries"); err != nil { // let the next update fail
			t.Fatal(err)
		}

		a.updateRowsCharacter(t.Context())
		assert.True(t, a.errorLabel.Visible())
		assert.Contains(t, a.errorLabel.Text, "ERROR:")

		a.filterRowsAsync("")
		assert.True(t, a.errorLabel.Visible())
	})
	t.Run("hides error after successful update", func(t *testing.T) {
		a, _ := newScreen(t)
		a.errorLabel.SetText("ERROR: failed")
		a.errorLabel.Show()

		a.updateRowsCharacter(t.Context())
		assert.False(t, a.errorLabel.Visible())
	})
}

func TestWalletJournalRow_DescriptionWithReason(t *testing.T) {
	t.Run("shows description without reason", func(t *testing.T) {
		// given
		r := walletJournalRow{description: "Bounty prizes"}
		// when
		got := r.descriptionWithReason()
		// then
		xassert.Equal(t, "Bounty prizes", got)
	})
	t.Run("marks description with reason", func(t *testing.T) {
		// given
		r := walletJournalRow{description: "Player donation", reason: optional.New("thanks")}
		// when
		got := r.descriptionWithReason()
		// then
		xassert.Equal(t, "[r] Player donation", got)
	})
}

func TestColorISKAmount(t *testing.T) {
	cases := []struct {
		name   string
		amount optional.Optional[float64]
		want   fyne.ThemeColorName
	}{
		{"empty", optional.Optional[float64]{}, theme.ColorNameDisabled},
		{"negative", optional.New(-1.0), theme.ColorNameError},
		{"positive", optional.New(1.0), theme.ColorNameSuccess},
		{"zero", optional.New(0.0), theme.ColorNameForeground},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got := colorISKAmount(tc.amount)
			// then
			xassert.Equal(t, tc.want, got)
		})
	}
}

func TestImportanceISKAmount(t *testing.T) {
	cases := []struct {
		name   string
		amount optional.Optional[float64]
		want   widget.Importance
	}{
		{"empty", optional.Optional[float64]{}, widget.LowImportance},
		{"negative", optional.New(-1.0), widget.DangerImportance},
		{"positive", optional.New(1.0), widget.SuccessImportance},
		{"zero", optional.New(0.0), widget.MediumImportance},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got := importanceISKAmount(tc.amount)
			// then
			xassert.Equal(t, tc.want, got)
		})
	}
}

func TestCharacterWalletJournal_Signals(t *testing.T) {
	t.Run("updates when wallet journal of current character changed", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterWalletJournal(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCharacterFull()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletJournal,
		})
		factory.CreateCharacterWalletJournalEntry(storage.CreateCharacterWalletJournalEntryParams{CharacterID: c.ID})
		a.character.Store(c)
		// when
		a.u.Signals().CharacterSectionChanged.Emit(t.Context(), app.CharacterSectionUpdated{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletJournal,
		})
		// then
		assert.Len(t, a.rows, 1)
	})
	t.Run("ignores change of other character", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterWalletJournal(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCharacterFull()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletJournal,
		})
		factory.CreateCharacterWalletJournalEntry(storage.CreateCharacterWalletJournalEntryParams{CharacterID: c.ID})
		a.character.Store(c)
		other := factory.CreateCharacterFull()
		// when
		a.u.Signals().CharacterSectionChanged.Emit(t.Context(), app.CharacterSectionUpdated{
			CharacterID: other.ID,
			Section:     app.SectionCharacterWalletJournal,
		})
		// then
		assert.Empty(t, a.rows)
	})
}

func TestCorporationWalletJournal(t *testing.T) {
	t.Run("shows entries of master wallet by default", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWalletJournal(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletJournal1)
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationWalletJournal1,
		})
		e := factory.CreateCorporationWalletJournalEntry(storage.CreateCorporationWalletJournalEntryParams{
			CorporationID: c.ID,
			Division:      1,
		})
		factory.CreateCorporationWalletJournalEntry(storage.CreateCorporationWalletJournalEntryParams{
			CorporationID: c.ID,
			Division:      2,
		})
		a.corporation.Store(c)
		// when
		a.updateRowsCorporation(t.Context())
		// then
		require.Len(t, a.rows, 1)
		xassert.Equal(t, e.RefID, a.rows[0].refID)
		xassert.Equal(t, app.Division1, a.rows[0].division)
		xassert.Equal(t, "Showing 1 / 1 entries", a.footerLabel.Text)
	})
	t.Run("shows entries of selected division", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWalletJournal(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletJournal2)
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationWalletJournal2,
		})
		e := factory.CreateCorporationWalletJournalEntry(storage.CreateCorporationWalletJournalEntryParams{
			CorporationID: c.ID,
			Division:      2,
		})
		factory.CreateCorporationWalletJournalEntry(storage.CreateCorporationWalletJournalEntryParams{
			CorporationID: c.ID,
			Division:      1,
		})
		a.corporation.Store(c)
		a.division.Store(app.Division2)
		// when
		a.updateRowsCorporation(t.Context())
		// then
		require.Len(t, a.rows, 1)
		xassert.Equal(t, e.RefID, a.rows[0].refID)
		xassert.Equal(t, app.Division2, a.rows[0].division)
		xassert.Equal(t, "Showing 1 / 1 entries", a.footerLabel.Text)
	})
	t.Run("shows no corporation without corporation", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWalletJournal(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		a.rows = []walletJournalRow{{refID: 1}}
		// when
		a.updateRowsCorporation(t.Context())
		// then
		xassert.Equal(t, "No corporation", a.footerLabel.Text)
		xassert.Equal(t, widget.LowImportance, a.footerLabel.Importance)
		assert.Empty(t, a.rows)
	})
	t.Run("shows no data when section was never updated", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWalletJournal(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCorporation()
		a.rows = []walletJournalRow{{refID: 1}}
		a.corporation.Store(c)
		// when
		a.updateRowsCorporation(t.Context())
		// then
		xassert.Equal(t, "No data", a.footerLabel.Text)
		xassert.Equal(t, widget.WarningImportance, a.footerLabel.Importance)
		assert.Empty(t, a.rows)
	})
	t.Run("shows error when section status can not be loaded", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWalletJournal(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCorporation()
		if _, err := db.Exec("DROP TABLE corporation_section_status"); err != nil {
			t.Fatal(err)
		}
		a.corporation.Store(c)
		// when
		a.updateRowsCorporation(t.Context())
		// then
		assert.True(t, a.errorLabel.Visible())
		assert.Contains(t, a.errorLabel.Text, "ERROR:")
		assert.Empty(t, a.rows)
	})
	t.Run("shows error when entries can not be loaded", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWalletJournal(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletJournal1)
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationWalletJournal1,
		})
		if _, err := db.Exec("DROP TABLE corporation_wallet_journal_entries"); err != nil {
			t.Fatal(err)
		}
		a.corporation.Store(c)
		// when
		a.updateRowsCorporation(t.Context())
		// then
		assert.True(t, a.errorLabel.Visible())
		assert.Contains(t, a.errorLabel.Text, "ERROR:")
		assert.Empty(t, a.rows)
	})
	t.Run("updates when wallet journal of division changed", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWalletJournal(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletJournal1)
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationWalletJournal1,
		})
		factory.CreateCorporationWalletJournalEntry(storage.CreateCorporationWalletJournalEntryParams{
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
		assert.Len(t, a.rows, 1)
	})
	t.Run("ignores change of other division", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWalletJournal(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletJournal1)
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationWalletJournal1,
		})
		factory.CreateCorporationWalletJournalEntry(storage.CreateCorporationWalletJournalEntryParams{
			CorporationID: c.ID,
			Division:      1,
		})
		a.corporation.Store(c)
		// when
		a.u.Signals().CorporationSectionChanged.Emit(t.Context(), app.CorporationSectionUpdated{
			CorporationID: c.ID,
			Section:       app.SectionCorporationWalletJournal2,
		})
		// then
		assert.Empty(t, a.rows)
	})
	t.Run("ignores change of other corporation", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWalletJournal(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletJournal1)
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationWalletJournal1,
		})
		factory.CreateCorporationWalletJournalEntry(storage.CreateCorporationWalletJournalEntryParams{
			CorporationID: c.ID,
			Division:      1,
		})
		a.corporation.Store(c)
		other := factory.CreateCorporation()
		// when
		a.u.Signals().CorporationSectionChanged.Emit(t.Context(), app.CorporationSectionUpdated{
			CorporationID: other.ID,
			Section:       app.SectionCorporationWalletJournal1,
		})
		// then
		assert.Empty(t, a.rows)
	})
}

func TestWalletJournal_UpdateBalanceCharacter(t *testing.T) {
	t.Run("shows and reports balance", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterWalletJournal(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		var reported []optional.Optional[float64]
		a.onBalanceUpdate = func(b optional.Optional[float64]) {
			reported = append(reported, b)
		}
		c := factory.CreateCharacterFull(storage.CreateCharacterParams{
			WalletBalance: optional.New(1234567.89),
		})
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletBalance,
		})
		a.character.Store(c)
		// when
		a.updateBalanceCharacter(t.Context())
		// then
		xassert.Equal(t, "1,234,567.89 ISK", a.balanceLabel.Text)
		xassert.Equal(t, widget.MediumImportance, a.balanceLabel.Importance)
		xassert.Equal(t, []optional.Optional[float64]{optional.New(1234567.89)}, reported)
	})
	t.Run("shows nothing and reports empty balance without character", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterWalletJournal(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		var reported []optional.Optional[float64]
		a.onBalanceUpdate = func(b optional.Optional[float64]) {
			reported = append(reported, b)
		}
		a.balanceLabel.SetText("1,000 ISK")
		// when
		a.updateBalanceCharacter(t.Context())
		// then
		assert.Empty(t, a.balanceLabel.Text)
		xassert.Equal(t, []optional.Optional[float64]{{}}, reported)
	})
	t.Run("shows no data when section was never updated", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterWalletJournal(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		var reported []optional.Optional[float64]
		a.onBalanceUpdate = func(b optional.Optional[float64]) {
			reported = append(reported, b)
		}
		a.character.Store(factory.CreateCharacterFull())
		// when
		a.updateBalanceCharacter(t.Context())
		// then
		xassert.Equal(t, "No data", a.balanceLabel.Text)
		xassert.Equal(t, widget.WarningImportance, a.balanceLabel.Importance)
		xassert.Equal(t, []optional.Optional[float64]{{}}, reported)
	})
	t.Run("shows no data when character has no wallet balance", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterWalletJournal(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		var reported []optional.Optional[float64]
		a.onBalanceUpdate = func(b optional.Optional[float64]) {
			reported = append(reported, b)
		}
		c := factory.CreateCharacter()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletBalance,
		})
		a.character.Store(c)
		// when
		a.updateBalanceCharacter(t.Context())
		// then
		xassert.Equal(t, "No data", a.balanceLabel.Text)
		xassert.Equal(t, widget.WarningImportance, a.balanceLabel.Importance)
		xassert.Equal(t, []optional.Optional[float64]{{}}, reported)
	})
	t.Run("shows error when section status can not be loaded", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterWalletJournal(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		var reported []optional.Optional[float64]
		a.onBalanceUpdate = func(b optional.Optional[float64]) {
			reported = append(reported, b)
		}
		a.character.Store(factory.CreateCharacterFull())
		if _, err := db.Exec("DROP TABLE character_section_status"); err != nil {
			t.Fatal(err)
		}
		// when
		a.updateBalanceCharacter(t.Context())
		// then
		assert.Contains(t, a.balanceLabel.Text, "ERROR:")
		xassert.Equal(t, widget.DangerImportance, a.balanceLabel.Importance)
		xassert.Equal(t, []optional.Optional[float64]{{}}, reported)
	})
	t.Run("keeps balance when update is canceled", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterWalletJournal(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		var reported []optional.Optional[float64]
		a.onBalanceUpdate = func(b optional.Optional[float64]) {
			reported = append(reported, b)
		}
		a.character.Store(factory.CreateCharacterFull())
		a.balanceLabel.SetText("1,000 ISK")
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		// when
		a.updateBalanceCharacter(ctx)
		// then
		xassert.Equal(t, "1,000 ISK", a.balanceLabel.Text)
		assert.Empty(t, reported)
	})
	t.Run("updates balance when wallet balance of current character changed", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterWalletJournal(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCharacterFull(storage.CreateCharacterParams{
			WalletBalance: optional.New(1000.0),
		})
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletBalance,
		})
		a.character.Store(c)
		// when
		a.u.Signals().CharacterSectionChanged.Emit(t.Context(), app.CharacterSectionUpdated{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletBalance,
		})
		// then
		xassert.Equal(t, "1,000.00 ISK", a.balanceLabel.Text)
	})
	t.Run("ignores balance change of other character", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterWalletJournal(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCharacterFull(storage.CreateCharacterParams{
			WalletBalance: optional.New(1000.0),
		})
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletBalance,
		})
		a.character.Store(c)
		other := factory.CreateCharacterFull()
		// when
		a.u.Signals().CharacterSectionChanged.Emit(t.Context(), app.CharacterSectionUpdated{
			CharacterID: other.ID,
			Section:     app.SectionCharacterWalletBalance,
		})
		// then
		assert.NotEqual(t, "1,000.00 ISK", a.balanceLabel.Text)
	})
	t.Run("ignores change of other section", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCharacterWalletJournal(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCharacterFull(storage.CreateCharacterParams{
			WalletBalance: optional.New(1000.0),
		})
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletBalance,
		})
		a.character.Store(c)
		// when
		a.u.Signals().CharacterSectionChanged.Emit(t.Context(), app.CharacterSectionUpdated{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletJournal,
		})
		// then
		assert.NotEqual(t, "1,000.00 ISK", a.balanceLabel.Text)
	})
}

func TestWalletJournal_UpdateBalanceCorporation(t *testing.T) {
	t.Run("shows and reports balance of master wallet", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWalletJournal(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		var reported []optional.Optional[float64]
		a.onBalanceUpdate = func(b optional.Optional[float64]) {
			reported = append(reported, b)
		}
		c := createCorporationWithWalletBalances(factory)
		a.corporation.Store(c)
		// when
		a.updateBalanceCorporation(t.Context())
		// then
		xassert.Equal(t, "1,000.00 ISK", a.balanceLabel.Text)
		xassert.Equal(t, widget.MediumImportance, a.balanceLabel.Importance)
		xassert.Equal(t, []optional.Optional[float64]{optional.New(1000.0)}, reported)
	})
	t.Run("shows balance of selected division", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWalletJournal(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		a.corporation.Store(createCorporationWithWalletBalances(factory))
		a.division.Store(app.Division2)
		// when
		a.updateBalanceCorporation(t.Context())
		// then
		xassert.Equal(t, "234.50 ISK", a.balanceLabel.Text)
	})
	t.Run("shows nothing and reports empty balance without corporation", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWalletJournal(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		var reported []optional.Optional[float64]
		a.onBalanceUpdate = func(b optional.Optional[float64]) {
			reported = append(reported, b)
		}
		a.balanceLabel.SetText("1,000 ISK")
		// when
		a.updateBalanceCorporation(t.Context())
		// then
		assert.Empty(t, a.balanceLabel.Text)
		xassert.Equal(t, []optional.Optional[float64]{{}}, reported)
	})
	t.Run("shows no data when section was never updated", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWalletJournal(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		var reported []optional.Optional[float64]
		a.onBalanceUpdate = func(b optional.Optional[float64]) {
			reported = append(reported, b)
		}
		c := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletBalances)
		a.corporation.Store(c)
		// when
		a.updateBalanceCorporation(t.Context())
		// then
		xassert.Equal(t, "No data", a.balanceLabel.Text)
		xassert.Equal(t, widget.WarningImportance, a.balanceLabel.Importance)
		xassert.Equal(t, []optional.Optional[float64]{{}}, reported)
	})
	t.Run("shows no data without permission", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWalletJournal(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		var reported []optional.Optional[float64]
		a.onBalanceUpdate = func(b optional.Optional[float64]) {
			reported = append(reported, b)
		}
		c := factory.CreateCorporation()
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationWalletBalances,
		})
		factory.CreateCorporationWalletBalance(storage.UpdateOrCreateCorporationWalletBalanceParams{
			CorporationID: c.ID,
			Division:      1,
			Balance:       1000,
		})
		a.corporation.Store(c)
		// when
		a.updateBalanceCorporation(t.Context())
		// then
		xassert.Equal(t, "No data", a.balanceLabel.Text)
		xassert.Equal(t, []optional.Optional[float64]{{}}, reported)
	})
	t.Run("shows error when section status can not be loaded", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWalletJournal(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		var reported []optional.Optional[float64]
		a.onBalanceUpdate = func(b optional.Optional[float64]) {
			reported = append(reported, b)
		}
		a.corporation.Store(factory.CreateCorporation())
		if _, err := db.Exec("DROP TABLE corporation_section_status"); err != nil {
			t.Fatal(err)
		}
		// when
		a.updateBalanceCorporation(t.Context())
		// then
		assert.Contains(t, a.balanceLabel.Text, "ERROR:")
		xassert.Equal(t, widget.DangerImportance, a.balanceLabel.Importance)
		xassert.Equal(t, []optional.Optional[float64]{{}}, reported)
	})
	t.Run("updates balance when wallet balances of current corporation changed", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWalletJournal(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := createCorporationWithWalletBalances(factory)
		a.corporation.Store(c)
		// when
		a.u.Signals().CorporationSectionChanged.Emit(t.Context(), app.CorporationSectionUpdated{
			CorporationID: c.ID,
			Section:       app.SectionCorporationWalletBalances,
		})
		// then
		xassert.Equal(t, "1,000.00 ISK", a.balanceLabel.Text)
	})
	t.Run("ignores balance change of other corporation", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWalletJournal(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		a.corporation.Store(createCorporationWithWalletBalances(factory))
		other := factory.CreateCorporation()
		// when
		a.u.Signals().CorporationSectionChanged.Emit(t.Context(), app.CorporationSectionUpdated{
			CorporationID: other.ID,
			Section:       app.SectionCorporationWalletBalances,
		})
		// then
		assert.NotEqual(t, "1,000.00 ISK", a.balanceLabel.Text)
	})
}

func TestDivisionSelect(t *testing.T) {
	t.Run("selects master wallet by default", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		// when
		w := NewDivisionSelect(nil)
		// then
		xassert.Equal(t, "[1] Master Wallet", w.Selected)
		assert.Len(t, w.Options, len(app.Divisions))
	})
	t.Run("reports division selected by user", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		var got []app.Division
		w := NewDivisionSelect(func(d app.Division) {
			got = append(got, d)
		})
		// when
		w.SetSelectedIndex(2)
		// then
		xassert.Equal(t, []app.Division{app.Division3}, got)
	})
	t.Run("can select division without reporting it", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		var got []app.Division
		w := NewDivisionSelect(func(d app.Division) {
			got = append(got, d)
		})
		// when
		w.SetDivisionSilent(app.Division2)
		// then
		xassert.Equal(t, "[2] 2nd Wallet", w.Selected)
		assert.Empty(t, got)
	})
	t.Run("keeps division selected by user when names are set", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		w := NewDivisionSelect(nil)
		w.SetSelectedIndex(2)
		// when
		w.SetNames(map[app.Division]string{app.Division3: "Alpha"})
		// then
		xassert.Equal(t, "[3] Alpha", w.Selected)
	})
	t.Run("shows default name for divisions without name", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		w := NewDivisionSelect(nil)
		// when
		w.SetNames(map[app.Division]string{app.Division3: "Alpha"})
		// then
		xassert.Equal(t, "[1] Master Wallet", w.Selected)
		xassert.Equal(t, "[3] Alpha", w.Options[2])
	})
	t.Run("resets names to defaults when names are nil", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		w := NewDivisionSelect(nil)
		w.SetNames(map[app.Division]string{app.Division1: "Alpha", app.Division3: "Bravo"})
		// when
		w.SetNames(nil)
		// then
		xassert.Equal(t, "[1] Master Wallet", w.Selected)
		xassert.Equal(t, "[1] Master Wallet", w.Options[0])
		xassert.Equal(t, "[3] "+app.Division3.DefaultWalletName(), w.Options[2])
	})
	t.Run("reports division selected by user after names are reset", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		var got []app.Division
		w := NewDivisionSelect(func(d app.Division) {
			got = append(got, d)
		})
		w.SetNames(map[app.Division]string{app.Division3: "Bravo"})
		w.SetNames(nil)
		// when
		w.SetSelectedIndex(2)
		// then
		xassert.Equal(t, []app.Division{app.Division3}, got)
	})
}

func TestWriteJournalRowsToCSV(t *testing.T) {
	header := []string{
		"ID",
		"Date",
		"Type",
		"Amount",
		"Balance",
		"Description",
		"First Party ID",
		"First Party Name",
		"Second Party ID",
		"Second Party Name",
		"Tax Receiver ID",
		"Tax Receiver Name",
		"Reason",
	}
	t.Run("writes header and all fields", func(t *testing.T) {
		// given
		rows := []walletJournalRow{{
			amount:      optional.New(-1234.5),
			balance:     optional.New(1000000.0),
			date:        time.Date(2026, 10, 24, 12, 30, 0, 0, time.UTC),
			description: "Bruce paid Alfred",
			firstParty:  optional.New(&app.EveEntity{ID: 1, Name: "Bruce"}),
			reason:      optional.New("Fuel, ammo"),
			refID:       42,
			refType:     "player_donation",
			secondParty: optional.New(&app.EveEntity{ID: 2, Name: "Alfred"}),
			taxReceiver: optional.New(&app.EveEntity{ID: 3, Name: "Wayne Enterprises"}),
		}}
		var buf bytes.Buffer
		// when
		err := writeJournalRowsToCSV(&buf, rows)
		// then
		require.NoError(t, err)
		got, err := csv.NewReader(&buf).ReadAll()
		require.NoError(t, err)
		xassert.Equal(t, [][]string{header, {
			"42",
			"2026-10-24T12:30:00Z",
			"player_donation",
			"-1234.5",
			"1000000",
			"Bruce paid Alfred",
			"1",
			"Bruce",
			"2",
			"Alfred",
			"3",
			"Wayne Enterprises",
			"Fuel, ammo",
		}}, got)
	})
	t.Run("writes empty fields for missing values", func(t *testing.T) {
		// given
		rows := []walletJournalRow{{
			date:    time.Date(2026, 10, 24, 12, 30, 0, 0, time.UTC),
			refID:   42,
			refType: "bounty_prizes",
		}}
		var buf bytes.Buffer
		// when
		err := writeJournalRowsToCSV(&buf, rows)
		// then
		require.NoError(t, err)
		got, err := csv.NewReader(&buf).ReadAll()
		require.NoError(t, err)
		xassert.Equal(t, [][]string{header, {
			"42",
			"2026-10-24T12:30:00Z",
			"bounty_prizes",
			"",
			"",
			"",
			"0",
			"",
			"0",
			"",
			"0",
			"",
			"",
		}}, got)
	})
	t.Run("writes only header when there are no rows", func(t *testing.T) {
		// given
		var buf bytes.Buffer
		// when
		err := writeJournalRowsToCSV(&buf, nil)
		// then
		require.NoError(t, err)
		got, err := csv.NewReader(&buf).ReadAll()
		require.NoError(t, err)
		xassert.Equal(t, [][]string{header}, got)
	})
}
