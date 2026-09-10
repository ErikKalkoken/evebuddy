package eveuniverseservice_test

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/go-set"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestGetType(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
	t.Run("should return existing type", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		et := factory.CreateEveType()
		// when
		got, err := s.GetType(context.Background(), et.ID)
		// then
		require.NoError(t, err)
		xassert.Equal(t, et, got)
	})
	t.Run("should return error when type does not exist", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		_, err := s.GetType(context.Background(), 666)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
}

func TestListGroupsForCategory(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
	t.Run("should return groups for a category", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		category := factory.CreateEveCategory()
		g1 := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: category.ID})
		factory.CreateEveGroup() // other category
		// when
		oo, err := s.ListGroupsForCategory(context.Background(), category.ID)
		// then
		require.NoError(t, err)
		if assert.Len(t, oo, 1) {
			xassert.Equal(t, g1.ID, oo[0].ID)
		}
	})
}

func TestGetDogmaAttribute(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
	t.Run("should return existing dogma attribute", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		da := factory.CreateEveDogmaAttribute()
		// when
		got, err := s.GetDogmaAttribute(context.Background(), da.ID)
		// then
		require.NoError(t, err)
		xassert.Equal(t, da, got)
	})
	t.Run("should return error when dogma attribute does not exist", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		_, err := s.GetDogmaAttribute(context.Background(), 666)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
}

func TestFormatDogmaValueWrapper(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
	t.Run("should format a simple dogma value", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		text, iconID := s.FormatDogmaValue(context.Background(), 0.04, app.EveUnitAbsolutePercent)
		// then
		xassert.Equal(t, "4%", text)
		xassert.Equal(t, int64(0), iconID)
	})
}

func TestListTypeDogmaAttributesForType(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
	t.Run("should return dogma attributes for a type", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		et := factory.CreateEveType()
		da := factory.CreateEveDogmaAttribute()
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        et.ID,
			DogmaAttributeID: da.ID,
			Value:            42,
		})
		// when
		oo, err := s.ListTypeDogmaAttributesForType(context.Background(), et.ID)
		// then
		require.NoError(t, err)
		if assert.Len(t, oo, 1) {
			xassert.Equal(t, da.ID, oo[0].DogmaAttribute.ID)
			xassert.Equal(t, 42.0, oo[0].Value)
		}
	})
}

func TestUpdateCategoryWithChildrenESI(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
	ctx := context.Background()
	t.Run("should fetch category with its groups and types from ESI", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi.evetech.net/universe/categories/\d+`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"category_id": 6,
				"groups":      []int{25},
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
				"types":       []int{587},
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
		err := s.UpdateCategoryWithChildrenESI(ctx, 6)
		// then
		require.NoError(t, err)
		_, err = st.GetEveType(ctx, 587)
		assert.NoError(t, err)
	})
}

func TestGetOrCreateEveCategoryESI(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
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
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
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
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
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
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
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

func TestGetOrCreateEveDogmaAttributeESI(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
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
			xassert.EqualOptional(t, 1.0, x1.DefaultValue)
			xassert.EqualOptional(t, "Factor by which top speed increases.", x1.Description)
			xassert.EqualOptional(t, "Maximum Velocity Bonus", x1.DisplayName)
			xassert.EqualOptional(t, int64(1389), x1.IconID)
			xassert.EqualOptional(t, "speedFactor", x1.Name)
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
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
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
			xassert.Empty(t, x)
		}
	})
}

func TestUpdateEveMarketPricesESI(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
	ctx := context.Background()
	const (
		knownTypeID = 32772
		otherTypeID = 10001
	)
	t.Run("should create new objects from ESI", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateEveType(storage.CreateEveTypeParams{
			ID: knownTypeID,
		})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/markets/prices",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"adjusted_price": 306988.09,
				"average_price":  306292.67,
				"type_id":        knownTypeID,
			}, {
				"adjusted_price": 123.45,
				"average_price":  456.78,
				"type_id":        otherTypeID,
			}}),
		)
		// when
		got, err := s.UpdateMarketPricesESI(ctx)
		// then
		require.NoError(t, err)
		want := set.Of[int64](knownTypeID)
		xassert.Equal(t, want, got)
		prices, err := st.ListEveMarketPrices(ctx)
		require.NoError(t, err)
		require.Len(t, prices, 2)
		for _, o := range prices {
			switch o.TypeID {
			case knownTypeID:
				xassert.EqualOptional(t, 306988.09, o.AdjustedPrice)
				xassert.EqualOptional(t, 306292.67, o.AveragePrice)
			case o.TypeID:
				xassert.EqualOptional(t, 123.45, o.AdjustedPrice)
				xassert.EqualOptional(t, 456.78, o.AveragePrice)
			}
		}
	})
	t.Run("should update existing objects from ESI", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateEveType(storage.CreateEveTypeParams{
			ID: knownTypeID,
		})
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:        knownTypeID,
			AdjustedPrice: optional.New(2.0),
			AveragePrice:  optional.New(3.0),
		})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/markets/prices",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"adjusted_price": 306988.09,
				"average_price":  306292.67,
				"type_id":        knownTypeID,
			}}),
		)
		// when
		got, err := s.UpdateMarketPricesESI(ctx)
		// then
		require.NoError(t, err)
		want := set.Of[int64](knownTypeID)
		xassert.Equal(t, want, got)
		o, err := st.GetEveMarketPrice(ctx, knownTypeID)
		require.NoError(t, err)
		xassert.EqualOptional(t, 306988.09, o.AdjustedPrice)
		xassert.EqualOptional(t, 306292.67, o.AveragePrice)
	})
	t.Run("should only report changes for known types", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateEveType(storage.CreateEveTypeParams{
			ID: knownTypeID,
		})
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:        knownTypeID,
			AdjustedPrice: optional.New(306988.09),
			AveragePrice:  optional.New(306292.67),
		})
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:        otherTypeID,
			AdjustedPrice: optional.New(111.09),
			AveragePrice:  optional.New(222.67),
		})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/markets/prices",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"adjusted_price": 306988.09,
				"average_price":  306292.67,
				"type_id":        knownTypeID,
			}, {
				"adjusted_price": 123.45,
				"average_price":  456.78,
				"type_id":        otherTypeID,
			}}),
		)
		// when
		got, err := s.UpdateMarketPricesESI(ctx)
		// then
		require.NoError(t, err)
		want := set.Of[int64]()
		xassert.Equal(t, want, got)
	})

	t.Run("should remove obsolete prices", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			AdjustedPrice: optional.New(111.09),
			AveragePrice:  optional.New(222.67),
		})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/markets/prices",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"adjusted_price": 306988.09,
				"average_price":  306292.67,
				"type_id":        knownTypeID,
			}}),
		)
		// when
		_, err := s.UpdateMarketPricesESI(ctx)
		// then
		require.NoError(t, err)
		got, err := st.ListEveMarketPriceIDs(ctx)
		require.NoError(t, err)
		want := set.Of[int64](knownTypeID)
		xassert.Equal(t, want, got)
	})
}
