package characterservice

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestCharacterService_EnsureValidToken(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("do nothing if token is still valid", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		character := factory.CreateCharacter()
		token1 := factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{
			AccessToken:  "access-old",
			CharacterID:  character.ID,
			RefreshToken: "refresh-old",
		})
		token2 := factory.CreateToken(app.Token{
			AccessToken:   "access-new",
			CharacterID:   character.ID,
			CharacterName: character.EveCharacter.Name,
			RefreshToken:  "refresh-new",
		})
		cs := NewFake(Params{
			Storage: st,
			AuthClient: testutil.AuthClientFake{
				Token: testutil.AuthTokenFromAppToken(token2),
			},
		})
		// when
		changed, err := cs.ensureValidToken(ctx, token1)
		// then
		require.NoError(t, err)
		assert.False(t, changed)
		xassert.Equal(t, "access-old", token1.AccessToken)
		xassert.Equal(t, "refresh-old", token1.RefreshToken)
	})
	t.Run("should refresh token when expired", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		character := factory.CreateCharacter()
		token := factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{
			AccessToken:  "access-old",
			CharacterID:  character.ID,
			ExpiresAt:    time.Now().UTC().Add(-10 * time.Second),
			RefreshToken: "refresh-old",
		})
		token2 := factory.CreateToken(app.Token{
			AccessToken:   "access-new",
			CharacterID:   character.ID,
			CharacterName: character.EveCharacter.Name,
			RefreshToken:  "refresh-new",
		})
		cs := NewFake(Params{Storage: st, AuthClient: testutil.AuthClientFake{
			Token: testutil.AuthTokenFromAppToken(token2),
		}})
		// when
		changed, err := cs.ensureValidToken(ctx, token)
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		xassert.Equal(t, "access-new", token.AccessToken)
		xassert.Equal(t, "refresh-new", token.RefreshToken)
		assert.True(t, token.ExpiresAt.After(time.Now()))
	})
}

func TestTokenSource_New(t *testing.T) {
	token := &app.CharacterToken{
		AccessToken:  "access",
		CharacterID:  42,
		ExpiresAt:    time.Now().Add(20 * time.Minute),
		RefreshToken: "refresh",
	}
	t.Run("should create tokens source", func(t *testing.T) {
		f := func(ctx context.Context, ct *app.CharacterToken) (bool, error) {
			return false, nil
		}
		ts := newTokenSource(token, f)
		xassert.Equal(t, token, ts.token)
	})
	t.Run("should panic when trying to create without token", func(t *testing.T) {
		assert.Panics(t, func() {
			newTokenSource(nil, func(ctx context.Context, ct *app.CharacterToken) (bool, error) {
				return false, nil
			})
		})
	})
	t.Run("should panic when trying to create without refresher func", func(t *testing.T) {
		assert.Panics(t, func() {
			newTokenSource(token, nil)
		})
	})

}

func TestTokenSource_Token(t *testing.T) {
	token := &app.CharacterToken{
		AccessToken:  "access",
		CharacterID:  42,
		ExpiresAt:    time.Now().Add(20 * time.Minute),
		RefreshToken: "refresh",
	}
	t.Run("should return token", func(t *testing.T) {
		ts := newTokenSource(token, func(ctx context.Context, ct *app.CharacterToken) (bool, error) {
			return false, nil
		})
		x, err := ts.Token()
		require.NoError(t, err)
		xassert.Equal(t, x.AccessToken, token.AccessToken)
		xassert.Equal(t, x.RefreshToken, token.RefreshToken)
		xassert.Equal(t, x.Expiry, token.ExpiresAt)
	})
	t.Run("should return error when no token undefined", func(t *testing.T) {
		ts := newTokenSource(token, func(ctx context.Context, ct *app.CharacterToken) (bool, error) {
			return false, nil
		})
		ts.token = nil
		_, err := ts.Token()
		require.Error(t, err)
	})
	t.Run("should return refreshed token when expired", func(t *testing.T) {
		token := &app.CharacterToken{
			AccessToken:  "access",
			CharacterID:  42,
			ExpiresAt:    time.Now().Add(-1 * time.Minute),
			RefreshToken: "refresh",
		}
		expiresAt2 := time.Now().Add(20 * time.Minute)
		ts := newTokenSource(token, func(ctx context.Context, ct *app.CharacterToken) (bool, error) {
			ct.AccessToken = "access2"
			ct.RefreshToken = "refresh2"
			ct.ExpiresAt = expiresAt2
			return true, nil
		})
		x, err := ts.Token()
		require.NoError(t, err)
		xassert.Equal(t, x.AccessToken, "access2")
		xassert.Equal(t, x.RefreshToken, "refresh2")
		xassert.Equal(t, x.Expiry, expiresAt2)
	})
	t.Run("should return error when refresh failedr", func(t *testing.T) {
		token := &app.CharacterToken{
			AccessToken:  "access",
			CharacterID:  42,
			ExpiresAt:    time.Now().Add(-1 * time.Minute),
			RefreshToken: "refresh",
		}
		ts := newTokenSource(token, func(ctx context.Context, ct *app.CharacterToken) (bool, error) {
			return false, fmt.Errorf("some error")
		})
		_, err := ts.Token()
		require.Error(t, err)
	})
}
