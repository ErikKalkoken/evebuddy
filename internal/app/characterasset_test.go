package app_test

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/stretchr/testify/assert"
)

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
