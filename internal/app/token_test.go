package app_test

import (
	"testing"
	"time"

	"github.com/ErikKalkoken/kx/set"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

func TestCharacterToken_RemainsValid(t *testing.T) {
	t.Run("return true, when token remains valid within duration", func(t *testing.T) {
		x := app.CharacterToken{ExpiresAt: time.Now().Add(60 * time.Second)}
		assert.True(t, x.RemainsValid(55*time.Second))
	})
	t.Run("return false, when token expired within duration", func(t *testing.T) {
		x := app.CharacterToken{ExpiresAt: time.Now().Add(60 * time.Second)}
		assert.False(t, x.RemainsValid(65*time.Second))
	})
}

func TestCharacterToken_HasScopes(t *testing.T) {
	cases := []struct {
		name            string
		currentScopes   set.Set[string]
		requestedScopes set.Set[string]
		want            bool
	}{
		{"has all scopes", set.Of("alpha"), set.Of("alpha"), true},
		{"has all scopes 2", set.Of("alpha", "bravo", "charlie"), set.Of("alpha", "bravo"), true},
		{"missing scopes", set.Of("alpha", "bravo"), set.Of("bravo", "charlie"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o := app.CharacterToken{
				Scopes: tc.currentScopes,
			}
			got := o.HasScopes(tc.requestedScopes)
			assert.Equal(t, tc.want, got)
		})
	}
}
