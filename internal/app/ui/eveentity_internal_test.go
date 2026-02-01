package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
)

func TestEntityIcon(t *testing.T) {
	alliance := fyne.NewStaticResource("alliance", []byte("alliance"))
	character := fyne.NewStaticResource("character", []byte("character"))
	corporation := fyne.NewStaticResource("corporation", []byte("corporation"))
	faction := fyne.NewStaticResource("faction", []byte("faction"))
	inventoryType := fyne.NewStaticResource("inventoryType", []byte("inventoryType"))
	fallback := icons.BlankSvg
	eis := &EveImageServiceFake{
		Alliance:    alliance,
		Character:   character,
		Corporation: corporation,
		Faction:     faction,
		Type:        inventoryType,
	}
	cases := []struct {
		category app.EveEntityCategory
		want     fyne.Resource
	}{
		{app.EveEntityAlliance, alliance},
		{app.EveEntityCharacter, character},
		{app.EveEntityCorporation, corporation},
		{app.EveEntityFaction, faction},
		{app.EveEntityInventoryType, inventoryType},
		{app.EveEntityStation, fallback},
	}
	for _, tc := range cases {
		t.Run(tc.category.String(), func(t *testing.T) {
			ee := &app.EveEntity{ID: 1, Category: tc.category, Name: "Dummy"}
			got, err := entityIcon(eis, ee, 64, fallback)
			if assert.NoError(t, err) {
				assert.Equal(t, tc.want, got)
			}
		})
	}
}
