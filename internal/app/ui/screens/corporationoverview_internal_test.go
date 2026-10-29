package screens

import (
	"database/sql"
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestCorporationOverviewFilter_Match(t *testing.T) {
	r := corporationOverviewRow{
		alliance: optional.New(&app.EveEntity{Name: "Goonswarm"}),
		faction:  optional.New(&app.EveEntity{Name: "Caldari State"}),
	}
	for _, tc := range []struct {
		name   string
		filter corporationOverviewFilter
		want   bool
	}{
		{"no filter", corporationOverviewFilter{}, true},
		{"alliance matches", corporationOverviewFilter{alliance: "Goonswarm"}, true},
		{"alliance differs", corporationOverviewFilter{alliance: "Test"}, false},
		{"faction matches", corporationOverviewFilter{faction: "Caldari State"}, true},
		{"faction differs", corporationOverviewFilter{faction: "Amarr Empire"}, false},
		{"both match", corporationOverviewFilter{alliance: "Goonswarm", faction: "Caldari State"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.filter.match(r))
		})
	}
}

func TestCorporationOverview_Filter(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	rows := []corporationOverviewRow{
		{corporationID: 1, name: "Wayne Inc", alliance: optional.New(&app.EveEntity{Name: "Goonswarm"}), searchTarget: "wayne inc"},
		{corporationID: 2, name: "Acme", searchTarget: "acme"},
	}
	newOverview := func(t *testing.T, isMobile bool) *CorporationOverview {
		a := NewCorporationOverview(testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			IsMobile: isMobile,
			Storage:  st,
		}))
		a.rows = rows
		a.filterRowsAsync("")
		require.Len(t, a.rowsFiltered, 2)
		return a
	}
	corporationIDs := func(a *CorporationOverview) []int64 {
		return xslices.Map(a.rowsFiltered, func(r corporationOverviewRow) int64 {
			return r.corporationID
		})
	}
	t.Run("can filter on mobile", func(t *testing.T) {
		a := newOverview(t, true)
		a.filterChip.SetSelected(map[string]string{corporationOverviewFilterAlliance: "Goonswarm"})
		assert.ElementsMatch(t, []int64{1}, corporationIDs(a))
	})
	t.Run("can filter on desktop", func(t *testing.T) {
		a := newOverview(t, false)
		a.selectAlliance.SetSelected("Goonswarm")
		assert.ElementsMatch(t, []int64{1}, corporationIDs(a))
	})
	t.Run("can search on mobile", func(t *testing.T) {
		a := newOverview(t, true)
		a.searchEntry.SetText("acm")
		assert.ElementsMatch(t, []int64{2}, corporationIDs(a))
	})
	t.Run("shows all filters on mobile", func(t *testing.T) {
		a := newOverview(t, true)
		assert.Equal(t, map[string]string{
			corporationOverviewFilterAlliance: "",
			corporationOverviewFilterFaction:  "",
		}, a.filterChip.Selected())
	})
}

func TestCorporationOverview_Update(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	corporationIDs := func(a *CorporationOverview) set.Set[int64] {
		return set.Collect(xiter.MapSlice(a.rows, func(r corporationOverviewRow) int64 {
			return r.corporationID
		}))
	}
	t.Run("lists all corporations", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		a := NewCorporationOverview(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c1 := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c1.ID, app.CorporationSectionWalletJournal(app.Division1))
		c2 := factory.CreateCorporation()
		factory.CreateCorporationToken(c2.ID, set.Of[app.Role](), set.Of[string]())
		// when
		a.update(t.Context())
		// then
		xassert.Equal(t, set.Of(c1.ID, c2.ID), corporationIDs(a))
	})
	t.Run("does not list hidden corporations", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		u.Settings().SetHideLimitedCorporations(true)
		a := NewCorporationOverview(u)
		c1 := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c1.ID, app.CorporationSectionWalletJournal(app.Division1))
		c2 := factory.CreateCorporation()
		factory.CreateCorporationToken(c2.ID, set.Of[app.Role](), set.Of[string]())
		// when
		a.update(t.Context())
		// then
		xassert.Equal(t, set.Of(c1.ID), corporationIDs(a))
	})
	t.Run("removes corporation that lost privileged access on roles change", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		u.Settings().SetHideLimitedCorporations(true)
		a := NewCorporationOverview(u)
		character := factory.CreateCharacter()
		c1 := factory.CreateCorporation(character.EveCharacter.Corporation.ID)
		factory.SetCharacterRoles(character.ID, set.Of(app.RoleDirector))
		a.update(t.Context())
		require.Equal(t, set.Of(c1.ID), corporationIDs(a))
		factory.SetCharacterRoles(character.ID, set.Of[app.Role]())
		// when
		u.Signals().CharacterSectionChanged.Emit(t.Context(), app.CharacterSectionUpdated{
			CharacterID: character.ID,
			Section:     app.SectionCharacterRoles,
		})
		// then
		xassert.Equal(t, 0, corporationIDs(a).Size())
	})
}

func TestCorporationOverview_ErrorDisplay(t *testing.T) {
	newScreen := func(t *testing.T) (*CorporationOverview, *sql.DB) {
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		a := NewCorporationOverview(u)
		return a, db
	}
	t.Run("shows error when update fails and keeps it on filtering", func(t *testing.T) {
		a, db := newScreen(t)
		if _, err := db.Exec("DROP TABLE corporations"); err != nil { // let the next update fail
			t.Fatal(err)
		}

		a.update(t.Context())
		assert.True(t, a.errorLabel.Visible())
		assert.Contains(t, a.errorLabel.Text, "ERROR:")

		a.filterRowsAsync("")
		assert.True(t, a.errorLabel.Visible())
	})
	t.Run("hides error after successful update", func(t *testing.T) {
		a, _ := newScreen(t)
		a.errorLabel.SetText("ERROR: failed")
		a.errorLabel.Show()

		a.update(t.Context())
		assert.False(t, a.errorLabel.Visible())
	})
}
