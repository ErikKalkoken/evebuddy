package updatestatus

import (
	"os"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
)

func TestMain(m *testing.M) {
	// Leftover goroutines from one test can render text concurrently with the next,
	// which is not safe in Fyne's shared text shaper.
	runAsync = func(f func()) { f() }
	os.Exit(m.Run())
}

func TestShow(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()

	t.Run("can render window", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID:  c.ID,
			Section:      app.CharacterSections[0],
			ErrorMessage: "error",
		})
		factory.CreateCorporation()
		factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
			Section: app.EveUniverseSections[0],
		})
		sc := new(statuscache.StatusCache)
		require.NoError(t, sc.Init(t.Context(), st))
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st, StatusCache: sc})
		// when
		Show(u)
		// then
		w, created := u.GetOrCreateWindow("statusWindow")
		assert.False(t, created)
		a := w.Content().(*updateStatus)
		a.entityList.Select(1)
		a.sectionList.Select(0)
		assert.NotNil(t, w.Canvas().Capture())
		w.Close()
	})
	t.Run("reuses existing window", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		Show(u)
		n := len(fyne.CurrentApp().Driver().AllWindows())
		// when
		Show(u)
		// then
		assert.Equal(t, n, len(fyne.CurrentApp().Driver().AllWindows()))
		w, _ := u.GetOrCreateWindow("statusWindow")
		w.Close()
	})
	t.Run("stops reacting to signals when closed", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		sc := new(statuscache.StatusCache)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st, StatusCache: sc})
		Show(u)
		w, _ := u.GetOrCreateWindow("statusWindow")
		a := w.Content().(*updateStatus)
		w.Close()
		c := factory.CreateCharacter()
		require.NoError(t, sc.Init(t.Context(), st))
		// when
		u.Signals().CharacterAdded.Emit(t.Context(), c)
		// then
		assert.Len(t, a.entities, 2) // only General
	})
}

func TestUpdateStatus_Update(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()

	t.Run("lists characters, corporations and general with headers", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		r := factory.CreateCorporation()
		sc := new(statuscache.StatusCache)
		require.NoError(t, sc.Init(t.Context(), st))
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st, StatusCache: sc})
		w := test.NewTempWindow(t, nil)
		a := newUpdateStatus(u, w)
		t.Cleanup(a.stop)
		// when
		a.update(t.Context())
		// then
		type item struct {
			category sectionCategory
			id       int64
			name     string
		}
		var got []item
		for _, x := range a.entities {
			got = append(got, item{x.category, x.id, x.name})
		}
		want := []item{
			{sectionHeader, 0, "Characters"},
			{sectionCharacter, c.ID, c.EveCharacter.Name},
			{sectionHeader, 0, "Corporations"},
			{sectionCorporation, r.ID, r.EveCorporation.Name},
			{sectionHeader, 0, "General"},
			{sectionGeneral, app.EveUniverseSectionEntityID, app.EveUniverseSectionEntityName},
		}
		assert.Equal(t, want, got)
	})
	t.Run("lists only general when there are no characters or corporations", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		w := test.NewTempWindow(t, nil)
		a := newUpdateStatus(u, w)
		t.Cleanup(a.stop)
		// when
		a.update(t.Context())
		// then
		require.Len(t, a.entities, 2)
		assert.Equal(t, sectionHeader, a.entities[0].category)
		assert.Equal(t, sectionGeneral, a.entities[1].category)
	})
	t.Run("includes the section summary of an entity", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID:  c.ID,
			Section:      app.CharacterSections[0],
			ErrorMessage: "error",
		})
		sc := new(statuscache.StatusCache)
		require.NoError(t, sc.Init(t.Context(), st))
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st, StatusCache: sc})
		w := test.NewTempWindow(t, nil)
		a := newUpdateStatus(u, w)
		t.Cleanup(a.stop)
		// when
		a.update(t.Context())
		// then
		assert.Equal(t, 1, a.entities[1].ss.Errors)
	})
	t.Run("refreshes sections of the selected entity", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		sc := new(statuscache.StatusCache)
		require.NoError(t, sc.Init(t.Context(), st))
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st, StatusCache: sc})
		w := test.NewTempWindow(t, nil)
		a := newUpdateStatus(u, w)
		t.Cleanup(a.stop)
		w.SetContent(a)
		a.update(t.Context())
		a.entityList.Select(1)
		a.sectionList.Select(0)
		sc.SetCharacterSection(&app.CharacterSectionStatus{
			CharacterID: c.ID,
			SectionStatus: app.SectionStatus{
				Section:      app.CharacterSections[0],
				CompletedAt:  time.Now(),
				ErrorMessage: "error",
			},
		})
		// when
		a.update(t.Context())
		// then
		assert.Equal(t, "error", a.entitySections[0].ErrorMessage)
		assert.Equal(t, "ERROR", a.sectionStatus.status.Text)
	})
}

