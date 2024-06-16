package eveuniverse_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/antihax/goesi"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverse"
	"github.com/ErikKalkoken/evebuddy/internal/app/sqlite"
	"github.com/ErikKalkoken/evebuddy/internal/app/sqlite/testutil"
)

func TestGetOrCreateEveRegionESI(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := goesi.NewAPIClient(nil, "")
	s := eveuniverse.New(r, client)
	ctx := context.Background()
	t.Run("should return existing region", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		factory.CreateEveRegion(sqlite.CreateEveRegionParams{ID: 6})
		// when
		x1, err := s.GetOrCreateEveRegionESI(ctx, 6)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(6), x1.ID)
		}
	})
	t.Run("should fetch region from ESI and create it", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		data := `{
			"constellations": [
			  20000302,
			  20000303
			],
			"description": "It has long been an established fact of civilization...",
			"name": "Metropolis",
			"region_id": 10000042
		  }`
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v1/universe/regions/10000042/",
			httpmock.NewStringResponder(200, data).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))

		// when
		x1, err := s.GetOrCreateEveRegionESI(ctx, 10000042)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(10000042), x1.ID)
			assert.Equal(t, "Metropolis", x1.Name)
			x2, err := r.GetEveRegion(ctx, 10000042)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
}

func TestGetOrCreateEveConstellationESI(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := goesi.NewAPIClient(nil, "")
	s := eveuniverse.New(r, client)
	ctx := context.Background()
	t.Run("should return existing constellation", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		factory.CreateEveConstellation(sqlite.CreateEveConstellationParams{ID: 25})
		// when
		x1, err := s.GetOrCreateEveConstellationESI(ctx, 25)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(25), x1.ID)
		}
	})
	t.Run("should fetch constellation from ESI and create it", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		factory.CreateEveRegion(sqlite.CreateEveRegionParams{ID: 10000001})
		data := `{
			"constellation_id": 20000009,
			"name": "Mekashtad",
			"position": {
			  "x": 67796138757472320,
			  "y": -70591121348560960,
			  "z": -59587016159270070
			},
			"region_id": 10000001,
			"systems": [
			  20000302,
			  20000303
			]
		  }`
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v1/universe/constellations/20000009/",
			httpmock.NewStringResponder(200, data).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))

		// when
		x1, err := s.GetOrCreateEveConstellationESI(ctx, 20000009)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(20000009), x1.ID)
			assert.Equal(t, "Mekashtad", x1.Name)
			assert.Equal(t, int32(10000001), x1.Region.ID)
			x2, err := r.GetEveConstellation(ctx, 20000009)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
}

func TestGetOrCreateEveSolarSystemESI(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := goesi.NewAPIClient(nil, "")
	s := eveuniverse.New(r, client)
	ctx := context.Background()
	t.Run("should return existing solar system", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		factory.CreateEveSolarSystem(sqlite.CreateEveSolarSystemParams{ID: 587})
		// when
		x1, err := s.GetOrCreateEveSolarSystemESI(ctx, 587)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(587), x1.ID)
		}
	})
	t.Run("should fetch solar system from ESI and create it", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		factory.CreateEveConstellation(sqlite.CreateEveConstellationParams{ID: 20000001})
		data := `{
			"constellation_id": 20000001,
			"name": "Akpivem",
			"planets": [
			  {
				"moons": [
				  40000042
				],
				"planet_id": 40000041
			  },
			  {
				"planet_id": 40000043
			  }
			],
			"position": {
			  "x": -91174141133075340,
			  "y": 43938227486247170,
			  "z": -56482824383339900
			},
			"security_class": "B",
			"security_status": 0.8462923765182495,
			"star_id": 40000040,
			"stargates": [
			  50000342
			],
			"system_id": 30000003
		  }`
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v4/universe/systems/30000003/",
			httpmock.NewStringResponder(200, data).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))

		// when
		x1, err := s.GetOrCreateEveSolarSystemESI(ctx, 30000003)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(30000003), x1.ID)
			assert.Equal(t, "Akpivem", x1.Name)
			assert.Equal(t, int32(20000001), x1.Constellation.ID)
			x2, err := r.GetEveSolarSystem(ctx, 30000003)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
	t.Run("should fetch solar system from ESI and create it (integration)", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()

		data1 := `{
			"constellations": [
			  20000001,
			  20000002,
			  20000003,
			  20000004,
			  20000005,
			  20000006,
			  20000007,
			  20000008,
			  20000009,
			  20000010,
			  20000011,
			  20000012,
			  20000013,
			  20000014,
			  20000015,
			  20000016
			],
			"description": "The Derelik region, sovereign seat of the Ammatar Mandate, became the shield to the Amarrian flank in the wake of the Minmatar Rebellion. Derelik witnessed many hostile exchanges between the Amarr and rebel forces as the latter tried to push deeper into the territory of their former masters. Having held their ground, thanks in no small part to the Ammatars' military efforts, the Amarr awarded the Ammatar with their own province. However, this portion of space shared borders with the newly forming Minmatar Republic as well as the Empire, and thus came to be situated in a dark recess surrounded by hostiles. \n\nGiven the lack of safe routes elsewhere, the local economies of this region were dependent on trade with the Amarr as their primary means of survival. The Ammatar persevered over many decades of economic stagnation and limited trade partners, and their determination has in recent decades been rewarded with an increase in economic prosperity. This harsh trail is a point of pride for all who call themselves Ammatar, and it has bolstered their faith in the Amarrian way to no end.",
			"name": "Derelik",
			"region_id": 10000001
		  }`
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v1/universe/regions/10000001/",
			httpmock.NewStringResponder(200, data1).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))

		data2 := `{
			"constellation_id": 20000001,
			"name": "San Matar",
			"position": {
			  "x": -94046559700991340,
			  "y": 49520153153798850,
			  "z": -42738731818401970
			},
			"region_id": 10000001,
			"systems": [
			  30000001,
			  30000002,
			  30000003,
			  30000004,
			  30000005,
			  30000006,
			  30000007,
			  30000008
			]
		  }`
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v1/universe/constellations/20000001/",
			httpmock.NewStringResponder(200, data2).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))

		data3 := `{
			"constellation_id": 20000001,
			"name": "Akpivem",
			"planets": [
			  {
				"moons": [
				  40000042
				],
				"planet_id": 40000041
			  },
			  {
				"planet_id": 40000043
			  }
			],
			"position": {
			  "x": -91174141133075340,
			  "y": 43938227486247170,
			  "z": -56482824383339900
			},
			"security_class": "B",
			"security_status": 0.8462923765182495,
			"star_id": 40000040,
			"stargates": [
			  50000342
			],
			"system_id": 30000003
		  }`
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v4/universe/systems/30000003/",
			httpmock.NewStringResponder(200, data3).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))

		// when
		x1, err := s.GetOrCreateEveSolarSystemESI(ctx, 30000003)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(30000003), x1.ID)
			assert.Equal(t, "Akpivem", x1.Name)
			assert.Equal(t, int32(20000001), x1.Constellation.ID)
			x2, err := r.GetEveSolarSystem(ctx, 30000003)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
}
