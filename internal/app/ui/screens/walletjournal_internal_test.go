package screens

import (
	"database/sql"
	"fmt"
	"testing"

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

// TODO: Add tests for export to CSV

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
		a.selectType.SetSelected("Bounty")
		assert.ElementsMatch(t, []int64{1}, refIDs(a))
	})
	t.Run("can filter direction on mobile", func(t *testing.T) {
		a := newWalletJournal(t, true)
		a.filterChip.SetSelected(map[string]string{walletJournalFilterDirection: walletJournalDirectionOutflow})
		assert.ElementsMatch(t, []int64{2}, refIDs(a))
	})
	t.Run("can filter direction on desktop", func(t *testing.T) {
		a := newWalletJournal(t, false)
		a.selectDirection.SetSelected(walletJournalDirectionOutflow)
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

	a.Update(t.Context())
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

		a.Update(t.Context())
		assert.True(t, a.errorLabel.Visible())
		assert.Contains(t, a.errorLabel.Text, "ERROR:")

		a.filterRowsAsync("")
		assert.True(t, a.errorLabel.Visible())
	})
	t.Run("hides error after successful update", func(t *testing.T) {
		a, _ := newScreen(t)
		a.errorLabel.SetText("ERROR: failed")
		a.errorLabel.Show()

		a.Update(t.Context())
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
	t.Run("shows entries of division when corporation is exchanged", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWalletJournal(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}), app.Division2)
		c := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletJournal2)
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationWalletJournal2,
		})
		e := factory.CreateCorporationWalletJournalEntry(storage.CreateCorporationWalletJournalEntryParams{
			CorporationID: c.ID,
			DivisionID:    2,
		})
		factory.CreateCorporationWalletJournalEntry(storage.CreateCorporationWalletJournalEntryParams{
			CorporationID: c.ID,
			DivisionID:    1,
		})
		// when
		a.u.Signals().CurrentCorporationExchanged.Emit(t.Context(), c)
		// then
		require.Len(t, a.rows, 1)
		xassert.Equal(t, e.RefID, a.rows[0].refID)
		assert.True(t, a.rows[0].forCorporation)
		xassert.Equal(t, app.Division2, a.rows[0].division)
		xassert.Equal(t, "Showing 1 / 1 entries", a.footerLabel.Text)
	})
	t.Run("shows no corporation when corporation is exchanged to none", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWalletJournal(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}), app.Division1)
		a.rows = []walletJournalRow{{refID: 1}}
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
		a := NewCorporationWalletJournal(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}), app.Division1)
		c := factory.CreateCorporation()
		a.rows = []walletJournalRow{{refID: 1}}
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
		a := NewCorporationWalletJournal(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}), app.Division1)
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
	t.Run("shows error when entries can not be loaded", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWalletJournal(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}), app.Division1)
		c := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletJournal1)
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationWalletJournal1,
		})
		if _, err := db.Exec("DROP TABLE corporation_wallet_journal_entries"); err != nil {
			t.Fatal(err)
		}
		// when
		a.u.Signals().CurrentCorporationExchanged.Emit(t.Context(), c)
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
		}), app.Division1)
		c := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletJournal1)
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationWalletJournal1,
		})
		factory.CreateCorporationWalletJournalEntry(storage.CreateCorporationWalletJournalEntryParams{
			CorporationID: c.ID,
			DivisionID:    1,
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
		}), app.Division1)
		c := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletJournal1)
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationWalletJournal1,
		})
		factory.CreateCorporationWalletJournalEntry(storage.CreateCorporationWalletJournalEntryParams{
			CorporationID: c.ID,
			DivisionID:    1,
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
		}), app.Division1)
		c := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletJournal1)
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationWalletJournal1,
		})
		factory.CreateCorporationWalletJournalEntry(storage.CreateCorporationWalletJournalEntryParams{
			CorporationID: c.ID,
			DivisionID:    1,
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

func TestShowCharacterWalletJournalEntryWindowAsync(t *testing.T) {
	t.Run("can show entry", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		e := factory.CreateCharacterWalletJournalEntry(storage.CreateCharacterWalletJournalEntryParams{
			FirstPartyID:  optional.New(factory.CreateEveEntityCharacter().ID),
			SecondPartyID: optional.New(factory.CreateEveEntityCharacter().ID),
			Tax:           optional.New(10.0),
			TaxReceiverID: optional.New(factory.CreateEveEntityCorporation().ID),
		})
		// when
		ShowCharacterWalletJournalEntryWindowAsync(u, e.CharacterID, "Bruce", e.RefID)
		// then
		w, created := u.GetOrCreateWindow(fmt.Sprintf("walletjournalentry-%d-%d", e.CharacterID, e.RefID))
		assert.False(t, created)
		assert.NotNil(t, w.Content())
	})
	t.Run("can show entry with related entity", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		e := factory.CreateCharacterWalletJournalEntry(storage.CreateCharacterWalletJournalEntryParams{
			ContextID:     optional.New(factory.CreateEveEntityCharacter().ID),
			ContextIDType: optional.New("character_id"),
		})
		// when
		ShowCharacterWalletJournalEntryWindowAsync(u, e.CharacterID, "Bruce", e.RefID)
		// then
		w, created := u.GetOrCreateWindow(fmt.Sprintf("walletjournalentry-%d-%d", e.CharacterID, e.RefID))
		assert.False(t, created)
		assert.NotNil(t, w.Content())
	})
	t.Run("can show entry with related contract", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		c := factory.CreateCharacterFull()
		o := factory.CreateCharacterContract(storage.CreateCharacterContractParams{CharacterID: c.ID})
		e := factory.CreateCharacterWalletJournalEntry(storage.CreateCharacterWalletJournalEntryParams{
			CharacterID:   c.ID,
			ContextID:     optional.New(o.ContractID),
			ContextIDType: optional.New("contract_id"),
		})
		// when
		ShowCharacterWalletJournalEntryWindowAsync(u, e.CharacterID, "Bruce", e.RefID)
		// then
		w, created := u.GetOrCreateWindow(fmt.Sprintf("walletjournalentry-%d-%d", e.CharacterID, e.RefID))
		assert.False(t, created)
		assert.NotNil(t, w.Content())
	})
	t.Run("can show entry with related market transaction", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		e := factory.CreateCharacterWalletJournalEntry(storage.CreateCharacterWalletJournalEntryParams{
			ContextID:     optional.New[int64](42),
			ContextIDType: optional.New("market_transaction_id"),
		})
		// when
		ShowCharacterWalletJournalEntryWindowAsync(u, e.CharacterID, "Bruce", e.RefID)
		// then
		w, created := u.GetOrCreateWindow(fmt.Sprintf("walletjournalentry-%d-%d", e.CharacterID, e.RefID))
		assert.False(t, created)
		assert.NotNil(t, w.Content())
	})
	t.Run("closes window when entry can not be loaded", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		c := factory.CreateCharacterFull()
		// when
		ShowCharacterWalletJournalEntryWindowAsync(u, c.ID, "Bruce", 42)
		// then
		_, created := u.GetOrCreateWindow(fmt.Sprintf("walletjournalentry-%d-%d", c.ID, 42))
		assert.True(t, created)
	})
	t.Run("reuses existing window", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		w, _ := u.GetOrCreateWindow(fmt.Sprintf("walletjournalentry-%d-%d", 7, 42))
		// when
		ShowCharacterWalletJournalEntryWindowAsync(u, 7, "Bruce", 42)
		// then
		assert.Nil(t, w.Content())
	})
}