func TestUpdateStatus_UpdateChangesEntities(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()

	t.Run("keeps showing general when a character is removed", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		sc := new(statuscache.StatusCache)
		require.NoError(t, sc.Init(t.Context(), st))
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st, StatusCache: sc})
		w := test.NewTempWindow(t, nil)
		a := newUpdateStatus(u, w)
		t.Cleanup(a.stop)
		w.SetContent(a)
		a.update(t.Context())
		a.entityList.Select(3)
		a.sectionList.Select(0)
		require.NoError(t, st.DeleteCharacter(t.Context(), c.ID))
		require.NoError(t, sc.UpdateCharacters(t.Context(), st))
		// when
		u.Signals().CharacterRemoved.Emit(t.Context(), &app.EntityShort{ID: c.ID})
		// then
		assert.Equal(t, entityKey{category: sectionGeneral, id: app.EveUniverseSectionEntityID}, a.currentEntity)
		assert.Len(t, a.entitySections, len(app.EveUniverseSections))
		assert.False(t, a.nav.IsRoot())
	})
	t.Run("keeps showing corporation when its position becomes a header", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacter()
		r := factory.CreateCorporation()
		sc := new(statuscache.StatusCache)
		require.NoError(t, sc.Init(t.Context(), st))
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st, StatusCache: sc})
		u.Corporation().Stop() // skip background updates, which would call ESI
		w := test.NewTempWindow(t, nil)
		a := newUpdateStatus(u, w)
		a.stop()
		a.sb = kxwidget.NewSnackbar(w.Canvas()) // not started, so messages don't render concurrently
		w.SetContent(a)
		a.update(t.Context())
		a.entityList.Select(4)
		require.NoError(t, st.DeleteCharacter(t.Context(), c.ID))
		require.NoError(t, sc.UpdateCharacters(t.Context(), st))
		// when
		a.update(t.Context())
		// then
		assert.Equal(t, entityKey{category: sectionCorporation, id: r.ID}, a.currentEntity)
		require.Len(t, a.entitySections, len(app.CorporationSections))
		assert.Equal(t, r.ID, a.entitySections[0].EntityID)
		assert.NotPanics(t, a.entityMoreButton.MenuItems()[0].Action)
	})
	t.Run("keeps showing character when another one is inserted before it", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter(storage.CreateCharacterParams{
			ID: factory.CreateEveCharacter(storage.CreateEveCharacterParams{Name: "Bravo"}).ID,
		})
		sc := new(statuscache.StatusCache)
		require.NoError(t, sc.Init(t.Context(), st))
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st, StatusCache: sc})
		w := test.NewTempWindow(t, nil)
		a := newUpdateStatus(u, w)
		t.Cleanup(a.stop)
		w.SetContent(a)
		a.update(t.Context())
		a.entityList.Select(1)
		c2 := factory.CreateCharacter(storage.CreateCharacterParams{
			ID: factory.CreateEveCharacter(storage.CreateEveCharacterParams{Name: "Alpha"}).ID,
		})
		require.NoError(t, sc.UpdateCharacters(t.Context(), st))
		// when
		u.Signals().CharacterAdded.Emit(t.Context(), c2)
		// then
		assert.Equal(t, entityKey{category: sectionCharacter, id: c.ID}, a.currentEntity)
		assert.Equal(t, c.ID, a.entitySections[0].EntityID)
	})
	t.Run("returns to the entity list when the shown character is removed", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		sc := new(statuscache.StatusCache)
		require.NoError(t, sc.Init(t.Context(), st))
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st, StatusCache: sc})
		w := test.NewTempWindow(t, nil)
		a := newUpdateStatus(u, w)
		t.Cleanup(a.stop)
		w.SetContent(a)
		a.update(t.Context())
		a.entityList.Select(1)
		a.sectionList.Select(0)
		require.NoError(t, st.DeleteCharacter(t.Context(), c.ID))
		require.NoError(t, sc.UpdateCharacters(t.Context(), st))
		// when
		u.Signals().CharacterRemoved.Emit(t.Context(), &app.EntityShort{ID: c.ID})
		// then
		assert.True(t, a.nav.IsRoot())
		assert.Equal(t, entityKey{}, a.currentEntity)
		assert.Equal(t, -1, a.currentSectionID)
		assert.Empty(t, a.entitySections)
	})
}

