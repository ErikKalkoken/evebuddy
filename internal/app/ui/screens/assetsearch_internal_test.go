package screens

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
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