func TestShowCorporationWalletJournalEntryWindowAsync(t *testing.T) {
	t.Run("can show entry", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		c := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletJournal1)
		e := factory.CreateCorporationWalletJournalEntry(storage.CreateCorporationWalletJournalEntryParams{
			CorporationID: c.ID,
			DivisionID:    1,
			FirstPartyID:  optional.New(factory.CreateEveEntityCharacter().ID),
			SecondPartyID: optional.New(factory.CreateEveEntityCharacter().ID),
			Tax:           optional.New(10.0),
			TaxReceiverID: optional.New(factory.CreateEveEntityCorporation().ID),
		})
		// when
		ShowCorporationWalletJournalEntryWindowAsync(u, e.CorporationID, "Wayne Enterprises", app.Division1, e.RefID)
		// then
		w, created := u.GetOrCreateWindow(fmt.Sprintf("walletjournalentry-%d-%d", e.CorporationID, e.RefID))
		assert.False(t, created)
		assert.NotNil(t, w.Content())
	})
	t.Run("closes window when entry can not be loaded", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		c := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletJournal1)
		// when
		ShowCorporationWalletJournalEntryWindowAsync(u, c.ID, "Wayne Enterprises", app.Division1, 42)
		// then
		_, created := u.GetOrCreateWindow(fmt.Sprintf("walletjournalentry-%d-%d", c.ID, 42))
		assert.True(t, created)
	})
	t.Run("reuses existing window", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		w, _ := u.GetOrCreateWindow(fmt.Sprintf("walletjournalentry-%d-%d", 7, 42))
		// when
		ShowCorporationWalletJournalEntryWindowAsync(u, 7, "Wayne Enterprises", app.Division1, 42)
		// then
		assert.Nil(t, w.Content())
	})
}