func TestUpdateStatus_Signals(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()

	cases := []struct {
		name string
		emit func(t *testing.T, s *app.Signals)
	}{
		{"character added", func(t *testing.T, s *app.Signals) {
			s.CharacterAdded.Emit(t.Context(), &app.Character{})
		}},
		{"character removed", func(t *testing.T, s *app.Signals) {
			s.CharacterRemoved.Emit(t.Context(), &app.EntityShort{})
		}},
		{"character section updated", func(t *testing.T, s *app.Signals) {
			s.CharacterSectionUpdated.Emit(t.Context(), app.CharacterSectionUpdated{})
		}},
		{"corporation section updated", func(t *testing.T, s *app.Signals) {
			s.CorporationSectionUpdated.Emit(t.Context(), app.CorporationSectionUpdated{})
		}},
		{"eve universe section updated", func(t *testing.T, s *app.Signals) {
			s.EveUniverseSectionUpdated.Emit(t.Context(), app.EveUniverseSectionUpdated{})
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name+" updates entities", func(t *testing.T) {
			// given
			testutil.MustTruncateTables(db)
			factory.CreateCharacter()
			sc := new(statuscache.StatusCache)
			require.NoError(t, sc.Init(t.Context(), st))
			u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st, StatusCache: sc})
			w := test.NewTempWindow(t, nil)
			a := newUpdateStatus(u, w)
			t.Cleanup(a.stop)
			// when
			tc.emit(t, u.Signals())
			// then
			assert.Len(t, a.entities, 4)
		})
	}
}

func TestUpdateStatus_SelectEntity(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()

	t.Run("ignores headers", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		w := test.NewTempWindow(t, nil)
		a := newUpdateStatus(u, w)
		t.Cleanup(a.stop)
		w.SetContent(a)
		a.update(t.Context())
		// when
		a.entityList.Select(0)
		// then
		assert.True(t, a.nav.IsRoot())
		assert.Equal(t, entityKey{}, a.currentEntity)
	})
	t.Run("shows sections of a character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		sc := new(statuscache.StatusCache)
		require.NoError(t, sc.Init(t.Context(), st))
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st, StatusCache: sc})
		w := test.NewTempWindow(t, nil)
		a := newUpdateStatus(u, w)
		t.Cleanup(a.stop)
		w.SetContent(a)
		a.update(t.Context())
		// when
		a.entityList.Select(1)
		// then
		assert.False(t, a.nav.IsRoot())
		assert.Equal(t, entityKey{category: sectionCharacter, id: c.ID}, a.currentEntity)
		assert.Equal(t, -1, a.currentSectionID)
		require.Len(t, a.entitySections, len(app.CharacterSections))
		assert.Equal(t, c.ID, a.entitySections[0].EntityID)
	})
	t.Run("shows sections of a corporation", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		r := factory.CreateCorporation()
		sc := new(statuscache.StatusCache)
		require.NoError(t, sc.Init(t.Context(), st))
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st, StatusCache: sc})
		w := test.NewTempWindow(t, nil)
		a := newUpdateStatus(u, w)
		t.Cleanup(a.stop)
		w.SetContent(a)
		a.update(t.Context())
		// when
		a.entityList.Select(1)
		// then
		assert.False(t, a.nav.IsRoot())
		require.Len(t, a.entitySections, len(app.CorporationSections))
		assert.Equal(t, r.ID, a.entitySections[0].EntityID)
	})
	t.Run("shows sections of general", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		w := test.NewTempWindow(t, nil)
		a := newUpdateStatus(u, w)
		t.Cleanup(a.stop)
		w.SetContent(a)
		a.update(t.Context())
		// when
		a.entityList.Select(1)
		// then
		assert.False(t, a.nav.IsRoot())
		assert.Len(t, a.entitySections, len(app.EveUniverseSections))
	})
	t.Run("disables entity menu when offline", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		w := test.NewTempWindow(t, nil)
		a := newUpdateStatus(u, w)
		t.Cleanup(a.stop)
		w.SetContent(a)
		a.update(t.Context())
		// when
		a.entityList.Select(1)
		// then
		assert.True(t, a.entityMoreButton.Disabled())
		assert.Len(t, a.entityMoreButton.MenuItems(), 1)
	})
	t.Run("enables entity menu when online", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st, IsOnline: true})
		w := test.NewTempWindow(t, nil)
		a := newUpdateStatus(u, w)
		t.Cleanup(a.stop)
		w.SetContent(a)
		a.update(t.Context())
		// when
		a.entityList.Select(1)
		// then
		assert.False(t, a.entityMoreButton.Disabled())
	})
}

