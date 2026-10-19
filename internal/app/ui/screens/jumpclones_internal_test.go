package screens

import (
	"database/sql"
	"fmt"
	"slices"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func makeJumpCloneRow(id int64, character, system, region string, tags ...string) jumpCloneRow {
	return jumpCloneRow{
		jc: &app.CharacterJumpClone2{
			ID:        id,
			Character: &app.EntityShort{Name: character},
			Location: &app.EveLocation{
				SolarSystem: optional.New(&app.EveSolarSystem{
					Name:          system,
					Constellation: &app.EveConstellation{Region: &app.EveRegion{Name: region}},
				}),
			},
		},
		tags: set.Of(tags...),
	}
}

func TestJumpClonesFilter_Match(t *testing.T) {
	r := makeJumpCloneRow(1, "Bruce", "Jita", "The Forge", "alpha")
	for _, tc := range []struct {
		name   string
		filter jumpClonesFilter
		want   bool
	}{
		{"no filter", jumpClonesFilter{}, true},
		{"character matches", jumpClonesFilter{character: "Bruce"}, true},
		{"character differs", jumpClonesFilter{character: "Alice"}, false},
		{"region matches", jumpClonesFilter{region: "The Forge"}, true},
		{"region differs", jumpClonesFilter{region: "Domain"}, false},
		{"system matches", jumpClonesFilter{solarSystem: "Jita"}, true},
		{"system differs", jumpClonesFilter{solarSystem: "Amarr"}, false},
		{"tag matches", jumpClonesFilter{tag: "alpha"}, true},
		{"tag missing", jumpClonesFilter{tag: "bravo"}, false},
		{"all match", jumpClonesFilter{character: "Bruce", region: "The Forge", solarSystem: "Jita", tag: "alpha"}, true},
		{"one of many differs", jumpClonesFilter{character: "Bruce", solarSystem: "Amarr"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.filter.match(r))
		})
	}
}

func TestJumpClones_Filter(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	rows := []jumpCloneRow{
		makeJumpCloneRow(1, "Bruce", "Jita", "The Forge"),
		makeJumpCloneRow(2, "Alice", "Amarr", "Domain"),
	}
	newJumpClones := func(t *testing.T, isMobile bool) *JumpClones {
		a := NewJumpClones(testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			IsMobile: isMobile,
			Storage:  st,
		}))
		a.rows = rows
		a.filterRowsAsync("")
		require.Len(t, a.rowsFiltered, 2)
		return a
	}
	cloneIDs := func(a *JumpClones) []int64 {
		return xslices.Map(a.rowsFiltered, func(r jumpCloneRow) int64 {
			return r.jc.ID
		})
	}
	t.Run("can filter on mobile", func(t *testing.T) {
		a := newJumpClones(t, true)
		a.filterChip.SetSelected(map[string]string{jumpClonesFilterSystem: "Amarr"})
		assert.ElementsMatch(t, []int64{2}, cloneIDs(a))
	})
	t.Run("can filter on desktop", func(t *testing.T) {
		a := newJumpClones(t, false)
		a.selectSolarSystem.SetSelected("Amarr")
		assert.ElementsMatch(t, []int64{2}, cloneIDs(a))
	})
	t.Run("shows all filters on mobile", func(t *testing.T) {
		a := newJumpClones(t, true)
		assert.Equal(t, map[string]string{
			jumpClonesFilterCharacter: "",
			jumpClonesFilterRegion:    "",
			jumpClonesFilterSystem:    "",
			jumpClonesFilterTag:       "",
		}, a.filterChip.Selected())
	})
}

func TestOriginTextLayout(t *testing.T) {
	test.NewTempApp(t)
	newObjects := func() (*xwidget.RichText, *widget.Label) {
		s := &app.EveSolarSystem{SecurityStatus: 0.9}
		security := xwidget.NewRichText(s.SecurityStatusRichText()...)
		label := widget.NewLabel("  Jita [shortest]")
		label.Truncation = fyne.TextTruncateEllipsis
		return security, label
	}
	t.Run("should place label over the security paddings, so the text reads as one", func(t *testing.T) {
		security, label := newObjects()
		originTextLayout{}.Layout([]fyne.CanvasObject{security, label}, fyne.NewSize(300, 40))

		want := security.MinSize().Width - 2*theme.Size(theme.SizeNameInnerPadding)
		assert.Equal(t, want, label.Position().X)
		assert.Equal(t, 300-want, label.Size().Width)
	})
	t.Run("should give label the full width when security is hidden", func(t *testing.T) {
		security, label := newObjects()
		security.Hide()
		originTextLayout{}.Layout([]fyne.CanvasObject{security, label}, fyne.NewSize(300, 40))

		assert.Equal(t, float32(0), label.Position().X)
		assert.Equal(t, float32(300), label.Size().Width)
	})
}

