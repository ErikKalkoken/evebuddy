package service_test

import (
	"context"
	"example/evebuddy/internal/model"
	"example/evebuddy/internal/service"
	"example/evebuddy/internal/testutil"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestGetOrCreateEveCategoryESI(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := service.NewService(r)
	ctx := context.Background()
	t.Run("should return existing category", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		factory.CreateEveCategory(model.EveCategory{ID: 6})
		// when
		x1, err := s.GetOrCreateEveCategoryESI(6)
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
		x1, err := s.GetOrCreateEveCategoryESI(6)
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
	s := service.NewService(r)
	ctx := context.Background()
	t.Run("should return existing group", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		factory.CreateEveGroup(model.EveGroup{ID: 25})
		// when
		x1, err := s.GetOrCreateEveGroupESI(25)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(25), x1.ID)
		}
	})
	t.Run("should fetch group from ESI and create it", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		factory.CreateEveCategory(model.EveCategory{ID: 6})
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
		x1, err := s.GetOrCreateEveGroupESI(25)
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
	s := service.NewService(r)
	ctx := context.Background()
	t.Run("should return existing type", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		factory.CreateEveType(model.EveType{ID: 587})
		// when
		x1, err := s.GetOrCreateEveTypeESI(587)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(587), x1.ID)
		}
	})
	t.Run("should fetch group from ESI and create it", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		factory.CreateEveGroup(model.EveGroup{ID: 25})
		data := `{
			"description": "The Rifter is a...",
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
		x1, err := s.GetOrCreateEveTypeESI(587)
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
		x1, err := s.GetOrCreateEveTypeESI(587)
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
