package xesi

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContext(t *testing.T) {
	t.Run("should set character ID and access token", func(t *testing.T) {
		ctx := NewContextWithAuth(context.Background(), 42, "token")
		characterID, ok := ctx.Value(contextCharacterID).(int32)
		assert.True(t, ok)
		assert.Equal(t, int32(42), characterID)
		ok2 := ContextHasAccessToken(ctx)
		assert.True(t, ok2)
	})
	t.Run("should report and return operation ID when set", func(t *testing.T) {
		ctx := NewContextWithOperationID(context.Background(), "op")
		id, ok := ctx.Value(contextOperationID).(string)
		assert.True(t, ok)
		assert.Equal(t, "op", id)
	})
}
