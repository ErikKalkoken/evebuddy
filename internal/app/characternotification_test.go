package app_test

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/stretchr/testify/assert"
)

func TestCharacterNotification(t *testing.T) {
	t.Run("can convert type to title", func(t *testing.T) {
		x := &app.CharacterNotification{
			Type: "AlphaBravoCharlie",
		}
		y := x.TitleFake()
		assert.Equal(t, "Alpha Bravo Charlie", y)
	})
	t.Run("can deal with short name", func(t *testing.T) {
		x := &app.CharacterNotification{
			Type: "Alpha",
		}
		y := x.TitleFake()
		assert.Equal(t, "Alpha", y)
	})
}

func TestCharacterNotificationBodyPlain(t *testing.T) {
	t.Run("can return body as plain text", func(t *testing.T) {
		n := &app.CharacterNotification{
			Type: "Alpha",
			Body: optional.New("**alpha**"),
		}
		got, err := n.BodyPlain()
		if assert.NoError(t, err) {
			assert.Equal(t, "alpha\n", got.MustValue())
		}
	})
	t.Run("should return empty when body is empty", func(t *testing.T) {
		n := &app.CharacterNotification{
			Type: "Alpha",
		}
		got, err := n.BodyPlain()
		if assert.NoError(t, err) {
			assert.True(t, got.IsEmpty())
		}
	})
}
