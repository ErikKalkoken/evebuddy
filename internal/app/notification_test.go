package app_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestType(t *testing.T) {
	t.Run("can convert to string", func(t *testing.T) {
		x := app.BountyClaimMsg
	xassert.Equal(t, "BountyClaimMsg", x.String())
	})
	t.Run("can convert to display string", func(t *testing.T) {
		x := app.SovereigntyTCUDamageMsg
	xassert.Equal(t, "Sovereignty TCU Damage Msg", x.Display())
	})
	t.Run("can return group", func(t *testing.T) {
	xassert.Equal(t, app.GroupStructure, app.StructureDestroyed.Group())
	})
}

func TestType_Category(t *testing.T) {
	t.Run("returns category when known", func(t *testing.T) {
	xassert.Equal(t, app.EveEntityCorporation, app.StructureDestroyed.Category())
	})
	t.Run("reports when category not known", func(t *testing.T) {
	xassert.Equal(t, app.EveEntityCharacter, app.AgentRetiredTrigravian.Category())
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
	xassert.Equal(t, "Structure Fuel Alert", y)
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
		xassert.Equal(t, "alpha\n", got.MustValue())
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
