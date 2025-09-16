package app_test

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/stretchr/testify/assert"
)

func TestCharacterAssetDisplayName(t *testing.T) {
	cases := []struct {
		name string
		ca   *app.CharacterAsset
		want string
	}{
		{"asset name", &app.CharacterAsset{Name: "name"}, "name"},
		{"type name", &app.CharacterAsset{Type: &app.EveType{Name: "type"}}, "type"},
		{
			"BPC name",
			&app.CharacterAsset{
				Type: &app.EveType{
					Name: "type",
				},
				IsBlueprintCopy: true,
			},
			"type (Copy)",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.ca.DisplayName())
		})
	}
}

func TestCharacterAssetDisplayName2(t *testing.T) {
	cases := []struct {
		name string
		ca   *app.CharacterAsset
		want string
	}{
		{
			"asset name",
			&app.CharacterAsset{
				Name: "name",
				Type: &app.EveType{Name: "type"},
			},
			"type \"name\"",
		},
		{
			"type name",
			&app.CharacterAsset{
				Type: &app.EveType{
					Name: "type",
				},
			},
			"type",
		},
		{
			"BPC name",
			&app.CharacterAsset{
				Type: &app.EveType{
					Name: "type",
				},
				IsBlueprintCopy: true,
			},
			"type (Copy)",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.ca.DisplayName2())
		})
	}
}

func TestCharacterAssetIsContainer(t *testing.T) {
	cases := []struct {
		IsSingleton   bool
		EveCategoryID int32
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
			ca := app.CharacterAsset{IsSingleton: tc.IsSingleton, Type: typ}
			assert.Equal(t, tc.want, ca.IsContainer())
		})
	}
}

func TestCharacterAssetTypeName(t *testing.T) {
	t.Run("has type", func(t *testing.T) {
		ca := &app.CharacterAsset{
			Type: &app.EveType{
				Name: "Alpha",
			},
		}
		assert.Equal(t, "Alpha", ca.TypeName())
	})
	t.Run("no type", func(t *testing.T) {
		ca := &app.CharacterAsset{}
		assert.Equal(t, "", ca.TypeName())
	})
}

func TestCharacterAssetVariant(t *testing.T) {
	cases := []struct {
		name string
		ca   *app.CharacterAsset
		want app.EveTypeVariant
	}{
		{
			"bpo",
			&app.CharacterAsset{
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
			&app.CharacterAsset{
				Type: &app.EveType{
					Group: &app.EveGroup{
						Category: &app.EveCategory{
							ID: app.EveCategoryBlueprint,
						}}},
				IsBlueprintCopy: true,
			},
			app.VariantBPC,
		},
		{
			"skin",
			&app.CharacterAsset{
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
			&app.CharacterAsset{
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
			assert.Equal(t, tc.want, tc.ca.Variant())
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
// 			ca := &app.CharacterAsset{LocationFlag: tc.locationFlag}
// 			assert.Equal(t, tc.want, ca.IsInCargoBay())
// 		})
// 	}
// }