func TestUpdateStatus_SelectSection(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()

	t.Run("shows details of a section", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID:  c.ID,
			Section:      app.CharacterSections[0],
			ErrorMessage: "error",
		})
		sc := new(statuscache.StatusCache)
		require.NoError(t, sc.Init(t.Context(), st))
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st, StatusCache: sc})
		w := test.NewTempWindow(t, nil)
		a := newUpdateStatus(u, w)
		t.Cleanup(a.stop)
		w.SetContent(a)
		a.update(t.Context())
		a.entityList.Select(1)
		// when
		a.sectionList.Select(0)
		// then
		assert.Equal(t, 0, a.currentSectionID)
		assert.Equal(t, "ERROR", a.sectionStatus.status.Text)
		assert.Equal(t, "error", a.sectionStatus.issue.Text)
		assert.Len(t, a.sectionMoreButton.MenuItems(), 2)
	})
}

func TestUpdateStatus_MoreMenu(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()

	t.Run("has reload items", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		w := test.NewTempWindow(t, nil)
		// when
		a := newUpdateStatus(u, w)
		t.Cleanup(a.stop)
		// then
		var got []string
		for _, it := range a.moreButton.MenuItems() {
			got = append(got, it.Label)
		}
		want := []string{
			"Reload all characters",
			"Reload all corporations",
			"Reload all general entities",
			"",
			"Reload notifications for all characters",
			"Reload colonies for all characters",
		}
		assert.Equal(t, want, got)
	})
	t.Run("has export item in developer mode", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st, IsDeveloperMode: true})
		w := test.NewTempWindow(t, nil)
		// when
		a := newUpdateStatus(u, w)
		t.Cleanup(a.stop)
		// then
		items := a.moreButton.MenuItems()
		assert.Equal(t, "Export notification fixtures...", items[len(items)-1].Label)
	})
	t.Run("is disabled when offline", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		w := test.NewTempWindow(t, nil)
		// when
		a := newUpdateStatus(u, w)
		t.Cleanup(a.stop)
		// then
		assert.True(t, a.moreButton.Disabled())
	})
	t.Run("is enabled when online", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st, IsOnline: true})
		w := test.NewTempWindow(t, nil)
		// when
		a := newUpdateStatus(u, w)
		t.Cleanup(a.stop)
		// then
		assert.False(t, a.moreButton.Disabled())
	})
	t.Run("can run reload actions", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		u.Character().Stop() // skip background updates, which would call ESI
		u.Corporation().Stop()
		u.EVEUniverse().Stop()
		w := test.NewTempWindow(t, nil)
		a := newUpdateStatus(u, w)
		a.stop()
		a.sb = kxwidget.NewSnackbar(w.Canvas()) // not started, so messages don't render concurrently
		a.update(t.Context())
		for _, it := range a.moreButton.MenuItems() {
			if it.IsSeparator {
				continue
			}
			// when/then
			assert.NotPanics(t, it.Action, it.Label)
		}
	})
}

func TestUpdateStatus_MakeEntityMenuItems(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()

	for name, id := range map[string]int{"character": 1, "corporation": 3, "general": 5} {
		t.Run("can run reload action for "+name, func(t *testing.T) {
			// given
			testutil.MustTruncateTables(db)
			factory.CreateCharacter()
			factory.CreateCorporation()
			sc := new(statuscache.StatusCache)
			require.NoError(t, sc.Init(t.Context(), st))
			u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st, StatusCache: sc})
			u.Character().Stop() // skip background updates, which would call ESI
			u.Corporation().Stop()
			u.EVEUniverse().Stop()
			w := test.NewTempWindow(t, nil)
			a := newUpdateStatus(u, w)
			a.stop()
			a.sb = kxwidget.NewSnackbar(w.Canvas()) // not started, so messages don't render concurrently
			a.update(t.Context())
			items := a.makeEntityMenuItems(a.entities[id])
			// when/then
			require.Len(t, items, 1)
			assert.NotPanics(t, items[0].Action)
		})
	}
}

