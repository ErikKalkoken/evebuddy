package model_test

import (
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestTokenRemainsValid(t *testing.T) {
	t.Run("return true, when token remains valid within duration", func(t *testing.T) {
		x := model.CharacterToken{ExpiresAt: time.Now().Add(60 * time.Second)}
		assert.True(t, x.RemainsValid(55*time.Second))
	})
	t.Run("return false, when token expired within duration", func(t *testing.T) {
		x := model.CharacterToken{ExpiresAt: time.Now().Add(60 * time.Second)}
		assert.False(t, x.RemainsValid(65*time.Second))
	})
}
