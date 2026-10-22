package infoviewer

import (
	"context"
	"fmt"
	"testing"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func entityItemTexts(w *entityList) []string {
	return xslices.Map(w.items, func(x entityItem) string {
		return x.category + ": " + x.text
	})
}

func TestLocationInfo_update(t *testing.T) {
	t.Run("should show station", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		httpmock.Activate()
		t.Cleanup(httpmock.DeactivateAndReset)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		et := factory.CreateEveType(storage.CreateEveTypeParams{Name: "Caldari Station", Description: "A station"})
		l := factory.CreateEveLocationStation(storage.UpdateOrCreateLocationParams{
			Name:   "Jita IV - Moon 4",
			TypeID: optional.New(et.ID),
		})
		httpmock.RegisterResponder("GET", fmt.Sprintf("https://esi.evetech.net/universe/stations/%d", l.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"max_dockable_ship_volume":   50000000,
				"name":                       l.Name,
				"office_rental_cost":         10000,
				"position":                   map[string]any{"x": 1, "y": 2, "z": 3},
				"reprocessing_efficiency":    0.5,
				"reprocessing_stations_take": 0.05,
				"services":                   []string{"repair-facilities", "cloning"},
				"station_id":                 l.ID,
				"system_id":                  l.SolarSystem.MustValue().ID,
				"type_id":                    et.ID,
			}))
		a := newLocationInfo(New(u), l.ID, 0)
		// when
		err := a.update(t.Context())
		// then
		require.NoError(t, err)
		assert.Equal(t, "Jita IV - Moon 4", a.name.Text)
		assert.Equal(t, "Caldari Station", a.typeInfo.Text)
		assert.Equal(t, "A station", a.description.Text)
		assert.Equal(t, l.Owner.MustValue().Name, a.owner.Text)
		es := l.SolarSystem.MustValue()
		assert.Equal(t, []showParams{
			{variant: Region, entityID: es.Constellation.Region.ID},
			{variant: Constellation, entityID: es.Constellation.ID},
			{variant: SolarSystem, entityID: es.ID},
		}, xslices.Map(a.location.items, func(x entityItem) showParams {
			return showParams{variant: x.infoVariant, entityID: x.id}
		}))
		assert.Equal(t, []string{"Service: Cloning", "Service: Repair Facilities"}, entityItemTexts(a.services))
		assert.Equal(t, []string{"Description", "Location", "Services"}, xslices.Map(a.tabs.Items, func(x *container.TabItem) string {
			return x.Text
		}))
	})
	t.Run("should show structure without services", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		httpmock.Activate()
		t.Cleanup(httpmock.DeactivateAndReset)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		l := factory.CreateEveLocationStructure()
		a := newLocationInfo(New(u), l.ID, 0)
		// when
		err := a.update(t.Context())
		// then
		require.NoError(t, err)
		assert.Equal(t, l.Name, a.name.Text)
		assert.Empty(t, a.services.items)
		assert.Equal(t, 0, httpmock.GetTotalCallCount())
	})
	t.Run("should show type name when type has no description", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		g := factory.CreateEveGroup()
		// created directly, because the factory always adds a description
		err := st.CreateEveType(t.Context(), storage.CreateEveTypeParams{ID: 35832, GroupID: g.ID, Name: "Astrahus"})
		require.NoError(t, err)
		l := factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{TypeID: optional.New[int64](35832)})
		a := newLocationInfo(New(u), l.ID, 0)
		// when
		err = a.update(t.Context())
		// then
		require.NoError(t, err)
		assert.Equal(t, "Astrahus", a.description.Text)
	})
	t.Run("should show owner when tapped", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(_ func(context.Context)) {},
			IsOnline: true,
			Storage:  st,
		})
		l := factory.CreateEveLocationStructure()
		iw := New(u)
		a := newLocationInfo(iw, l.ID, 0)
		err := a.update(t.Context())
		require.NoError(t, err)
		// when
		a.owner.OnTapped()
		// then
		owner := l.Owner.MustValue()
		assert.Equal(t, &showParams{variant: eveEntity2InfoVariant(owner), entityID: owner.ID}, iw.current)
	})
	t.Run("should show EVE ID in developer mode", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), IsDeveloperMode: true, Storage: st})
		l := factory.CreateEveLocationStructure()
		a := newLocationInfo(New(u), l.ID, 0)
		// when
		err := a.update(t.Context())
		// then
		require.NoError(t, err)
		assert.Equal(t, "Attributes", a.tabs.Items[len(a.tabs.Items)-1].Text)
	})
	t.Run("should report error when structure name is not accessible", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		// created directly, because the factory always adds a name
		err := st.UpdateOrCreateEveLocation(t.Context(), storage.UpdateOrCreateLocationParams{ID: 1_000_000_000_001})
		require.NoError(t, err)
		a := newLocationInfo(New(u), 1_000_000_000_001, 0)
		// when
		err = a.update(t.Context())
		// then
		assert.ErrorContains(t, err, "not allowed to access structure")
	})
	t.Run("should show structure with character token", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		l := factory.CreateEveLocationStructure()
		a := newLocationInfo(New(u), l.ID, c.ID)
		// when
		err := a.update(t.Context())
		// then
		require.NoError(t, err)
		assert.Equal(t, l.Name, a.name.Text)
	})
	t.Run("should report error when character has no token", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		c := factory.CreateCharacter()
		l := factory.CreateEveLocationStructure()
		a := newLocationInfo(New(u), l.ID, c.ID)
		// when
		err := a.update(t.Context())
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
		assert.Equal(t, "Loading...", a.name.Text)
	})
}
