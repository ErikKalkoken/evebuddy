package app_test

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestAssetDisplayName(t *testing.T) {
	cases := []struct {
		name string
		ca   *app.Asset
		want string
	}{
		{"asset name", &app.Asset{Name: "name"}, "name"},
		{"type name", &app.Asset{Type: &app.EveType{Name: "type"}}, "type"},
		{
			"BPC name",
			&app.Asset{
				Type: &app.EveType{
					Name: "type",
				},
				IsBlueprintCopy: optional.New(true),
			},
			"type (Copy)",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
		xassert.Equal(t, tc.want, tc.ca.DisplayName())
		})
	}
}

func TestAssetDisplayName2(t *testing.T) {
	cases := []struct {
		name string
		ca   *app.Asset
		want string
	}{
		{
			"asset name",
			&app.Asset{
				Name: "name",
				Type: &app.EveType{Name: "type"},
			},
			"type \"name\"",
		},
		{
			"type name",
			&app.Asset{
				Type: &app.EveType{
					Name: "type",
				},
			},
			"type",
		},
		{
			"BPC name",
			&app.Asset{
				Type: &app.EveType{
					Name: "type",
				},
				IsBlueprintCopy: optional.New(true),
			},
			"type (Copy)",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
		xassert.Equal(t, tc.want, tc.ca.DisplayName2())
		})
	}
}

func TestAssetIsContainer(t *testing.T) {
	cases := []struct {
		IsSingleton   bool
		EveCategoryID int64
		want          bool
	}{
		{true, app.EveCategoryShip, true},
		{false, app.EveCategoryShip, false},
		{true, app.EveCategoryDrone, false},
	}
	for _, tc := range cases {
		t.Run("should report whether asset is a container", func(t *testing.T) {
			c := &app.EveCategory{ID: tc.EveCategoryID}
			g := &app.EveGroup{Category: c}
			typ := &app.EveType{Group: g}
			ca := app.Asset{IsSingleton: tc.IsSingleton, Type: typ}
		xassert.Equal(t, tc.want, ca.IsContainer())
		})
	}
}

func TestAssetTypeName(t *testing.T) {
	t.Run("has type", func(t *testing.T) {
		ca := &app.Asset{
			Type: &app.EveType{
				Name: "Alpha",
			},
		}
	xassert.Equal(t, "Alpha", ca.TypeName())
	})
	t.Run("no type", func(t *testing.T) {
		ca := &app.Asset{}
	xassert.Equal(t, "", ca.TypeName())
	})
}

func TestAssetVariant(t *testing.T) {
	cases := []struct {
		name string
		ca   *app.Asset
		want app.InventoryTypeVariant
	}{
		{
			"bpo",
			&app.Asset{
				Type: &app.EveType{
					Group: &app.EveGroup{
						Category: &app.EveCategory{
							ID: app.EveCategoryBlueprint,
						}}},
			},
			app.VariantBPO,
		},
		{
			"bpc",
			&app.Asset{
				Type: &app.EveType{
					Group: &app.EveGroup{
						Category: &app.EveCategory{
							ID: app.EveCategoryBlueprint,
						}}},
				IsBlueprintCopy: optional.New(true),
			},
			app.VariantBPC,
		},
		{
			"skin",
			&app.Asset{
				Type: &app.EveType{
					Group: &app.EveGroup{
						Category: &app.EveCategory{
							ID: app.EveCategorySKINs,
						}}},
			},
			app.VariantSKIN,
		},
		{
			"other",
			&app.Asset{
				Type: &app.EveType{
					Group: &app.EveGroup{
						Category: &app.EveCategory{
							ID: app.EveCategoryShip,
						}}},
			},
			app.VariantRegular,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
		xassert.Equal(t, tc.want, tc.ca.Variant())
		})
	}
}

// func TestCharacterAsset_LocationCategory(t *testing.T) {
// 	cases := []struct {
// 		locationFlag string
// 		want         app.LocationFlag
// 	}{
// 		{"Cargo", app.LocationCargoBay},
// 		{"Hangar", app.LocationHangar},
// 		{"AssetSafety", app.LocationAssetSafety},
// 		{"DroneBay", app.LocationDroneBay},
// 		{"FighterBay", app.LocationFighterBay},
// 		{"FighterTube1", app.LocationFighterBay},
// 		{"Hangar", app.LocationFitting},
// 		{"SpecializedFuelBay", app.LocationFuelBay},
// 		{"FrigateEscapeBay", app.LocationFrigateEscapeBay},
// 		{"Hangar", app.LocationOther},
// 	}
// 	for _, tc := range cases {
// 		t.Run(tc.locationFlag, func(t *testing.T) {
// 			ca := &app.Asset{LocationFlag: tc.locationFlag}
// 		xassert.Equal(t, tc.want, ca.IsInCargoBay())
// 		})
// 	}
// }
