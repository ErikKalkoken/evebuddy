package app_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

func TestNotificationTypesSupported(t *testing.T) {
	x := app.NotificationTypesSupported()
	assert.True(t, x.Contains(app.AllAnchoringMsg))
	assert.False(t, x.Contains(app.AgentRetiredTrigravian))
}

func TestEveNotificationTypes(t *testing.T) {
	got := app.EveNotificationTypes()
	assert.Contains(t, got, app.StructureDestroyed)
	assert.NotEmpty(t, got)
}

func TestEveNotificationGroupIsContainer(t *testing.T) {
	cases := []struct {
		g    app.EveNotificationGroup
		want bool
	}{
		{app.GroupUndefined, true},
		{app.GroupAll, true},
		{app.GroupUnread, true},
		{app.GroupStructure, false},
		{app.GroupWar, false},
	}
	for _, tc := range cases {
		t.Run(tc.g.String(), func(t *testing.T) {
			xassert.Equal(t, tc.want, tc.g.IsContainer())
		})
	}
}

func TestEveNotificationGroupString(t *testing.T) {
	xassert.Equal(t, "War", app.GroupWar.String())
}

func TestNotificationGroups(t *testing.T) {
	got := app.NotificationGroups()
	assert.Contains(t, got, app.GroupWar)
	assert.NotContains(t, got, app.GroupAll)
	assert.NotContains(t, got, app.GroupUnread)
	assert.NotContains(t, got, app.GroupUndefined)
}

func TestEveNotificationTypeFromString(t *testing.T) {
	t.Run("known type", func(t *testing.T) {
		got, ok := app.EveNotificationTypeFromString("StructureDestroyed")
		assert.True(t, ok)
		xassert.Equal(t, app.StructureDestroyed, got)
	})
	t.Run("unknown type", func(t *testing.T) {
		got, ok := app.EveNotificationTypeFromString("DoesNotExist")
		assert.False(t, ok)
		xassert.Equal(t, app.UnknownNotification, got)
	})
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

func TestCharacterNotification_TitleDisplay(t *testing.T) {
	t.Run("has title", func(t *testing.T) {
		x := &app.CharacterNotification{
			Type:  app.StructureFuelAlert,
			Title: optional.New("Custom Title"),
		}
		xassert.Equal(t, "Custom Title", x.TitleDisplay())
	})
	t.Run("falls back to fake title", func(t *testing.T) {
		x := &app.CharacterNotification{
			Type: app.StructureFuelAlert,
		}
		xassert.Equal(t, "Structure Fuel Alert", x.TitleDisplay())
	})
}

func TestCharacterNotification_BodyPlain(t *testing.T) {
	t.Run("can return body as plain text", func(t *testing.T) {
		n := &app.CharacterNotification{
			Type: app.StructureDestroyed,
			Body: optional.New("**alpha**"),
		}
		got, err := n.BodyPlain()
		require.NoError(t, err)
		xassert.EqualOptional(t, "alpha\n", got)
	})
	t.Run("should return empty when body is empty", func(t *testing.T) {
		n := &app.CharacterNotification{
			Type: app.StructureDestroyed,
		}
		got, err := n.BodyPlain()
		require.NoError(t, err)
		assert.True(t, got.IsEmpty())
	})
}

func TestCharacterNotification_ToJSON(t *testing.T) {
	t.Run("should return empty when receiver is nil", func(t *testing.T) {
		var n *app.CharacterNotification
		got, err := n.ToJSON()
		require.NoError(t, err)
		assert.Nil(t, got)
	})
	t.Run("can return notification as JSON", func(t *testing.T) {
		timestamp := time.Date(2026, 8, 1, 12, 30, 0, 0, time.UTC)
		n := &app.CharacterNotification{
			Type:           app.CorpAppNewMsg,
			NotificationID: 42,
			Sender: &app.EveEntity{
				Category: app.EveEntityCorporation,
				ID:       1011,
			},
			Text:      optional.New("applicationText: example\ncharID: 1011\ncorpID: 2001\n"),
			Timestamp: timestamp,
		}
		got, err := n.ToJSON()
		require.NoError(t, err)
		want := `{
	"notification_id": 42,
	"sender_id": 1011,
	"sender_type": "corporation",
	"text": {
		"applicationText": "example",
		"charID": 1011,
		"corpID": 2001
	},
	"timestamp": "2026-08-01T12:30:00Z",
	"type": "CorpAppNewMsg"
}`
		got2 := string(got)
		assert.JSONEq(t, want, got2)
	})
	t.Run("should return error for invalid text", func(t *testing.T) {
		n := &app.CharacterNotification{
			Type:   app.CorpAppNewMsg,
			Sender: &app.EveEntity{Category: app.EveEntityCorporation},
			Text:   optional.New("key: [invalid"),
		}
		_, err := n.ToJSON()
		assert.Error(t, err)
	})
}