func TestUpdateStatus_MakeSectionMenuItems(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()

	ss := app.CacheSectionStatus{
		EntityID:     42,
		EntityName:   "Bruce Wayne",
		ErrorMessage: "failed",
		SectionID:    string(app.CharacterSections[0]),
		SectionName:  "Assets",
	}
	t.Run("disables reload when offline", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		w := test.NewTempWindow(t, nil)
		a := newUpdateStatus(u, w)
		t.Cleanup(a.stop)
		// when
		items := a.makeSectionMenuItems(ss, sectionCharacter)
		// then
		require.Len(t, items, 2)
		assert.True(t, items[0].Disabled)
		assert.False(t, items[1].Disabled)
	})
	t.Run("enables reload when online", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st, IsOnline: true})
		w := test.NewTempWindow(t, nil)
		a := newUpdateStatus(u, w)
		t.Cleanup(a.stop)
		// when
		items := a.makeSectionMenuItems(ss, sectionCharacter)
		// then
		assert.False(t, items[0].Disabled)
	})
	for name, c := range map[string]sectionCategory{
		"character":   sectionCharacter,
		"corporation": sectionCorporation,
		"general":     sectionGeneral,
		"undefined":   sectionHeader,
	} {
		t.Run("can run reload action for "+name, func(t *testing.T) {
			// given
			testutil.MustTruncateTables(db)
			u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
			u.Character().Stop() // skip background updates, which would call ESI
			u.Corporation().Stop()
			u.EVEUniverse().Stop()
			w := test.NewTempWindow(t, nil)
			a := newUpdateStatus(u, w)
			a.stop()
			a.sb = kxwidget.NewSnackbar(w.Canvas()) // not started, so messages don't render concurrently
			items := a.makeSectionMenuItems(ss, c)
			// when/then
			assert.NotPanics(t, items[0].Action)
		})
	}
	t.Run("copies issue to clipboard", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		w := test.NewTempWindow(t, nil)
		a := newUpdateStatus(u, w)
		a.stop()
		a.sb = kxwidget.NewSnackbar(w.Canvas()) // not started, so messages don't render concurrently
		items := a.makeSectionMenuItems(ss, sectionCharacter)
		// when
		items[1].Action()
		// then
		got := fyne.CurrentApp().Clipboard().Content()
		assert.Equal(t, "Bruce Wayne [42] / Assets: failed", got)
	})
}

func TestEntityItem(t *testing.T) {
	loadIcon := func(_ int64, _ int, setIcon func(fyne.Resource)) {
		setIcon(icons.Characterplaceholder64Jpeg)
	}
	t.Run("shows header", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		w := newEntityItem(false, loadIcon, loadIcon)
		// when
		w.set(entity{category: sectionHeader, name: "Characters"})
		// then
		assert.Equal(t, "Characters", w.name.Text)
		assert.True(t, w.name.TextStyle.Bold)
		assert.False(t, w.icon.Visible())
		assert.False(t, w.spinner.Visible())
		assert.False(t, w.status.Visible())
		assert.True(t, w.thief.Visible())
	})
	t.Run("shows character after header", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		var loadedID int64
		w := newEntityItem(false, func(id int64, _ int, setIcon func(fyne.Resource)) {
			loadedID = id
			setIcon(icons.Characterplaceholder64Jpeg)
		}, loadIcon)
		w.set(entity{category: sectionHeader, name: "Characters"})
		// when
		w.set(entity{
			category: sectionCharacter,
			id:       42,
			name:     "Bruce Wayne",
			ss:       app.StatusSummary{Current: 1, Total: 1},
		})
		// then
		assert.Equal(t, int64(42), loadedID)
		assert.Equal(t, "Bruce Wayne", w.name.Text)
		assert.False(t, w.name.TextStyle.Bold)
		assert.Equal(t, icons.Characterplaceholder64Jpeg, w.icon.Resource)
		assert.True(t, w.icon.Visible())
		assert.True(t, w.status.Visible())
		assert.Equal(t, "OK", w.status.Text)
		assert.Equal(t, widget.SuccessImportance, w.status.Importance)
		assert.False(t, w.thief.Visible())
		assert.False(t, w.spinner.Visible())
	})
	t.Run("loads corporation logo", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		var loadedID int64
		w := newEntityItem(false, loadIcon, func(id int64, _ int, setIcon func(fyne.Resource)) {
			loadedID = id
			setIcon(icons.Corporationplaceholder64Png)
		})
		// when
		w.set(entity{category: sectionCorporation, id: 42, name: "Wayne Inc"})
		// then
		assert.Equal(t, int64(42), loadedID)
		assert.Equal(t, icons.Corporationplaceholder64Png, w.icon.Resource)
	})
	t.Run("shows star map icon for general", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		w := newEntityItem(false, loadIcon, loadIcon)
		// when
		w.set(entity{category: sectionGeneral, name: "General"})
		// then
		assert.Equal(t, eveicon.FromName(eveicon.StarMap), w.icon.Resource)
	})
	t.Run("shows spinner when running", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		w := newEntityItem(false, loadIcon, loadIcon)
		// when
		w.set(entity{category: sectionCharacter, ss: app.StatusSummary{IsRunning: true}})
		// then
		assert.True(t, w.spinner.Visible())
	})
	t.Run("hides spinner when running in offline mode", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		w := newEntityItem(true, loadIcon, loadIcon)
		// when
		w.set(entity{category: sectionCharacter, ss: app.StatusSummary{IsRunning: true}})
		// then
		assert.False(t, w.spinner.Visible())
	})
}

