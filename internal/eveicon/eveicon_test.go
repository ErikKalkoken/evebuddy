package eveicon_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
)

func TestGetResourceByIconID(t *testing.T) {
	t.Run("should return resource for valid ID", func(t *testing.T) {
		r, ok := eveicon.FromID(26)
		assert.True(t, ok)
		assert.Equal(t, "resources/6_64_5.png", r.Name())
	})
	t.Run("should return undefined resource for invalid ID", func(t *testing.T) {
		r, ok := eveicon.FromID(4711)
		assert.False(t, ok)
		assert.Equal(t, "resources/7_64_15.png", r.Name())
	})
}

func TestGetResourceByName(t *testing.T) {
	t.Run("should return a named resource", func(t *testing.T) {
		r := eveicon.FromName(eveicon.Faction)
		assert.Equal(t, "resources/73_16_246.png", r.Name())
	})
}

func TestFromSchematicID(t *testing.T) {
	t.Run("should return resource for valid ID", func(t *testing.T) {
		r, ok := eveicon.FromSchematicID(66)
		assert.True(t, ok)
		assert.Equal(t, "resources/24_64_6.png", r.Name())
	})
	t.Run("should return undefined resource for invalid ID", func(t *testing.T) {
		r, ok := eveicon.FromSchematicID(1)
		assert.False(t, ok)
		assert.Equal(t, "resources/7_64_15.png", r.Name())
	})
}
