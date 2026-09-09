package app_test

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestInventoryTypeVariantString(t *testing.T) {
	cases := []struct {
		v    app.InventoryTypeVariant
		want string
	}{
		{app.VariantRegular, "Regular"},
		{app.VariantBPO, "BPO"},
		{app.VariantBPC, "BPC"},
		{app.VariantSKIN, "SKIN"},
		{app.InventoryTypeVariant(99), ""},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			xassert.Equal(t, tc.want, tc.v.String())
		})
	}
}

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

func TestAssetDisplayName3(t *testing.T) {
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
			"name (type)",
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
			xassert.Equal(t, tc.want, tc.ca.DisplayName3())
		})
	}
}

func TestAssetID(t *testing.T) {
	ca := app.Asset{ItemID: 42}
	xassert.Equal(t, int64(42), ca.ID())
}

func TestAssetUnwrap(t *testing.T) {
	ca := app.Asset{ItemID: 42}
	xassert.Equal(t, ca, ca.Unwrap())
}

func TestAssetCanHaveName(t *testing.T) {
	cases := []struct {
		name        string
		IsSingleton bool
		Type        *app.EveType
		want        bool
	}{
		{"not singleton", false, &app.EveType{Group: &app.EveGroup{Category: &app.EveCategory{ID: app.EveCategoryShip}}}, false},
		{"no type", true, nil, false},
		{"ship category", true, &app.EveType{Group: &app.EveGroup{Category: &app.EveCategory{ID: app.EveCategoryShip}}}, true},
		{"cargo container group", true, &app.EveType{Group: &app.EveGroup{ID: app.EveGroupCargoContainer, Category: &app.EveCategory{ID: app.EveCategoryDrone}}}, true},
		{"neither", true, &app.EveType{Group: &app.EveGroup{Category: &app.EveCategory{ID: app.EveCategoryDrone}}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ca := app.Asset{IsSingleton: tc.IsSingleton, Type: tc.Type}
			xassert.Equal(t, tc.want, ca.CanHaveName())
		})
	}
}

func TestAssetIsBPO(t *testing.T) {
	cases := []struct {
		name string
		ca   app.Asset
		want bool
	}{
		{"no type", app.Asset{}, false},
		{
			"blueprint original",
			app.Asset{Type: &app.EveType{Group: &app.EveGroup{Category: &app.EveCategory{ID: app.EveCategoryBlueprint}}}},
			true,
		},
		{
			"blueprint copy",
			app.Asset{
				Type:            &app.EveType{Group: &app.EveGroup{Category: &app.EveCategory{ID: app.EveCategoryBlueprint}}},
				IsBlueprintCopy: optional.New(true),
			},
			false,
		},
		{
			"not a blueprint",
			app.Asset{Type: &app.EveType{Group: &app.EveGroup{Category: &app.EveCategory{ID: app.EveCategoryShip}}}},
			false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			xassert.Equal(t, tc.want, tc.ca.IsBPO())
		})
	}
}

func TestAssetIsSKIN(t *testing.T) {
	cases := []struct {
		name string
		ca   app.Asset
		want bool
	}{
		{"no type", app.Asset{}, false},
		{
			"skin",
			app.Asset{Type: &app.EveType{Group: &app.EveGroup{Category: &app.EveCategory{ID: app.EveCategorySKINs}}}},
			true,
		},
		{
			"not a skin",
			app.Asset{Type: &app.EveType{Group: &app.EveGroup{Category: &app.EveCategory{ID: app.EveCategoryShip}}}},
			false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			xassert.Equal(t, tc.want, tc.ca.IsSKIN())
		})
	}
}

func TestAssetLocationCategory(t *testing.T) {
	ca := app.Asset{}
	xassert.Equal(t, app.FlagUndefined, ca.LocationCategory())
}

func TestAssetIsContainer(t *testing.T) {
	cases := []struct {
		name        string
		IsSingleton bool
		Type        *app.EveType
		want        bool
	}{
		{
			"ship",
			true,
			&app.EveType{Group: &app.EveGroup{Category: &app.EveCategory{ID: app.EveCategoryShip}}},
			true,
		},
		{
			"not singleton",
			false,
			&app.EveType{Group: &app.EveGroup{Category: &app.EveCategory{ID: app.EveCategoryShip}}},
			false,
		},
		{
			"no type",
			true,
			nil,
			false,
		},
		{
			"asset safety wrap",
			true,
			&app.EveType{ID: app.EveTypeAssetSafetyWrap, Group: &app.EveGroup{Category: &app.EveCategory{ID: app.EveCategoryDrone}}},
			true,
		},
		{
			"office",
			true,
			&app.EveType{ID: app.EveTypeOffice, Group: &app.EveGroup{Category: &app.EveCategory{ID: app.EveCategoryDrone}}},
			true,
		},
		{
			"cargo container",
			true,
			&app.EveType{Group: &app.EveGroup{ID: app.EveGroupCargoContainer, Category: &app.EveCategory{ID: app.EveCategoryDrone}}},
			true,
		},
		{
			"neither ship nor container",
			true,
			&app.EveType{Group: &app.EveGroup{Category: &app.EveCategory{ID: app.EveCategoryDrone}}},
			false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ca := app.Asset{IsSingleton: tc.IsSingleton, Type: tc.Type}
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
