package eveuniverseservice_test

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/go-set"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestGetOrCreateEveCategoryESI(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("should return existing category", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		factory.CreateEveCategory(storage.CreateEveCategoryParams{ID: 6})
		// when
		x1, err := s.GetOrCreateCategoryESI(ctx, 6)
		// then
		if assert.NoError(t, err) {
			xassert.Equal(t, int64(6), x1.ID)
		}
	})
	t.Run("should fetch category from ESI and create it", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi.evetech.net/universe/categories/\d+`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"category_id": 6,
				"groups":      []int{25, 26, 27},
				"name":        "Ship",
				"published":   true,
			}))

		// when
		x1, err := s.GetOrCreateCategoryESI(ctx, 6)
		// then
		if assert.NoError(t, err) {
			xassert.Equal(t, int64(6), x1.ID)
			xassert.Equal(t, "Ship", x1.Name)
			xassert.Equal(t, true, x1.IsPublished)
			x2, err := st.GetEveCategory(ctx, 6)
			if assert.NoError(t, err) {
				xassert.Equal(t, x1, x2)
			}
		}
	})
}

func TestGetOrCreateEveGroupESI(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("should return existing group", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		factory.CreateEveGroup(storage.CreateEveGroupParams{ID: 25})
		// when
		x1, err := s.GetOrCreateGroupESI(ctx, 25)
		// then
		if assert.NoError(t, err) {
			xassert.Equal(t, int64(25), x1.ID)
		}
	})
	t.Run("should fetch group from ESI and create it", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		factory.CreateEveCategory(storage.CreateEveCategoryParams{ID: 6})
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi.evetech.net/universe/groups/\d+`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"category_id": 6,
				"group_id":    25,
				"name":        "Frigate",
				"published":   true,
				"types":       []int64{587, 586, 585},
			}))

		// when
		x1, err := s.GetOrCreateGroupESI(ctx, 25)
		// then
		if assert.NoError(t, err) {
			xassert.Equal(t, int64(25), x1.ID)
			xassert.Equal(t, "Frigate", x1.Name)
			xassert.Equal(t, int64(6), x1.Category.ID)
			xassert.Equal(t, true, x1.IsPublished)
			x2, err := st.GetEveGroup(ctx, 25)
			if assert.NoError(t, err) {
				xassert.Equal(t, x1, x2)
			}
		}
	})
}

func TestGetOrCreateEveTypeESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("should return existing type", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 587})
		// when
		x1, err := s.GetOrCreateTypeESI(ctx, 587)
		// then
		if assert.NoError(t, err) {
			xassert.Equal(t, int64(587), x1.ID)
		}
	})
	t.Run("should fetch type from ESI and create it", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		factory.CreateEveGroup(storage.CreateEveGroupParams{ID: 25})
		factory.CreateEveDogmaAttribute(storage.CreateEveDogmaAttributeParams{ID: 161})
		factory.CreateEveDogmaAttribute(storage.CreateEveDogmaAttributeParams{ID: 162})
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi.evetech.net/universe/types/\d+`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"description": "The Rifter is a...",
				"dogma_attributes": []map[string]any{
					{
						"attribute_id": 161,
						"value":        11,
					},
					{
						"attribute_id": 162,
						"value":        12,
					},
				},
				"dogma_effects": []map[string]any{
					{
						"effect_id":  111,
						"is_default": true,
					},
					{
						"effect_id":  112,
						"is_default": false,
					},
				},
				"group_id":  25,
				"name":      "Rifter",
				"published": true,
				"type_id":   587,
			}),
		)
		// when
		x1, err := s.GetOrCreateTypeESI(ctx, 587)
		// then
		if assert.NoError(t, err) {
			xassert.Equal(t, int64(587), x1.ID)
			xassert.Equal(t, "Rifter", x1.Name)
			xassert.Equal(t, int64(25), x1.Group.ID)
			xassert.Equal(t, true, x1.IsPublished)
			x2, err := st.GetEveType(ctx, 587)
			if assert.NoError(t, err) {
				xassert.Equal(t, x1, x2)
			}
			y, err := st.GetEveTypeDogmaAttribute(ctx, 587, 161)
			if assert.NoError(t, err) {
				xassert.Equal(t, 11.0, y)
			}
			z, err := st.GetEveTypeDogmaEffect(ctx, 587, 111)
			if assert.NoError(t, err) {
				assert.True(t, z)
			}

		}
	})
	t.Run("should fetch group from ESI and create it (integration)", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi.evetech.net/universe/categories/\d+`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"category_id": 6,
				"groups":      []int{25, 26, 27},
				"name":        "Ship",
				"published":   true,
			}),
		)
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi.evetech.net/universe/groups/\d+`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"category_id": 6,
				"group_id":    25,
				"name":        "Frigate",
				"published":   true,
				"types":       []int{587, 586, 585},
			}),
		)
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi.evetech.net/universe/types/\d+`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"description": "The Rifter is a...",
				"group_id":    25,
				"name":        "Rifter",
				"published":   true,
				"type_id":     587,
			}),
		)
		// when
		x1, err := s.GetOrCreateTypeESI(ctx, 587)
		// then
		if assert.NoError(t, err) {
			xassert.Equal(t, int64(587), x1.ID)
			xassert.Equal(t, "Rifter", x1.Name)
			xassert.Equal(t, int64(25), x1.Group.ID)
			xassert.Equal(t, true, x1.IsPublished)
			x2, err := st.GetEveType(ctx, 587)
			if assert.NoError(t, err) {
				xassert.Equal(t, x1, x2)
			}
		}
	})
}

