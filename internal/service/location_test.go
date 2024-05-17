package service

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

const (
	stationID   = 60000277
	structureID = 1_000_000_000_009
)

func TestLocationStations(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewService(r)
	ctx := context.Background()
	t.Run("should fetch station from ESI and create it", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		owner := factory.CreateEveEntityCorporation(model.EveEntity{ID: 1000003})
		system := factory.CreateEveSolarSystem(model.EveSolarSystem{ID: 30000148})
		myType := factory.CreateEveType(model.EveType{ID: 1531})
		data := `{
			"max_dockable_ship_volume": 50000000,
			"name": "Jakanerva III - Moon 15 - Prompt Delivery Storage",
			"office_rental_cost": 10000,
			"owner": 1000003,
			"position": {
			  "x": 165632286720,
			  "y": 2771804160,
			  "z": -2455331266560
			},
			"race_id": 1,
			"reprocessing_efficiency": 0.5,
			"reprocessing_stations_take": 0.05,
			"services": [
			  "courier-missions",
			  "reprocessing-plant",
			  "market",
			  "repair-facilities",
			  "fitting",
			  "news",
			  "storage",
			  "insurance",
			  "docking",
			  "office-rental",
			  "loyalty-point-store",
			  "navy-offices"
			],
			"station_id": 60000277,
			"system_id": 30000148,
			"type_id": 1531
		  }`
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v2/universe/stations/%d/", stationID),
			httpmock.NewStringResponder(http.StatusOK, data).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))
		// when
		x1, err := s.getOrCreateLocationESI(ctx, stationID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int64(stationID), x1.ID)
			assert.Equal(t, "Jakanerva III - Moon 15 - Prompt Delivery Storage", x1.Name)
			assert.Equal(t, owner, x1.Owner)
			assert.Equal(t, system, x1.SolarSystem)
			assert.Equal(t, myType, x1.Type)
			x2, err := r.GetLocation(ctx, stationID)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
}

func TestLocationStructures(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewService(r)
	ctx := context.Background()
	t.Run("should return existing structure", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		factory.CreateLocationStructure(storage.UpdateOrCreateLocationParams{ID: structureID, Name: "Alpha"})
		// when
		x, err := s.getOrCreateLocationESI(ctx, structureID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, "Alpha", x.Name)
		}
	})
	t.Run("should fetch structure from ESI and create it", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		owner := factory.CreateEveEntityCorporation(model.EveEntity{ID: 109299958})
		system := factory.CreateEveSolarSystem(model.EveSolarSystem{ID: 30000142})
		myType := factory.CreateEveType(model.EveType{ID: 99})
		data := `{
			"name": "V-3YG7 VI - The Capital",
			"owner_id": 109299958,
			"solar_system_id": 30000142,
			"type_id": 99,
			"position": {
				"x": 1.1,
				"y": 2.2,
				"z": 3.3
			}
		  }`
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v2/universe/structures/%d/", structureID),
			httpmock.NewStringResponder(http.StatusOK, data).HeaderSet(
				http.Header{"Content-Type": []string{"application/json"}}))

		// when
		x1, err := s.getOrCreateLocationESI(ctx, structureID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int64(structureID), x1.ID)
			assert.Equal(t, "V-3YG7 VI - The Capital", x1.Name)
			assert.Equal(t, owner, x1.Owner)
			assert.Equal(t, system, x1.SolarSystem)
			assert.Equal(t, myType, x1.Type)
			x2, err := r.GetLocation(ctx, structureID)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
	t.Run("should create empty structure from ESI when no access", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		data := `{
			"error": "forbidden"
			}`
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v2/universe/structures/%d/", structureID),
			httpmock.NewStringResponder(http.StatusForbidden, data).HeaderSet(
				http.Header{"Content-Type": []string{"application/json"}}))

		// when
		x1, err := s.getOrCreateLocationESI(ctx, structureID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int64(structureID), x1.ID)
			assert.Equal(t, "", x1.Name)
			assert.Nil(t, x1.Owner)
			assert.Nil(t, x1.SolarSystem)
			assert.Nil(t, x1.Type)
			x2, err := r.GetLocation(ctx, structureID)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
	t.Run("should return error when other http error occurs", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		data := `{
			"error": "xxx"
			}`
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v2/universe/structures/%d/", structureID),
			httpmock.NewStringResponder(http.StatusNotFound, data).HeaderSet(
				http.Header{"Content-Type": []string{"application/json"}}))

		// when
		_, err := s.getOrCreateLocationESI(ctx, structureID)
		// then
		assert.Error(t, err)
	})
}
