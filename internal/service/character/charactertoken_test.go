package character

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestHasTokenWithScopes(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	s := New(r, nil, nil, nil, nil, nil)
	ctx := context.Background()
	t.Run("should create new queue", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(model.CharacterToken{CharacterID: c.ID, Scopes: esiScopes})
		// when
		x, err := s.CharacterHasTokenWithScopes(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.True(t, x)
		}

	})
}