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
			xassert.Equal(t, tc.want, app.AssetCanHaveName(tc.IsSingleton, tc.Type))
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

func TestAssetVolume(t *testing.T) {
	both := &app.EveType{Volume: optional.New(2500.0), PackagedVolume: optional.New(50.0)}
	volumeOnly := &app.EveType{Volume: optional.New(2500.0)}
	t.Run("uses packaged volume when not a singleton", func(t *testing.T) {
		ca := app.Asset{Type: both}
		xassert.EqualOptional(t, 50.0, ca.Volume())
	})
	t.Run("uses volume when a singleton", func(t *testing.T) {
		ca := app.Asset{Type: both, IsSingleton: true}
		xassert.EqualOptional(t, 2500.0, ca.Volume())
	})
	t.Run("falls back to volume when packaged volume is missing", func(t *testing.T) {
		ca := app.Asset{Type: volumeOnly}
		xassert.EqualOptional(t, 2500.0, ca.Volume())
	})
	t.Run("is empty when type has no volumes", func(t *testing.T) {
		ca := app.Asset{Type: &app.EveType{}}
		xassert.Empty(t, ca.Volume())
	})
	t.Run("is empty when singleton has only a packaged volume", func(t *testing.T) {
		ca := app.Asset{Type: &app.EveType{PackagedVolume: optional.New(50.0)}, IsSingleton: true}
		xassert.Empty(t, ca.Volume())
	})
	t.Run("is empty without a type", func(t *testing.T) {
		ca := app.Asset{}
		xassert.Empty(t, ca.Volume())
	})
}

func TestAssetTotalPrice(t *testing.T) {
	t.Run("multiplies price with quantity", func(t *testing.T) {
		ca := app.Asset{Price: optional.New(2.5), Quantity: 4}
		xassert.EqualOptional(t, 10.0, ca.TotalPrice())
	})
	t.Run("is empty without a price", func(t *testing.T) {
		ca := app.Asset{Quantity: 4}
		xassert.Empty(t, ca.TotalPrice())
	})
}

func TestAssetTotalVolume(t *testing.T) {
	t.Run("multiplies packaged volume with quantity", func(t *testing.T) {
		ca := app.Asset{
			Quantity: 4,
			Type:     &app.EveType{Volume: optional.New(2500.0), PackagedVolume: optional.New(50.0)},
		}
		xassert.EqualOptional(t, 200.0, ca.TotalVolume())
	})
	t.Run("uses volume for a singleton", func(t *testing.T) {
		ca := app.Asset{
			IsSingleton: true,
			Quantity:    1,
			Type:        &app.EveType{Volume: optional.New(2500.0), PackagedVolume: optional.New(50.0)},
		}
		xassert.EqualOptional(t, 2500.0, ca.TotalVolume())
	})
	t.Run("removes floating point noise", func(t *testing.T) {
		ca := app.Asset{Quantity: 3, Type: &app.EveType{Volume: optional.New(0.1)}}
		xassert.EqualOptional(t, 0.3, ca.TotalVolume())
	})
	t.Run("keeps precision of small volumes", func(t *testing.T) {
		ca := app.Asset{Quantity: 1, Type: &app.EveType{Volume: optional.New(0.0025)}}
		xassert.EqualOptional(t, 0.0025, ca.TotalVolume())
	})
	t.Run("is empty without a volume", func(t *testing.T) {
		ca := app.Asset{Quantity: 4, Type: &app.EveType{}}
		xassert.Empty(t, ca.TotalVolume())
	})
	t.Run("is empty without a type", func(t *testing.T) {
		ca := app.Asset{Quantity: 4}
		xassert.Empty(t, ca.TotalVolume())
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
