package characterservice

import (
	"context"
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
		cs := NewFake(st, Params{AuthClient: AuthClientFake{Token: AuthTokenFromAppToken(token2)}})
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
		cs := NewFake(st, Params{AuthClient: AuthClientFake{Token: AuthTokenFromAppToken(token2)}})
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
