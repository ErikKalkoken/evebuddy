package app_test

import (
	"testing"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
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

func TestCharacterToken_IsValid(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		x := app.CharacterToken{ExpiresAt: time.Now().Add(time.Hour)}
		assert.True(t, x.IsValid())
	})
	t.Run("expired", func(t *testing.T) {
		x := app.CharacterToken{ExpiresAt: time.Now().Add(-time.Hour)}
		assert.False(t, x.IsValid())
	})
}

func TestCharacterToken_AuthToken(t *testing.T) {
	expiresAt := time.Now().Add(time.Hour)
	x := app.CharacterToken{
		AccessToken:  "access",
		CharacterID:  42,
		ExpiresAt:    expiresAt,
		RefreshToken: "refresh",
		Scopes:       set.Of("alpha", "bravo"),
		TokenType:    "Bearer",
	}
	got := x.AuthToken()
	xassert.Equal(t, "access", got.AccessToken)
	xassert.Equal(t, int32(42), got.CharacterID)
	xassert.Equal(t, expiresAt, got.ExpiresAt)
	xassert.Equal(t, "refresh", got.RefreshToken)
	xassert.Equal(t, "Bearer", got.TokenType)
	assert.ElementsMatch(t, []string{"alpha", "bravo"}, got.Scopes)
}

func TestCharacterToken_OauthToken(t *testing.T) {
	expiresAt := time.Now().Add(time.Hour)
	x := app.CharacterToken{
		AccessToken:  "access",
		ExpiresAt:    expiresAt,
		RefreshToken: "refresh",
	}
	got := x.OauthToken()
	xassert.Equal(t, "access", got.AccessToken)
	xassert.Equal(t, "refresh", got.RefreshToken)
	xassert.Equal(t, expiresAt, got.Expiry)
	assert.InDelta(t, 3600, got.ExpiresIn, 2)
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
			xassert.Equal(t, tc.want, got)
		})
	}
}
