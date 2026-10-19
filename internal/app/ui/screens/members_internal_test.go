package screens

import (
	"slices"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestMembers_ErrorDisplay(t *testing.T) {
	newMembers := func(t *testing.T, st *storage.Storage, factory testutil.Factory) *Members {
		a := NewMembers(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		ec := factory.CreateEveCharacter()
		a.corporation.Store(factory.CreateCorporation(ec.Corporation.ID))
		return a
	}
	t.Run("shows error when update fails and keeps it on filtering", func(t *testing.T) {
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := newMembers(t, st, factory)
		if _, err := db.Exec("DROP TABLE corporation_section_status"); err != nil { // let the next update fail
			t.Fatal(err)
		}

		a.update(t.Context())
		assert.True(t, a.errorLabel.Visible())
		assert.Contains(t, a.errorLabel.Text, "ERROR:")

		a.searchEntry.SetText("alpha")
		assert.True(t, a.errorLabel.Visible())
		assert.Contains(t, a.footerLabel.Text, "Showing")
	})
	t.Run("hides error after successful update", func(t *testing.T) {
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := newMembers(t, st, factory)
		a.errorLabel.SetText("ERROR: failed")
		a.errorLabel.Show()

		a.update(t.Context())
		assert.False(t, a.errorLabel.Visible())
	})
}

func TestMembers_ResetClearsFooter(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	a := NewMembers(testdouble.NewUIFake(testdouble.UIParams{
		App:     test.NewTempApp(t),
		Storage: st,
	}))
	a.rows = []memberRow{{id: 1, name: "Alpha"}, {id: 2, name: "Bravo"}}
	var pending []func() // run background work later, like the real app
	orig := runAsync
	runAsync = func(f func()) { pending = append(pending, f) }
	t.Cleanup(func() { runAsync = orig })
	a.filterRowsAsync() // a filter still running from before

	a.update(t.Context()) // no corporation
	for _, f := range pending {
		f()
	}

	assert.Equal(t, "", a.footerLabel.Text)
	assert.Empty(t, a.rowsFiltered)
}

func TestMembers_Update(t *testing.T) {
	t.Run("shows members sorted by name with CEO and owned characters", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewMembers(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCorporation()
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationMembers,
		})
		ceo := c.EveCorporation.Ceo.MustValue()
		factory.CreateCorporationMember(storage.CorporationMemberParams{CorporationID: c.ID, CharacterID: ceo.ID})
		mine := factory.CreateEveEntityCharacter(app.EveEntity{Name: "Alpha"})
		factory.CreateEveCharacter(storage.CreateEveCharacterParams{ID: mine.ID, CorporationID: c.ID, Name: mine.Name})
		factory.CreateCharacterFull(storage.CreateCharacterParams{ID: mine.ID})
		factory.CreateCorporationMember(storage.CorporationMemberParams{CorporationID: c.ID, CharacterID: mine.ID})
		other := factory.CreateEveEntityCharacter(app.EveEntity{Name: "Bravo"})
		factory.CreateCorporationMember(storage.CorporationMemberParams{CorporationID: c.ID, CharacterID: other.ID})
		// when
		a.u.Signals().CurrentCorporationExchanged.Emit(t.Context(), c)
		// then
		got := make(map[int64]memberRow)
		for _, r := range a.rows {
			r.searchTarget = ""
			got[r.id] = r
		}
		want := map[int64]memberRow{
			ceo.ID:   {id: ceo.ID, name: ceo.Name, isCEO: true},
			mine.ID:  {id: mine.ID, name: "Alpha", isOwned: true},
			other.ID: {id: other.ID, name: "Bravo"},
		}
		xassert.Equal(t, want, got)
		names := xslices.Map(a.rowsFiltered, func(r memberRow) string {
			return r.name
		})
		assert.True(t, slices.IsSorted(names))
		xassert.Equal(t, "Showing 3 / 3 members", a.footerLabel.Text)
	})
	t.Run("shows no members when section was never updated", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewMembers(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCorporation()
		factory.CreateCorporationMember(storage.CorporationMemberParams{CorporationID: c.ID})
		a.rows = []memberRow{{id: 1, name: "Alpha"}}
		// when
		a.u.Signals().CurrentCorporationExchanged.Emit(t.Context(), c)
		// then
		assert.Empty(t, a.rows)
		assert.Empty(t, a.footerLabel.Text)
	})
	t.Run("shows error when members can not be loaded", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewMembers(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCorporation()
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationMembers,
		})
		if _, err := db.Exec("DROP TABLE corporation_members"); err != nil {
			t.Fatal(err)
		}
		// when
		a.u.Signals().CurrentCorporationExchanged.Emit(t.Context(), c)
		// then
		assert.True(t, a.errorLabel.Visible())
		assert.Contains(t, a.errorLabel.Text, "ERROR:")
		assert.Empty(t, a.rows)
	})
}

