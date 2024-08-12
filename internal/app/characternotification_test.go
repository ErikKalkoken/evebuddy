package app_test

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/stretchr/testify/assert"
)

func TestCharacterNotification(t *testing.T) {
	t.Run("can convert type to title", func(t *testing.T) {
		x := &app.CharacterNotification{
			Type: "AlphaBravoCharlie",
		}
		y := x.FakeTitle()
		assert.Equal(t, "Alpha Bravo Charlie", y)
	})
	t.Run("can deal with short name", func(t *testing.T) {
		x := &app.CharacterNotification{
			Type: "Alpha",
		}
		y := x.FakeTitle()
		assert.Equal(t, "Alpha", y)
	})
}