func TestJumpClones_ErrorDisplay(t *testing.T) {
	newScreen := func(t *testing.T) (*JumpClones, *sql.DB) {
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		a := NewJumpClones(u)
		return a, db
	}
	t.Run("shows error when update fails and keeps it on filtering", func(t *testing.T) {
		a, db := newScreen(t)
		if _, err := db.Exec("DROP TABLE character_jump_clones"); err != nil { // let the next update fail
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

func TestJumpCloneRow_Jumps(t *testing.T) {
	t.Run("shows question mark when route is not calculated", func(t *testing.T) {
		// given
		r := jumpCloneRow{}
		// when
		got := r.jumps()
		// then
		xassert.Equal(t, "?", got)
	})
	t.Run("shows no route when route is empty", func(t *testing.T) {
		// given
		r := jumpCloneRow{route: []*app.EveSolarSystem{}}
		// when
		got := r.jumps()
		// then
		xassert.Equal(t, "No route", got)
	})
	t.Run("shows number of jumps", func(t *testing.T) {
		// given
		r := jumpCloneRow{route: []*app.EveSolarSystem{{ID: 1}, {ID: 2}, {ID: 3}}}
		// when
		got := r.jumps()
		// then
		xassert.Equal(t, "2", got)
	})
}

func TestJumpCloneRow_Compare(t *testing.T) {
	t.Run("sorts by jumps, then not calculated, then no route", func(t *testing.T) {
		// given
		noRoute := jumpCloneRow{route: []*app.EveSolarSystem{}}
		notCalculated := jumpCloneRow{}
		twoJumps := jumpCloneRow{route: []*app.EveSolarSystem{{ID: 1}, {ID: 2}, {ID: 3}}}
		zeroJumps := jumpCloneRow{route: []*app.EveSolarSystem{{ID: 1}}}
		rows := []jumpCloneRow{noRoute, notCalculated, twoJumps, zeroJumps}
		// when
		slices.SortFunc(rows, func(a, b jumpCloneRow) int {
			return a.compare(b)
		})
		// then
		got := xslices.Map(rows, func(r jumpCloneRow) string {
			return r.jumps()
		})
		xassert.Equal(t, []string{"0", "2", "?", "No route"}, got)
	})
}

func TestJumpClones_UpdateRoutes(t *testing.T) {
	makeRow := func(id int64, solarSystemID int64) jumpCloneRow {
		return jumpCloneRow{
			jc: &app.CharacterJumpClone2{
				ID:        id,
				Character: &app.EntityShort{Name: "Bruce"},
				Location: &app.EveLocation{
					SolarSystem: optional.New(&app.EveSolarSystem{
						ID:            solarSystemID,
						Constellation: &app.EveConstellation{Region: &app.EveRegion{}},
					}),
				},
			},
		}
	}
	t.Run("calculates routes and sorts by jumps", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewJumpClones(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		origin := &app.EveSolarSystem{ID: 30000142}
		a.origin = origin
		a.rows = []jumpCloneRow{
			{jc: &app.CharacterJumpClone2{ID: 1, Character: &app.EntityShort{Name: "Bruce"}, Location: &app.EveLocation{}}},
			makeRow(2, origin.ID),
		}
		// when
		a.updateRoutesAsync()
		// then
		got := xslices.Map(a.rowsFiltered, func(r jumpCloneRow) string {
			return fmt.Sprintf("%d:%s", r.jc.ID, r.jumps())
		})
		xassert.Equal(t, []string{"2:0", "1:?"}, got)
	})
	t.Run("shows no route when route was not found", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewJumpClones(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		a.origin = &app.EveSolarSystem{ID: 30000142}
		a.rows = []jumpCloneRow{makeRow(1, 31000001)} // wormhole system
		// when
		a.updateRoutesAsync()
		// then
		require.Len(t, a.rowsFiltered, 1)
		xassert.Equal(t, "No route", a.rowsFiltered[0].jumps())
	})
	t.Run("does nothing without origin", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewJumpClones(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		route := []*app.EveSolarSystem{{ID: 1}}
		r := makeRow(1, 30000142)
		r.route = route
		a.rows = []jumpCloneRow{r}
		// when
		a.updateRoutesAsync()
		// then
		xassert.Equal(t, route, a.rows[0].route)
		assert.Empty(t, a.rowsFiltered)
	})
}

func TestJumpClones_Signals(t *testing.T) {
	t.Run("loads clones with tags when jump clones changed", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewJumpClones(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCharacterFull()
		tag := factory.CreateCharacterTag("alpha")
		factory.AddCharacterToTag(tag, c)
		jc := factory.CreateCharacterJumpClone(storage.CreateCharacterJumpCloneParams{CharacterID: c.ID})
		// when
		a.u.Signals().CharacterSectionChanged.Emit(t.Context(), app.CharacterSectionUpdated{
			CharacterID: c.ID,
			Section:     app.SectionCharacterJumpClones,
		})
		// then
		require.Len(t, a.rows, 1)
		xassert.Equal(t, jc.ID, a.rows[0].jc.ID)
		xassert.Equal(t, set.Of("alpha"), a.rows[0].tags)
		xassert.Equal(t, "Showing 1 / 1 clones", a.footerLabel.Text)
	})
	t.Run("calculates routes when clones are loaded with origin set", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewJumpClones(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		location := factory.CreateEveLocationStation()
		factory.CreateCharacterJumpClone(storage.CreateCharacterJumpCloneParams{LocationID: location.ID})
		a.origin = location.SolarSystem.MustValue()
		// when
		a.u.Signals().CharacterSectionChanged.Emit(t.Context(), app.CharacterSectionUpdated{
			Section: app.SectionCharacterJumpClones,
		})
		// then
		require.Len(t, a.rows, 1)
		xassert.Equal(t, "0", a.rows[0].jumps())
	})
	t.Run("ignores change of other section", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewJumpClones(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		jc := factory.CreateCharacterJumpClone()
		// when
		a.u.Signals().CharacterSectionChanged.Emit(t.Context(), app.CharacterSectionUpdated{
			CharacterID: jc.CharacterID,
			Section:     app.SectionCharacterImplants,
		})
		// then
		assert.Empty(t, a.rows)
	})
	t.Run("loads clones on app init", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewJumpClones(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		factory.CreateCharacterJumpClone()
		// when
		a.u.Signals().AppInit.Emit(t.Context(), struct{}{})
		// then
		assert.Len(t, a.rows, 1)
	})
	t.Run("loads clones when character was added", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewJumpClones(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCharacterFull()
		factory.CreateCharacterJumpClone(storage.CreateCharacterJumpCloneParams{CharacterID: c.ID})
		// when
		a.u.Signals().CharacterAdded.Emit(t.Context(), c)
		// then
		assert.Len(t, a.rows, 1)
	})
	t.Run("removes clones when character was removed", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewJumpClones(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		a.rows = []jumpCloneRow{makeJumpCloneRow(1, "Bruce", "Jita", "The Forge")}
		c := factory.CreateCharacterFull()
		// when
		a.u.Signals().CharacterRemoved.Emit(t.Context(), &app.EntityShort{ID: c.ID})
		// then
		assert.Empty(t, a.rows)
	})
	t.Run("updates tags when tags changed", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewJumpClones(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCharacterFull()
		factory.CreateCharacterJumpClone(storage.CreateCharacterJumpCloneParams{CharacterID: c.ID})
		a.update(t.Context())
		factory.AddCharacterToTag(factory.CreateCharacterTag("alpha"), c)
		// when
		a.u.Signals().TagsChanged.Emit(t.Context(), struct{}{})
		// then
		require.Len(t, a.rows, 1)
		xassert.Equal(t, set.Of("alpha"), a.rows[0].tags)
	})
}
