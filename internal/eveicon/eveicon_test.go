package eveicon_test

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
	"github.com/stretchr/testify/assert"
)

func TestGetResourceByIconID(t *testing.T) {
	t.Run("should return resource for valid ID", func(t *testing.T) {
		r, ok := eveicon.GetResourceByIconID(26)
		assert.True(t, ok)
		assert.Equal(t, "6_64_5.png", r.Name())
	})
	t.Run("should return undefined resource for invalid ID", func(t *testing.T) {
		r, ok := eveicon.GetResourceByIconID(4711)
		assert.False(t, ok)
		assert.Equal(t, "7_64_15.png", r.Name())
	})
}

func TestGetResourceByName(t *testing.T) {
	t.Run("should return a named resource", func(t *testing.T) {
		r := eveicon.GetResourceByName(eveicon.Faction)
		assert.Equal(t, "73_16_246.png", r.Name())
	})
}
