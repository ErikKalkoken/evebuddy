package screens

import (
	"bytes"
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestCompileRowsForClipboard(t *testing.T) {
	tests := []struct {
		name string
		rows []assetRow
		want string
	}{
		{
			name: "no rows",
			rows: nil,
			want: "",
		},
		{
			name: "single row",
			rows: []assetRow{
				{typeID: 1, typeName: "Tritanium", quantity: 100},
			},
			want: "Tritanium 100\n",
		},
		{
			name: "aggregates quantities for same type",
			rows: []assetRow{
				{typeID: 1, typeName: "Tritanium", quantity: 100},
				{typeID: 1, typeName: "Tritanium", quantity: 50},
			},
			want: "Tritanium 150\n",
		},
		{
			name: "sorts multiple types by name",
			rows: []assetRow{
				{typeID: 1, typeName: "Tritanium", quantity: 100},
				{typeID: 2, typeName: "Pyerite", quantity: 20},
			},
			want: "Pyerite 20\nTritanium 100\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := consolidateAssetRows(tt.rows)
			xassert.Equal(t, tt.want, got)
		})
	}
}

func TestMakeCSVFromRows(t *testing.T) {
	rows := []assetRow{
		{
			itemID:          1000000000001,
			typeID:          34,
			typeName:        "Tritanium",
			name:            "Tritanium Stack",
			groupID:         18,
			groupName:       "Mineral",
			categoryID:      4,
			categoryName:    "Material",
			locationName:    "Jita IV - Moon 4",
			locationFlag:    app.FlagHangar,
			state:           "Personal",
			quantity:        1000,
			isSingleton:     true,
			variant:         app.VariantBPO,
			solarSystemID:   30000142,
			solarSystemName: "Jita",
			regionID:        10000002,
			regionName:      "The Forge",
			price:           optional.New(5.5),
			total:           optional.New(5500.0),
			owner:           &app.EveEntity{ID: 1001, Name: "Bruce Wayne"},
			tagsDisplay:     "PVE",
		},
		{
			itemID:       1000000000002,
			typeID:       35,
			typeName:     "Pyerite",
			groupID:      18,
			groupName:    "Mineral",
			locationName: "Jita IV - Moon 4",
			state:        "Personal",
			quantity:     50,
			owner:        &app.EveEntity{ID: 1001, Name: "Bruce Wayne"},
		},
	}
	t.Run("for character includes owner and tags", func(t *testing.T) {
		var b bytes.Buffer
		err := writeAssetRowsToCSV(&b, rows, false)
		require.NoError(t, err)
		got := b.String()
		want := "Item ID,Type ID,Type Name,Item Name,Group ID,Group Name,Category ID,Category Name,Location Name,Location Flag,State,Quantity,Is Singleton,Variant,Solar System ID,Solar System Name,Region ID,Region Name,Price,Total,Owner ID,Owner Name,Tags\n" +
			"1000000000001,34,Tritanium,Tritanium Stack,18,Mineral,4,Material,Jita IV - Moon 4,FlagHangar,Personal,1000,true,BPO,30000142,Jita,10000002,The Forge,5.5,5500,1001,Bruce Wayne,PVE\n" +
			"1000000000002,35,Pyerite,,18,Mineral,0,,Jita IV - Moon 4,FlagUndefined,Personal,50,false,,0,,0,,,,1001,Bruce Wayne,\n"
		xassert.Equal(t, want, got)
	})
	t.Run("for corporation includes owner but omits tags", func(t *testing.T) {
		var b bytes.Buffer
		err := writeAssetRowsToCSV(&b, rows, true)
		require.NoError(t, err)
		got := b.String()
		want := "Item ID,Type ID,Type Name,Item Name,Group ID,Group Name,Category ID,Category Name,Location Name,Location Flag,State,Quantity,Is Singleton,Variant,Solar System ID,Solar System Name,Region ID,Region Name,Price,Total,Owner ID,Owner Name\n" +
			"1000000000001,34,Tritanium,Tritanium Stack,18,Mineral,4,Material,Jita IV - Moon 4,FlagHangar,Personal,1000,true,BPO,30000142,Jita,10000002,The Forge,5.5,5500,1001,Bruce Wayne\n" +
			"1000000000002,35,Pyerite,,18,Mineral,0,,Jita IV - Moon 4,FlagUndefined,Personal,50,false,,0,,0,,,,1001,Bruce Wayne\n"
		xassert.Equal(t, want, got)
	})
	t.Run("no rows returns only header", func(t *testing.T) {
		var b bytes.Buffer
		err := writeAssetRowsToCSV(&b, nil, false)
		require.NoError(t, err)
		got := b.String()
		want := "Item ID,Type ID,Type Name,Item Name,Group ID,Group Name,Category ID,Category Name,Location Name,Location Flag,State,Quantity,Is Singleton,Variant,Solar System ID,Solar System Name,Region ID,Region Name,Price,Total,Owner ID,Owner Name,Tags\n"
		xassert.Equal(t, want, got)
	})
}

