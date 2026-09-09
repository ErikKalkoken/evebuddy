package app_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestEveTypeDescriptionPlain(t *testing.T) {
	et := app.EveType{Description: "alpha<br>bravo"}
	xassert.Equal(t, "alpha\nbravo", et.DescriptionPlain())
}

func TestEveTypeIsBlueprint(t *testing.T) {
	et1 := app.EveType{Group: &app.EveGroup{Category: &app.EveCategory{ID: app.EveCategoryBlueprint}}}
	assert.True(t, et1.IsBlueprint())
	et2 := app.EveType{Group: &app.EveGroup{Category: &app.EveCategory{ID: app.EveCategoryShip}}}
	assert.False(t, et2.IsBlueprint())
}

func TestEveTypeIsSKIN(t *testing.T) {
	et1 := app.EveType{Group: &app.EveGroup{Category: &app.EveCategory{ID: app.EveCategorySKINs}}}
	assert.True(t, et1.IsSKIN())
	et2 := app.EveType{Group: &app.EveGroup{Category: &app.EveCategory{ID: app.EveCategoryShip}}}
	assert.False(t, et2.IsSKIN())
}

func TestEveTypeIsTradeable(t *testing.T) {
	et1 := app.EveType{MarketGroupID: optional.New[int64](42)}
	assert.True(t, et1.IsTradeable())
	et2 := app.EveType{}
	assert.False(t, et2.IsTradeable())
}

func TestEveTypeHasFuelBay(t *testing.T) {
	cases := []struct {
		name          string
		eveCategoryID int64
		eveGroupID    int64
		want          bool
	}{
		{"not a ship", app.EveCategoryDrone, app.EveGroupCarrier, false},
		{"carrier", app.EveCategoryShip, app.EveGroupCarrier, true},
		{"titan", app.EveCategoryShip, app.EveGroupTitan, true},
		{"regular ship", app.EveCategoryShip, app.EveGroupCharacter, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			et := app.EveType{
				Group: &app.EveGroup{
					ID:       tc.eveGroupID,
					Category: &app.EveCategory{ID: tc.eveCategoryID},
				},
			}
			xassert.Equal(t, tc.want, et.HasFuelBay())
		})
	}
}

func TestEveTypeHasRender(t *testing.T) {
	cases := []struct {
		name          string
		eveCategoryID int64
		want          bool
	}{
		{"ship", app.EveCategoryShip, true},
		{"drone", app.EveCategoryDrone, true},
		{"skill", app.EveCategorySkill, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			et := app.EveType{Group: &app.EveGroup{Category: &app.EveCategory{ID: tc.eveCategoryID}}}
			xassert.Equal(t, tc.want, et.HasRender())
		})
	}
}

func TestEveTypeIcon(t *testing.T) {
	t.Run("no icon ID", func(t *testing.T) {
		et := app.EveType{}
		res, ok := et.Icon()
		assert.False(t, ok)
		assert.Nil(t, res)
	})
	t.Run("icon not found", func(t *testing.T) {
		et := app.EveType{IconID: optional.New[int64](-999)}
		res, ok := et.Icon()
		assert.False(t, ok)
		assert.Nil(t, res)
	})
	t.Run("icon found", func(t *testing.T) {
		et := app.EveType{IconID: optional.New[int64](0)}
		res, ok := et.Icon()
		assert.True(t, ok)
		assert.NotNil(t, res)
	})
}

func TestEveTypeToEveEntity(t *testing.T) {
	et := app.EveType{ID: 42, Name: "name"}
	got := et.ToEveEntity()
	xassert.Equal(t, int64(42), got.ID)
	xassert.Equal(t, "name", got.Name)
	xassert.Equal(t, app.EveEntityInventoryType, got.Category)
}

func TestEveMarketPriceEqual(t *testing.T) {
	cases := []struct {
		name string
		a    app.EveMarketPrice
		b    app.EveMarketPrice
		want bool
	}{
		{
			"equal",
			app.EveMarketPrice{TypeID: 1, AdjustedPrice: optional.New(1.0), AveragePrice: optional.New(2.0)},
			app.EveMarketPrice{TypeID: 1, AdjustedPrice: optional.New(1.0), AveragePrice: optional.New(2.0)},
			true,
		},
		{
			"different type",
			app.EveMarketPrice{TypeID: 1},
			app.EveMarketPrice{TypeID: 2},
			false,
		},
		{
			"different adjusted price",
			app.EveMarketPrice{TypeID: 1, AdjustedPrice: optional.New(1.0)},
			app.EveMarketPrice{TypeID: 1, AdjustedPrice: optional.New(2.0)},
			false,
		},
		{
			"different average price",
			app.EveMarketPrice{TypeID: 1, AveragePrice: optional.New(1.0)},
			app.EveMarketPrice{TypeID: 1, AveragePrice: optional.New(2.0)},
			false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			xassert.Equal(t, tc.want, tc.a.Equal(tc.b))
		})
	}
}
