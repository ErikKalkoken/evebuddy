package screens

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestCloneJumpsDisplay(t *testing.T) {
	t.Run("shows not calculated without origin", func(t *testing.T) {
		// given
		r := jumpCloneRow{route: []*app.EveSolarSystem{{ID: 1}, {ID: 2}}}
		// when
		got := cloneJumpsDisplay(r, nil, app.RouteShorter)
		// then
		xassert.Equal(t, "Not calculated", got)
	})
	t.Run("shows not calculated when route was not requested yet", func(t *testing.T) {
		// given
		r := jumpCloneRow{}
		// when
		got := cloneJumpsDisplay(r, &app.EveSolarSystem{ID: 1}, app.RouteShorter)
		// then
		xassert.Equal(t, "Not calculated", got)
	})
	t.Run("shows no route found when route is empty", func(t *testing.T) {
		// given
		r := jumpCloneRow{route: []*app.EveSolarSystem{}}
		// when
		got := cloneJumpsDisplay(r, &app.EveSolarSystem{ID: 1}, app.RouteShorter)
		// then
		xassert.Equal(t, "No route found", got)
	})
	t.Run("shows jumps with route preference", func(t *testing.T) {
		// given
		r := jumpCloneRow{route: []*app.EveSolarSystem{{ID: 1}, {ID: 2}, {ID: 3}}}
		// when
		got := cloneJumpsDisplay(r, &app.EveSolarSystem{ID: 1}, app.RouteSafer)
		// then
		xassert.Equal(t, "2 (secure)", got)
	})
}

func TestShowCloneDetailWindow(t *testing.T) {
	t.Run("can show clone with implants and route", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		c := factory.CreateCharacterFull()
		jc := factory.CreateCharacterJumpClone(storage.CreateCharacterJumpCloneParams{CharacterID: c.ID})
		r := jumpCloneRow{
			jc: &app.CharacterJumpClone2{
				Character:     &app.EntityShort{ID: c.ID, Name: c.EveCharacter.Name},
				CloneID:       jc.CloneID,
				ID:            jc.ID,
				ImplantsCount: len(jc.Implants),
				Location:      factory.CreateEveLocationStructure(),
			},
			route: []*app.EveSolarSystem{factory.CreateEveSolarSystem(), factory.CreateEveSolarSystem()},
		}
		// when
		showCloneDetailWindow(u, r, factory.CreateEveSolarSystem(), app.RouteShorter)
		// then
		_, created := u.GetOrCreateWindow(fmt.Sprintf("clone-%d-%d", c.ID, jc.ID))
		assert.False(t, created)
	})
	t.Run("can show clone without implants and origin", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		c := factory.CreateCharacterFull()
		r := jumpCloneRow{
			jc: &app.CharacterJumpClone2{
				Character: &app.EntityShort{ID: c.ID, Name: c.EveCharacter.Name},
				CloneID:   7,
				ID:        42,
				Location:  factory.CreateEveLocationStructure(),
			},
		}
		// when
		showCloneDetailWindow(u, r, nil, app.RouteShorter)
		// then
		_, created := u.GetOrCreateWindow(fmt.Sprintf("clone-%d-%d", c.ID, 42))
		assert.False(t, created)
	})
	t.Run("shows no window when row has no clone", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		fyneApp := test.NewTempApp(t)
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     fyneApp,
			Storage: st,
		})
		n := len(fyneApp.Driver().AllWindows())
		// when
		showCloneDetailWindow(u, jumpCloneRow{}, nil, app.RouteShorter)
		// then
		assert.Len(t, fyneApp.Driver().AllWindows(), n)
	})
	t.Run("reuses existing window", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		c := factory.CreateCharacterFull()
		r := jumpCloneRow{
			jc: &app.CharacterJumpClone2{
				Character: &app.EntityShort{ID: c.ID, Name: c.EveCharacter.Name},
				ID:        42,
				Location:  factory.CreateEveLocationStructure(),
			},
		}
		w, _ := u.GetOrCreateWindow(fmt.Sprintf("clone-%d-%d", c.ID, 42))
		// when
		showCloneDetailWindow(u, r, nil, app.RouteShorter)
		// then
		assert.Nil(t, w.Content())
	})
}

func TestMakeImplantsList(t *testing.T) {
	t.Run("loads implants of clone", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		fyneApp := test.NewTempApp(t)
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     fyneApp,
			Storage: st,
		})
		c := factory.CreateCharacterFull()
		jc := factory.CreateCharacterJumpClone(storage.CreateCharacterJumpCloneParams{
			CharacterID: c.ID,
			Implants:    []int64{factory.CreateEveType().ID, factory.CreateEveType().ID},
		})
		// when
		list := makeImplantsList(u, c.ID, jc.CloneID, false, fyneApp.NewWindow("test"))
		// then
		xassert.Equal(t, 2, list.Length())
	})
	t.Run("shows no implants when clone can not be loaded", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		fyneApp := test.NewTempApp(t)
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     fyneApp,
			Storage: st,
		})
		c := factory.CreateCharacterFull()
		jc := factory.CreateCharacterJumpClone(storage.CreateCharacterJumpCloneParams{CharacterID: c.ID})
		if _, err := db.Exec("DROP TABLE character_jump_clone_implants"); err != nil {
			t.Fatal(err)
		}
		// when
		list := makeImplantsList(u, c.ID, jc.CloneID, false, fyneApp.NewWindow("test"))
		// then
		xassert.Equal(t, 0, list.Length())
	})
}
