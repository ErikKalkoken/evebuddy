package infoviewer_test

import (
	"context"
	"fmt"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/infoviewer"
)

// loadAndRender runs the pending page load and returns the rendered info window.
func loadAndRender(t *testing.T, jobs []func(context.Context)) string {
	t.Helper()
	require.Len(t, jobs, 1)
	jobs[0](t.Context())
	ws := fyne.CurrentApp().Driver().AllWindows()
	w := ws[len(ws)-1]
	return test.RenderToMarkup(w.Canvas())
}

func TestInfoViewer_CanRenderPages(t *testing.T) {
	t.Run("alliance", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		httpmock.Activate()
		t.Cleanup(httpmock.DeactivateAndReset)
		var jobs []func(context.Context)
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(job func(context.Context)) { jobs = append(jobs, job) },
			IsOnline: true,
			Storage:  st,
		})
		alliance := factory.CreateEveEntityAlliance()
		creator := factory.CreateEveEntityCharacter()
		corporation := factory.CreateEveEntityCorporation()
		httpmock.RegisterResponder("GET", fmt.Sprintf("https://esi.evetech.net/alliances/%d", alliance.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"creator_corporation_id":  corporation.ID,
				"creator_id":              creator.ID,
				"date_founded":            "2016-06-26T21:00:00Z",
				"executor_corporation_id": corporation.ID,
				"name":                    "C C P Alliance",
				"ticker":                  "CCP",
			}))
		httpmock.RegisterResponder("GET", fmt.Sprintf("https://esi.evetech.net/alliances/%d/corporations", alliance.ID),
			httpmock.NewJsonResponderOrPanic(200, []int64{corporation.ID}))
		iw := infoviewer.New(u)
		iw.Show(alliance)
		// when
		got := loadAndRender(t, jobs)
		// then
		assert.Contains(t, got, "C C P Alliance")
		assert.NotContains(t, got, "ERROR")
	})
	t.Run("bloodline", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		var jobs []func(context.Context)
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(job func(context.Context)) { jobs = append(jobs, job) },
			IsOnline: true,
			Storage:  st,
		})
		b := factory.CreateEveBloodline()
		iw := infoviewer.New(u)
		iw.ShowBloodline(b.ID)
		// when
		got := loadAndRender(t, jobs)
		// then
		assert.Contains(t, got, b.Name)
		assert.NotContains(t, got, "ERROR")
	})
	t.Run("character", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		httpmock.Activate()
		t.Cleanup(httpmock.DeactivateAndReset)
		var jobs []func(context.Context)
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(job func(context.Context)) { jobs = append(jobs, job) },
			IsOnline: true,
			Storage:  st,
		})
		c := factory.CreateEveCharacter()
		httpmock.RegisterResponder("GET", fmt.Sprintf("https://esi.evetech.net/characters/%d/corporationhistory", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{"corporation_id": c.Corporation.ID, "record_id": 1, "start_date": "2020-01-02T00:00:00Z"},
			}))
		iw := infoviewer.New(u)
		iw.Show(c.ToEveEntity())
		// when
		got := loadAndRender(t, jobs)
		// then
		assert.Contains(t, got, c.Name)
		assert.NotContains(t, got, "ERROR")
	})
	t.Run("constellation", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		httpmock.Activate()
		t.Cleanup(httpmock.DeactivateAndReset)
		var jobs []func(context.Context)
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(job func(context.Context)) { jobs = append(jobs, job) },
			IsOnline: true,
			Storage:  st,
		})
		constellation := factory.CreateEveConstellation()
		system := factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ConstellationID: constellation.ID})
		httpmock.RegisterResponder("GET", fmt.Sprintf("https://esi.evetech.net/universe/constellations/%d", constellation.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"constellation_id": constellation.ID,
				"name":             constellation.Name,
				"position":         map[string]any{"x": 1, "y": 2, "z": 3},
				"region_id":        constellation.Region.ID,
				"systems":          []int64{system.ID},
			}))
		iw := infoviewer.New(u)
		iw.Show(constellation.ToEveEntity())
		// when
		got := loadAndRender(t, jobs)
		// then
		assert.Contains(t, got, constellation.Name)
		assert.NotContains(t, got, "ERROR")
	})
	t.Run("corporation", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		httpmock.Activate()
		t.Cleanup(httpmock.DeactivateAndReset)
		var jobs []func(context.Context)
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(job func(context.Context)) { jobs = append(jobs, job) },
			IsOnline: true,
			Storage:  st,
		})
		c := factory.CreateEveCorporation()
		httpmock.RegisterResponder("GET", fmt.Sprintf("https://esi.evetech.net/corporations/%d/alliancehistory", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{}))
		iw := infoviewer.New(u)
		iw.Show(c.ToEveEntity())
		// when
		got := loadAndRender(t, jobs)
		// then
		assert.Contains(t, got, c.Name)
		assert.NotContains(t, got, "ERROR")
	})
	t.Run("faction", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		var jobs []func(context.Context)
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(job func(context.Context)) { jobs = append(jobs, job) },
			IsOnline: true,
			Storage:  st,
		})
		f := factory.CreateEveFaction()
		iw := infoviewer.New(u)
		iw.Show(&app.EveEntity{ID: f.ID, Name: f.Name, Category: app.EveEntityFaction})
		// when
		got := loadAndRender(t, jobs)
		// then
		assert.Contains(t, got, f.Name)
		assert.NotContains(t, got, "ERROR")
	})
	t.Run("location", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		var jobs []func(context.Context)
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(job func(context.Context)) { jobs = append(jobs, job) },
			IsOnline: true,
			Storage:  st,
		})
		l := factory.CreateEveLocationStructure()
		iw := infoviewer.New(u)
		iw.ShowLocation(l.ID)
		// when
		got := loadAndRender(t, jobs)
		// then
		assert.Contains(t, got, l.Name)
		assert.NotContains(t, got, "ERROR")
	})
	t.Run("race", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		var jobs []func(context.Context)
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(job func(context.Context)) { jobs = append(jobs, job) },
			IsOnline: true,
			Storage:  st,
		})
		r := factory.CreateEveRace()
		iw := infoviewer.New(u)
		iw.ShowRace(r.ID)
		// when
		got := loadAndRender(t, jobs)
		// then
		assert.Contains(t, got, r.Name)
		assert.NotContains(t, got, "ERROR")
	})
	t.Run("region", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		httpmock.Activate()
		t.Cleanup(httpmock.DeactivateAndReset)
		var jobs []func(context.Context)
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(job func(context.Context)) { jobs = append(jobs, job) },
			IsOnline: true,
			Storage:  st,
		})
		region := factory.CreateEveRegion()
		httpmock.RegisterResponder("GET", fmt.Sprintf("https://esi.evetech.net/universe/regions/%d", region.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"constellations": []int64{},
				"name":           region.Name,
				"region_id":      region.ID,
			}))
		iw := infoviewer.New(u)
		iw.Show(region.ToEveEntity())
		// when
		got := loadAndRender(t, jobs)
		// then
		assert.Contains(t, got, region.Name)
		assert.NotContains(t, got, "ERROR")
	})
	t.Run("solar system", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		httpmock.Activate()
		t.Cleanup(httpmock.DeactivateAndReset)
		var jobs []func(context.Context)
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(job func(context.Context)) { jobs = append(jobs, job) },
			IsOnline: true,
			Storage:  st,
		})
		system := factory.CreateEveSolarSystem()
		factory.GetOrCreateEveEntity(*system.ToEveEntity())
		factory.GetOrCreateEveEntity(*system.Constellation.ToEveEntity())
		httpmock.RegisterResponder("GET", fmt.Sprintf("https://esi.evetech.net/universe/systems/%d", system.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"constellation_id": system.Constellation.ID,
				"name":             system.Name,
				"position":         map[string]any{"x": 1, "y": 2, "z": 3},
				"security_status":  system.SecurityStatus,
				"system_id":        system.ID,
			}))
		iw := infoviewer.New(u)
		iw.Show(system.ToEveEntity())
		// when
		got := loadAndRender(t, jobs)
		// then
		assert.Contains(t, got, system.Name)
		assert.NotContains(t, got, "ERROR")
	})
	t.Run("type", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		var jobs []func(context.Context)
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(job func(context.Context)) { jobs = append(jobs, job) },
			IsOnline: true,
			Storage:  st,
		})
		c := factory.CreateCharacter()
		factory.GetOrCreateEveEntity(*c.EveCharacter.ToEveEntity())
		et := factory.CreateEveType()
		da := factory.CreateEveDogmaAttribute(storage.CreateEveDogmaAttributeParams{
			ID:     app.EveDogmaAttributeArmorHitpoints,
			UnitID: app.EveUnitHitpoints,
		})
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        et.ID,
			DogmaAttributeID: da.ID,
			Value:            1500,
		})
		iw := infoviewer.New(u)
		iw.ShowType(et.ID, c.ID)
		// when
		got := loadAndRender(t, jobs)
		// then
		assert.Contains(t, got, et.Name)
		assert.NotContains(t, got, "ERROR")
	})
}
