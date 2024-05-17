package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func TestGetOrCreateStructureESI(t *testing.T) {
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
		factory.CreateStructure(storage.CreateStructureParams{ID: 42, Name: "Alpha"})
		// when
		x, err := s.getOrCreateStructureESI(ctx, 42)
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
			"https://esi.evetech.net/v2/universe/structures/42/",
			httpmock.NewStringResponder(200, data).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))

		// when
		x1, err := s.getOrCreateStructureESI(ctx, 42)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int64(42), x1.ID)
			assert.Equal(t, "V-3YG7 VI - The Capital", x1.Name)
			assert.Equal(t, owner, x1.Owner)
			assert.Equal(t, system, x1.SolarSystem)
			assert.Equal(t, myType, x1.Type)
			assert.Equal(t, model.Position{X: 1.1, Y: 2.2, Z: 3.3}, x1.Position)
			x2, err := r.GetStructure(ctx, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
}