func TestAssetSearchFilter_Match(t *testing.T) {
	r := assetRow{
		categoryName: "Ship",
		groupName:    "Frigate",
		locationName: "Jita IV - Moon 4",
		owner:        &app.EveEntity{Name: "Bruce"},
		regionName:   "The Forge",
		state:        "In space",
		tags:         set.Of("alpha"),
		total:        optional.New(1.5),
	}
	noTotal := r
	noTotal.total = optional.Optional[float64]{}
	for _, tc := range []struct {
		name   string
		filter assetSearchFilter
		row    assetRow
		want   bool
	}{
		{"no filter", assetSearchFilter{}, r, true},
		{"category matches", assetSearchFilter{category: "Ship"}, r, true},
		{"category differs", assetSearchFilter{category: "Module"}, r, false},
		{"group differs", assetSearchFilter{group: "Cruiser"}, r, false},
		{"location differs", assetSearchFilter{location: "Amarr"}, r, false},
		{"owner differs", assetSearchFilter{owner: "Alice"}, r, false},
		{"region differs", assetSearchFilter{region: "Domain"}, r, false},
		{"state differs", assetSearchFilter{state: "In hangar"}, r, false},
		{"tag matches", assetSearchFilter{tag: "alpha"}, r, true},
		{"tag missing", assetSearchFilter{tag: "bravo"}, r, false},
		{"has total", assetSearchFilter{total: assetSearchTotalYes}, r, true},
		{"has total but none", assetSearchFilter{total: assetSearchTotalYes}, noTotal, false},
		{"has no total", assetSearchFilter{total: assetSearchTotalNo}, noTotal, true},
		{"has no total but has", assetSearchFilter{total: assetSearchTotalNo}, r, false},
		{"all match", assetSearchFilter{category: "Ship", group: "Frigate", region: "The Forge", tag: "alpha", total: assetSearchTotalYes}, r, true},
		{"one of many differs", assetSearchFilter{category: "Ship", group: "Cruiser"}, r, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.filter.match(tc.row))
		})
	}
}

func TestAssetSearch_Filter(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	rows := []assetRow{
		{itemID: 1, categoryName: "Ship", owner: &app.EveEntity{Name: "Bruce"}},
		{itemID: 2, categoryName: "Module", owner: &app.EveEntity{Name: "Bruce"}},
	}
	newAssetSearchWith := func(t *testing.T, isMobile, forCorporation bool) *AssetSearch {
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			IsMobile: isMobile,
			Storage:  st,
		})
		var a *AssetSearch
		if forCorporation {
			a = NewAssetSearchForCorporation(u)
		} else {
			a = NewAssetSearchForAll(u)
		}
		a.rows = rows
		a.filterRowsAsync("")
		require.Len(t, a.rowsFiltered, 2)
		return a
	}
	newAssetSearch := func(t *testing.T, isMobile bool) *AssetSearch {
		return newAssetSearchWith(t, isMobile, false)
	}
	t.Run("can filter on mobile", func(t *testing.T) {
		a := newAssetSearch(t, true)
		a.filterChip.SetSelected(map[string]string{assetSearchFilterCategory: "Ship"})
		if assert.Len(t, a.rowsFiltered, 1) {
			assert.EqualValues(t, 1, a.rowsFiltered[0].itemID)
		}
	})
	t.Run("can filter on desktop", func(t *testing.T) {
		a := newAssetSearch(t, false)
		a.selectCategory.SetSelected("Ship")
		if assert.Len(t, a.rowsFiltered, 1) {
			assert.EqualValues(t, 1, a.rowsFiltered[0].itemID)
		}
	})
	t.Run("shows all filters on mobile", func(t *testing.T) {
		a := newAssetSearch(t, true)
		assert.Equal(t, map[string]string{
			assetSearchFilterCategory: "",
			assetSearchFilterGroup:    "",
			assetSearchFilterLocation: "",
			assetSearchFilterOwner:    "",
			assetSearchFilterRegion:   "",
			assetSearchFilterState:    "",
			assetSearchFilterTag:      "",
			assetSearchFilterTotal:    "",
		}, a.filterChip.Selected())
	})
	t.Run("hides tag and owner filters for corporation on mobile", func(t *testing.T) {
		a := newAssetSearchWith(t, true, true)
		assert.Equal(t, map[string]string{
			assetSearchFilterCategory: "",
			assetSearchFilterGroup:    "",
			assetSearchFilterLocation: "",
			assetSearchFilterRegion:   "",
			assetSearchFilterState:    "",
			assetSearchFilterTotal:    "",
		}, a.filterChip.Selected())
	})
	t.Run("resets filters when corporation changes on mobile", func(t *testing.T) {
		a := newAssetSearchWith(t, true, true)
		a.filterChip.SetSelected(map[string]string{assetSearchFilterCategory: "Ship"})
		require.True(t, a.filterChip.IsOn())

		a.u.Signals().CurrentCorporationExchanged.Emit(t.Context(), factory.CreateCorporation())

		assert.False(t, a.filterChip.IsOn())
	})
	t.Run("resets filters when corporation changes on desktop", func(t *testing.T) {
		a := newAssetSearchWith(t, false, true)
		a.selectCategory.SetSelected("Ship")

		a.u.Signals().CurrentCorporationExchanged.Emit(t.Context(), factory.CreateCorporation())

		assert.Empty(t, a.currentFilter())
	})
}
