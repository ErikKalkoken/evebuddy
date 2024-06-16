package eveuniverse_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/antihax/goesi"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverse"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func TestGetOrCreateEveCategoryESI(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := goesi.NewAPIClient(nil, "")
	s := eveuniverse.New(r, client, nil, nil)
	ctx := context.Background()
	t.Run("should return existing category", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		factory.CreateEveCategory(storage.CreateEveCategoryParams{ID: 6})
		// when
		x1, err := s.GetOrCreateEveCategoryESI(ctx, 6)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(6), x1.ID)
		}
	})
	t.Run("should fetch category from ESI and create it", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		data := `{
			"category_id": 6,
			"groups": [
			  25,
			  26,
			  27
			],
			"name": "Ship",
			"published": true
		  }`
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v1/universe/categories/6/",
			httpmock.NewStringResponder(200, data).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))

		// when
		x1, err := s.GetOrCreateEveCategoryESI(ctx, 6)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(6), x1.ID)
			assert.Equal(t, "Ship", x1.Name)
			assert.Equal(t, true, x1.IsPublished)
			x2, err := r.GetEveCategory(ctx, 6)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
}

func TestGetOrCreateEveGroupESI(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := goesi.NewAPIClient(nil, "")
	s := eveuniverse.New(r, client, nil, nil)
	ctx := context.Background()
	t.Run("should return existing group", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		factory.CreateEveGroup(storage.CreateEveGroupParams{ID: 25})
		// when
		x1, err := s.GetOrCreateEveGroupESI(ctx, 25)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(25), x1.ID)
		}
	})
	t.Run("should fetch group from ESI and create it", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		factory.CreateEveCategory(storage.CreateEveCategoryParams{ID: 6})
		data := `{
			"category_id": 6,
			"group_id": 25,
			"name": "Frigate",
			"published": true,
			"types": [
			  587,
			  586,
			  585
			]
		  }`
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v1/universe/groups/25/",
			httpmock.NewStringResponder(200, data).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))

		// when
		x1, err := s.GetOrCreateEveGroupESI(ctx, 25)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(25), x1.ID)
			assert.Equal(t, "Frigate", x1.Name)
			assert.Equal(t, int32(6), x1.Category.ID)
			assert.Equal(t, true, x1.IsPublished)
			x2, err := r.GetEveGroup(ctx, 25)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
}

func TestGetOrCreateEveTypeESI(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := goesi.NewAPIClient(nil, "")
	s := eveuniverse.New(r, client, nil, nil)
	ctx := context.Background()
	t.Run("should return existing type", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 587})
		// when
		x1, err := s.GetOrCreateEveTypeESI(ctx, 587)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(587), x1.ID)
		}
	})
	t.Run("should fetch type from ESI and create it", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		factory.CreateEveGroup(storage.CreateEveGroupParams{ID: 25})
		factory.CreateEveDogmaAttribute(storage.CreateEveDogmaAttributeParams{ID: 161})
		factory.CreateEveDogmaAttribute(storage.CreateEveDogmaAttributeParams{ID: 162})
		data := `{
			"description": "The Rifter is a...",
			"dogma_attributes": [
				{
					"attribute_id": 161,
					"value": 11
					},
				{
					"attribute_id": 162,
					"value": 12
				}
			],
			"dogma_effects": [
				{
					"effect_id": 111,
					"is_default": true
					},
				{
					"effect_id": 112,
					"is_default": false
				}
			],
			"group_id": 25,
			"name": "Rifter",
			"published": true,
			"type_id": 587
		  }`
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v3/universe/types/587/",
			httpmock.NewStringResponder(200, data).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))

		// when
		x1, err := s.GetOrCreateEveTypeESI(ctx, 587)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(587), x1.ID)
			assert.Equal(t, "Rifter", x1.Name)
			assert.Equal(t, int32(25), x1.Group.ID)
			assert.Equal(t, true, x1.IsPublished)
			x2, err := r.GetEveType(ctx, 587)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
			y, err := r.GetEveTypeDogmaAttribute(ctx, 587, 161)
			if assert.NoError(t, err) {
				assert.Equal(t, float32(11), y)
			}
			z, err := r.GetEveTypeDogmaEffect(ctx, 587, 111)
			if assert.NoError(t, err) {
				assert.True(t, z)
			}

		}
	})
	t.Run("should fetch group from ESI and create it (integration)", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()

		data1 := `{
			"category_id": 6,
			"groups": [
			  25,
			  26,
			  27
			],
			"name": "Ship",
			"published": true
		  }`
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v1/universe/categories/6/",
			httpmock.NewStringResponder(200, data1).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))

		data2 := `{
			"category_id": 6,
			"group_id": 25,
			"name": "Frigate",
			"published": true,
			"types": [
			  587,
			  586,
			  585
			]
		  }`
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v1/universe/groups/25/",
			httpmock.NewStringResponder(200, data2).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))

		data3 := `{
			"description": "The Rifter is a...",
			"group_id": 25,
			"name": "Rifter",
			"published": true,
			"type_id": 587
		  }`
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v3/universe/types/587/",
			httpmock.NewStringResponder(200, data3).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))

		// when
		x1, err := s.GetOrCreateEveTypeESI(ctx, 587)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(587), x1.ID)
			assert.Equal(t, "Rifter", x1.Name)
			assert.Equal(t, int32(25), x1.Group.ID)
			assert.Equal(t, true, x1.IsPublished)
			x2, err := r.GetEveType(ctx, 587)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
}