func TestAddMissingEveTypes(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	t.Run("do nothing when all types already exist", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		x1 := factory.CreateEveType()
		// when
		err := s.AddMissingTypes(ctx, set.Of(x1.ID))
		// then
		if assert.NoError(t, err) {
			xassert.Equal(t, 0, httpmock.GetTotalCallCount())
		}
	})
	t.Run("ignore invalid IDs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		x1 := factory.CreateEveType()
		// when
		err := s.AddMissingTypes(ctx, set.Of(x1.ID, 0))
		// then
		if assert.NoError(t, err) {
			xassert.Equal(t, 0, httpmock.GetTotalCallCount())
		}
	})
}

func TestGetOrCreateEveRaceESI(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("should return existing race", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		x1 := factory.CreateEveRace(app.EveRace{ID: 7})
		// when
		x2, err := s.GetOrCreateRaceESI(ctx, 7)
		// then
		if assert.NoError(t, err) {
			xassert.Equal(t, x1, x2)
		}
	})
	t.Run("should create race from ESI when it does not exit in DB", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/universe/races",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"alliance_id": 500001,
					"description": "Founded on the tenets of patriotism and hard work...",
					"name":        "Caldari",
					"race_id":     7,
				},
			}))

		// when
		x1, err := s.GetOrCreateRaceESI(ctx, 7)
		// then
		if assert.NoError(t, err) {
			xassert.Equal(t, "Caldari", x1.Name)
			xassert.Equal(t, "Founded on the tenets of patriotism and hard work...", x1.Description)
			x2, err := st.GetEveRace(ctx, 7)
			if assert.NoError(t, err) {
				xassert.Equal(t, x1, x2)
			}
		}
	})
	t.Run("should return specific error when race ID is invalid", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/universe/races",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"alliance_id": 500001,
					"description": "Founded on the tenets of patriotism and hard work...",
					"name":        "Caldari",
					"race_id":     7,
				},
			}))

		// when
		_, err := s.GetOrCreateRaceESI(ctx, 42)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
}

func TestGetOrCreateEveDogmaAttributeESI(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("should return existing object", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		x1 := factory.CreateEveDogmaAttribute()
		// when
		x2, err := s.GetOrCreateDogmaAttributeESI(ctx, x1.ID)
		// then
		if assert.NoError(t, err) {
			xassert.Equal(t, x2, x1)
		}
	})
	t.Run("should create new object from ESI when it does not exist", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/dogma/attributes/20",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"attribute_id":  20,
				"default_value": 1,
				"description":   "Factor by which top speed increases.",
				"display_name":  "Maximum Velocity Bonus",
				"high_is_good":  true,
				"icon_id":       1389,
				"name":          "speedFactor",
				"published":     true,
				"unit_id":       124,
			}))
		// when
		x1, err := s.GetOrCreateDogmaAttributeESI(ctx, 20)
		// then
		if assert.NoError(t, err) {
			xassert.Equal(t, int64(20), x1.ID)
			xassert.Equal(t, 1.0, x1.DefaultValue.ValueOrZero())
			xassert.Equal(t, "Factor by which top speed increases.", x1.Description.ValueOrZero())
			xassert.Equal(t, "Maximum Velocity Bonus", x1.DisplayName.ValueOrZero())
			xassert.Equal(t, int64(1389), x1.IconID.ValueOrZero())
			xassert.Equal(t, "speedFactor", x1.Name.ValueOrZero())
			assert.True(t, x1.IsHighGood.ValueOrZero())
			assert.True(t, x1.IsPublished.ValueOrZero())
			assert.False(t, x1.IsStackable.ValueOrZero())
			xassert.Equal(t, app.EveUnitID(124), x1.Unit)
			x2, err := st.GetEveDogmaAttribute(ctx, 20)
			if assert.NoError(t, err) {
				xassert.Equal(t, x1, x2)
			}
		}
	})
}

func TestMarketPrice(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("return price when it exists", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		o := factory.CreateEveType()
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:       o.ID,
			AveragePrice: optional.New(12.34),
		})
		x, err := s.MarketPrice(ctx, o.ID)
		if assert.NoError(t, err) {
			assert.InDelta(t, 12.34, x.MustValue(), 0.01)
		}
	})
	t.Run("return empty when no price exists", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		o := factory.CreateEveType()
		x, err := s.MarketPrice(ctx, o.ID)
		if assert.NoError(t, err) {
			assert.True(t, x.IsEmpty())
		}
	})
}
