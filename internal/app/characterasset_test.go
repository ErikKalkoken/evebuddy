package app_test

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/stretchr/testify/assert"
)

func TestCharacterAsset(t *testing.T) {
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
		t.Run("should report wether asset is a container", func(t *testing.T) {
			c := &app.EveCategory{ID: tc.EveCategoryID}
			g := &app.EveGroup{Category: c}
			typ := &app.EveType{Group: g}
			ca := app.CharacterAsset{IsSingleton: tc.IsSingleton, EveType: typ}
			assert.Equal(t, tc.want, ca.IsContainer())
		})
	}
}