func TestSectionItem(t *testing.T) {
	running := app.CacheSectionStatus{
		SectionName:  "Assets",
		StartedAt:    time.Now(),
		ErrorMessage: "failed",
	}
	t.Run("shows section", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		w := newSectionItem(false)
		// when
		w.set(running)
		// then
		assert.Equal(t, "Assets", w.name.Text)
		assert.Equal(t, "ERROR", w.status.Text)
		assert.Equal(t, widget.DangerImportance, w.status.Importance)
		assert.True(t, w.spinner.Visible())
	})
	t.Run("hides spinner when not running", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		w := newSectionItem(false)
		// when
		w.set(app.CacheSectionStatus{SectionName: "Assets"})
		// then
		assert.False(t, w.spinner.Visible())
	})
	t.Run("hides spinner when running in offline mode", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		w := newSectionItem(true)
		// when
		w.set(running)
		// then
		assert.False(t, w.spinner.Visible())
	})
}

func TestSectionStatus(t *testing.T) {
	t.Run("shows error as issue", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		w := newSectionStatus(false)
		// when
		w.set(app.CacheSectionStatus{ErrorMessage: "failed", Comment: "comment"})
		// then
		assert.Equal(t, "failed", w.issue.Text)
		assert.Equal(t, widget.DangerImportance, w.issue.Importance)
	})
	t.Run("shows comment as issue", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		w := newSectionStatus(false)
		w.set(app.CacheSectionStatus{ErrorMessage: "failed"})
		// when
		w.set(app.CacheSectionStatus{Comment: "comment"})
		// then
		assert.Equal(t, "comment", w.issue.Text)
		assert.Equal(t, widget.MediumImportance, w.issue.Importance)
	})
	t.Run("shows placeholders when empty", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		w := newSectionStatus(false)
		// when
		w.set(app.CacheSectionStatus{})
		// then
		assert.Equal(t, "-", w.issue.Text)
		assert.Equal(t, "?", w.completedAt.Text)
		assert.Equal(t, "-", w.startedAt.Text)
		assert.Equal(t, "?", w.nextUpdate.Text)
		assert.Equal(t, "Missing", w.status.Text)
	})
	t.Run("shows times", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		w := newSectionStatus(false)
		// when
		w.set(app.CacheSectionStatus{
			CompletedAt: time.Now().Add(-time.Hour + time.Minute),
			StartedAt:   time.Now().Add(-2*time.Hour + time.Minute),
			Timeout:     3 * time.Hour,
		})
		// then
		assert.Equal(t, "59 minutes ago", w.completedAt.Text)
		assert.Equal(t, "1 hour ago", w.startedAt.Text)
		assert.Equal(t, "3h 0m", w.timeout.Text)
		assert.Equal(t, "2 hours from now", w.nextUpdate.Text)
	})
	t.Run("shows overdue next update", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		w := newSectionStatus(false)
		// when
		w.set(app.CacheSectionStatus{
			CompletedAt: time.Now().Add(-5*time.Hour - time.Minute),
			Timeout:     3 * time.Hour,
		})
		// then
		assert.Equal(t, "2 hours ago", w.nextUpdate.Text)
	})
}
