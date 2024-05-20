package service

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestHasTokenWithScopes(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	s := NewService(r)
	t.Run("should create new queue", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateMyCharacter()
		factory.CreateToken(model.CharacterToken{CharacterID: c.ID, Scopes: esiScopes})
		// when
		x, err := s.CharacterHasTokenWithScopes(c.ID)
		// then
		if assert.NoError(t, err) {
			assert.True(t, x)
		}

	})
}
