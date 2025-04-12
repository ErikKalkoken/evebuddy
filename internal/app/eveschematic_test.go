package app_test

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/stretchr/testify/assert"
)

func TestEveSchematic(t *testing.T) {
	es := &app.EveSchematic{
		ID:   66,
		Name: "Cooliant",
	}
	r, ok := es.Icon()
	if assert.True(t, ok) {
		assert.NotNil(t, r)
	}
}
