package app_test

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/stretchr/testify/assert"
)

func TestType(t *testing.T) {
	t.Run("can convert to string", func(t *testing.T) {
		x := app.BountyClaimMsg
		assert.Equal(t, "BountyClaimMsg", x.String())
	})
	t.Run("can convert to display string", func(t *testing.T) {
		x := app.SovereigntyTCUDamageMsg
		assert.Equal(t, "Sovereignty TCU Damage Msg", x.Display())
	})
	t.Run("can return group", func(t *testing.T) {
		assert.Equal(t, app.GroupStructure, app.StructureDestroyed.Group())
	})
}

func TestType_Category(t *testing.T) {
	t.Run("returns category when known", func(t *testing.T) {
		c, ok := app.StructureDestroyed.Category()
		if assert.True(t, ok) {
			assert.Equal(t, app.EveEntityCorporation, c)
		}
	})
	t.Run("reports when category not known", func(t *testing.T) {
		_, ok := app.AgentRetiredTrigravian.Category()
		assert.False(t, ok)
	})
}

func TestGroupTypes(t *testing.T) {
	x := app.NotificationGroupTypes(app.GroupStructure)
	assert.True(t, x.Contains(app.StructureDestroyed))
}

func TestCharacterNotification(t *testing.T) {
	t.Run("can convert type to fake title", func(t *testing.T) {
		x := &app.CharacterNotification{
			Type: app.StructureFuelAlert,
		}
		y := x.TitleFake()
		assert.Equal(t, "Structure Fuel Alert", y)
	})
}

func TestCharacterNotificationBodyPlain(t *testing.T) {
	t.Run("can return body as plain text", func(t *testing.T) {
		n := &app.CharacterNotification{
			Type: app.StructureDestroyed,
			Body: optional.New("**alpha**"),
		}
		got, err := n.BodyPlain()
		if assert.NoError(t, err) {
			assert.Equal(t, "alpha\n", got.MustValue())
		}
	})
	t.Run("should return empty when body is empty", func(t *testing.T) {
		n := &app.CharacterNotification{
			Type: app.StructureDestroyed,
		}
		got, err := n.BodyPlain()
		if assert.NoError(t, err) {
			assert.True(t, got.IsEmpty())
		}
	})
}
