package character

import (
	"context"
	"slices"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/stretchr/testify/assert"
)

func TestHasTokenWithScopes(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	s := newCharacterService(st)
	ctx := context.Background()
	t.Run("should return true when token has same scopes", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID, Scopes: esiScopes})
		// when
		x, err := s.CharacterHasTokenWithScopes(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.True(t, x)
		}
	})
	t.Run("should return false when token is missing scopes", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		esiScopes2 := []string{"esi-assets.read_assets.v1"}
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID, Scopes: esiScopes2})
		// when
		x, err := s.CharacterHasTokenWithScopes(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.False(t, x)
		}
	})
	t.Run("should return true when token has at least requested scopes", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID, Scopes: slices.Concat(esiScopes, []string{"extra"})})
		// when
		x, err := s.CharacterHasTokenWithScopes(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.True(t, x)
		}
	})
}
