package model_test

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestCharacterAsset(t *testing.T) {
	cases := []struct {
		IsSingleton   bool
		EveCategoryID int32
		want          bool
	}{
		{true, model.EveCategoryShip, true},
		{false, model.EveCategoryShip, false},
		{true, model.EveCategoryDrone, false},
	}
	for _, tc := range cases {
		t.Run("should report wether asset is a container", func(t *testing.T) {
			c := &model.EveCategory{ID: tc.EveCategoryID}
			g := &model.EveGroup{Category: c}
			typ := &model.EveType{Group: g}
			ca := model.CharacterAsset{IsSingleton: tc.IsSingleton, EveType: typ}
			assert.Equal(t, tc.want, ca.IsContainer())
		})
	}
}
