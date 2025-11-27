package xesi_test

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/xesi"
	"github.com/stretchr/testify/assert"
)

func TestContextCharacterID(t *testing.T) {
	t.Run("should report and return character ID when set", func(t *testing.T) {
		ctx := xesi.NewContextWithCharacterID(context.Background(), 42)
		characterID, ok := xesi.CharacterIDFromContext(ctx)
		assert.True(t, ok)
		assert.Equal(t, int32(42), characterID)
	})
	t.Run("should report when character ID was not set", func(t *testing.T) {
		_, ok := xesi.CharacterIDFromContext(context.Background())
		assert.False(t, ok)
	})
}

func TestContextAccessToke(t *testing.T) {
	t.Run("should report when access token is set", func(t *testing.T) {
		ctx := xesi.NewContextWithAccessToken(context.Background(), "token")
		ok := xesi.ContextHasAccessToken(ctx)
		assert.True(t, ok)
	})
	t.Run("should report when access token was not set", func(t *testing.T) {
		ok := xesi.ContextHasAccessToken(context.Background())
		assert.False(t, ok)
	})
}
