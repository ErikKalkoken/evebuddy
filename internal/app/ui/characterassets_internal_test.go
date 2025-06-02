package ui

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/assetcollection"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/stretchr/testify/assert"
)

func TestCharacterAssetsMakeLocationTreeData(t *testing.T) {
	t.Run("can create simple tree", func(t *testing.T) {
		el := &app.EveLocation{
			ID:          100000,
			Name:        "Alpha 1",
			SolarSystem: &app.EveSolarSystem{Name: "Alpha", ID: 1},
		}
		a := &app.CharacterAsset{ItemID: 1, LocationID: el.ID}
		b := &app.CharacterAsset{ItemID: 2, LocationID: 1}
		c := &app.CharacterAsset{ItemID: 3, LocationID: 2}
		d := &app.CharacterAsset{ItemID: 4, LocationID: 2}
		assets := []*app.CharacterAsset{a, b, c, d}
		locations := []*app.EveLocation{el}
		ac := assetcollection.New(assets, locations)
		tree := makeLocationTreeData(ac.Locations(), 42)
		assert.Greater(t, tree.Size(), 0)
	})
	t.Run("can have multiple locations with items in space", func(t *testing.T) {
		l1 := &app.EveLocation{
			ID:          100000,
			Name:        "Alpha 1",
			SolarSystem: &app.EveSolarSystem{Name: "Alpha", ID: 1},
		}
		l2 := &app.EveLocation{
			ID:          100001,
			Name:        "Alpha 2",
			SolarSystem: &app.EveSolarSystem{Name: "Alpha", ID: 1},
		}
		a := &app.CharacterAsset{ItemID: 1, LocationID: l1.ID}
		b := &app.CharacterAsset{ItemID: 2, LocationID: 1}
		c := &app.CharacterAsset{ItemID: 3, LocationID: 1}
		d := &app.CharacterAsset{ItemID: 4, LocationID: 1}
		e := &app.CharacterAsset{ItemID: 5, LocationID: l2.ID}
		assets := []*app.CharacterAsset{a, b, c, d, e}
		locations := []*app.EveLocation{l1, l2}
		ac := assetcollection.New(assets, locations)
		tree := makeLocationTreeData(ac.Locations(), 42)
		assert.Greater(t, tree.Size(), 0)
	})
}

func TestSplitLines(t *testing.T) {
	const maxLine = 10
	cases := []struct {
		name  string
		in    string
		want1 string
		want2 string
	}{
		{"single line 1", "alpha", "alpha", ""},
		{"single line 2", "alpha boy", "alpha boy", ""},
		{"two lines single word", "verySophisticated", "verySophis", "ticated"},
		{"two lines long word", "verySophisticatedIndeed", "verySophis", "ticatedInd"},
		{"two lines", "first second", "first", "second"},
		{"two lines with truncation", "first second third", "first", "second thi"},
		{"one long word", "firstSecondThirdForth", "firstSecon", "dThirdFort"},
		{"special 1", "Erik Kalkoken's Cald", "Erik", "Kalkoken's"},
		// {"two lines two words", "Contaminated Nanite", "Contaminat", "ed Nanite"}, FIXME!
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got1, got2 := splitLines(tc.in, maxLine)
			assert.Equal(t, tc.want1, got1)
			assert.Equal(t, tc.want2, got2)
		})
	}
}

func TestCharacterAssets(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	t.Run("can update with data", func(t *testing.T) {
		bu := NewFakeBaseUI(st, test.NewTempApp(t))
		factory.CreateCharacterAsset()
		bu.characterAsset.update()
	})
	t.Run("can update without data", func(t *testing.T) {
		bu := NewFakeBaseUI(st, test.NewTempApp(t))
		bu.characterAsset.update()
	})
}