func TestMembers_Signals(t *testing.T) {
	t.Run("updates when members of current corporation changed", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewMembers(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCorporation()
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationMembers,
		})
		factory.CreateCorporationMember(storage.CorporationMemberParams{CorporationID: c.ID})
		a.corporation.Store(c)
		// when
		a.u.Signals().CorporationSectionChanged.Emit(t.Context(), app.CorporationSectionUpdated{
			CorporationID: c.ID,
			Section:       app.SectionCorporationMembers,
		})
		// then
		assert.Len(t, a.rows, 1)
	})
	t.Run("ignores change of other corporation", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewMembers(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCorporation()
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationMembers,
		})
		factory.CreateCorporationMember(storage.CorporationMemberParams{CorporationID: c.ID})
		a.corporation.Store(c)
		other := factory.CreateCorporation()
		// when
		a.u.Signals().CorporationSectionChanged.Emit(t.Context(), app.CorporationSectionUpdated{
			CorporationID: other.ID,
			Section:       app.SectionCorporationMembers,
		})
		// then
		assert.Empty(t, a.rows)
	})
	t.Run("ignores change of other section", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewMembers(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCorporation()
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationMembers,
		})
		factory.CreateCorporationMember(storage.CorporationMemberParams{CorporationID: c.ID})
		a.corporation.Store(c)
		// when
		a.u.Signals().CorporationSectionChanged.Emit(t.Context(), app.CorporationSectionUpdated{
			CorporationID: c.ID,
			Section:       app.SectionCorporationAssets,
		})
		// then
		assert.Empty(t, a.rows)
	})
	t.Run("marks member as owned when character was added", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewMembers(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCorporation()
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationMembers,
		})
		ec := factory.CreateEveEntityCharacter()
		factory.CreateEveCharacter(storage.CreateEveCharacterParams{ID: ec.ID, CorporationID: c.ID, Name: ec.Name})
		factory.CreateCorporationMember(storage.CorporationMemberParams{CorporationID: c.ID, CharacterID: ec.ID})
		a.corporation.Store(c)
		a.update(t.Context())
		require.Len(t, a.rows, 1)
		require.False(t, a.rows[0].isOwned)
		character := factory.CreateCharacterFull(storage.CreateCharacterParams{ID: ec.ID})
		// when
		a.u.Signals().CharacterAdded.Emit(t.Context(), character)
		// then
		require.Len(t, a.rows, 1)
		assert.True(t, a.rows[0].isOwned)
	})
}

func TestMembers_Search(t *testing.T) {
	t.Run("filters members by name", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewMembers(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		a.rows = []memberRow{
			{id: 1, name: "Alpha", searchTarget: "alpha"},
			{id: 2, name: "Bravo", searchTarget: "bravo"},
		}
		a.filterRowsAsync()
		// when
		a.searchEntry.SetText("al")
		// then
		got := xslices.Map(a.rowsFiltered, func(r memberRow) int64 {
			return r.id
		})
		xassert.Equal(t, []int64{1}, got)
		xassert.Equal(t, "Showing 1 / 2 members", a.footerLabel.Text)
	})
	t.Run("ignores search with single character", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewMembers(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		a.rows = []memberRow{
			{id: 1, name: "Alpha", searchTarget: "alpha"},
			{id: 2, name: "Bravo", searchTarget: "bravo"},
		}
		a.filterRowsAsync()
		// when
		a.searchEntry.SetText("a")
		// then
		assert.Len(t, a.rowsFiltered, 2)
	})
}

func TestCorporationMemberItem_Set(t *testing.T) {
	cases := []struct {
		name      string
		row       memberRow
		wantOwned bool
		wantCEO   bool
	}{
		{"plain member", memberRow{id: 1, name: "Alpha"}, false, false},
		{"owned member", memberRow{id: 1, name: "Alpha", isOwned: true}, true, false},
		{"CEO", memberRow{id: 1, name: "Alpha", isCEO: true}, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			test.NewTempApp(t)
			w := newCorporationMemberItem(func(*app.EveEntity, int, func(fyne.Resource)) {})
			// when
			w.set(tc.row)
			// then
			xassert.Equal(t, tc.wantOwned, w.owned.Visible())
			xassert.Equal(t, tc.wantCEO, w.ceo.Visible())
		})
	}
}
