package character

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/sqlite/testutil"
	"github.com/stretchr/testify/assert"
)

func TestHasTokenWithScopes(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	s := newCharacterService(st)
	ctx := context.Background()
	t.Run("should create new queue", func(t *testing.T) {
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
}
