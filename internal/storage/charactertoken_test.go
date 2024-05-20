package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func TestToken(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateMyCharacter()
		now := time.Now()
		o1 := model.CharacterToken{
			AccessToken:  "access",
			CharacterID:  c.ID,
			ExpiresAt:    now,
			RefreshToken: "refresh",
			Scopes:       []string{"alpha", "bravo"},
			TokenType:    "xxx",
		}
		// when
		err := r.UpdateOrCreateCharacterToken(ctx, &o1)
		// then
		assert.NoError(t, err)
		assert.Equal(t, "access", o1.AccessToken)
		assert.Equal(t, c.ID, o1.CharacterID)
		assert.Equal(t, now.Unix(), o1.ExpiresAt.Unix())
		assert.Equal(t, []string{"alpha", "bravo"}, o1.Scopes)
		assert.Equal(t, "xxx", o1.TokenType)
		o2, err := r.GetCharacterToken(ctx, c.ID)
		if assert.NoError(t, err) {
			assert.Equal(t, o1.AccessToken, o2.AccessToken)
			assert.Equal(t, c.ID, o2.CharacterID)
			assert.Equal(t, o1.ExpiresAt.Unix(), o2.ExpiresAt.Unix())
			assert.Equal(t, o1.Scopes, o2.Scopes)
			assert.Equal(t, o1.TokenType, o2.TokenType)
		}
	})
	t.Run("can fetch existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateToken()
		// when
		r, err := r.GetCharacterToken(ctx, c.CharacterID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, c.AccessToken, r.AccessToken)
			assert.Equal(t, c.CharacterID, r.CharacterID)
			assert.Equal(t, c.ExpiresAt.Unix(), c.ExpiresAt.Unix())
			assert.Equal(t, c.RefreshToken, r.RefreshToken)
			assert.Equal(t, c.TokenType, r.TokenType)
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateMyCharacter()
		o1 := factory.CreateToken(model.CharacterToken{CharacterID: c.ID})
		o1.AccessToken = "changed"
		o1.Scopes = []string{"alpha", "bravo"}
		// when
		err := r.UpdateOrCreateCharacterToken(ctx, o1)
		// then
		assert.NoError(t, err)
		o2, err := r.GetCharacterToken(ctx, c.ID)
		if assert.NoError(t, err) {
			assert.Equal(t, o1.AccessToken, o2.AccessToken)
			assert.Equal(t, c.ID, o2.CharacterID)
			assert.Equal(t, o1.ExpiresAt.Unix(), o2.ExpiresAt.Unix())
			assert.Equal(t, o1.Scopes, o2.Scopes)
			assert.Equal(t, o1.TokenType, o2.TokenType)
		}
	})

	t.Run("should return correct error when not found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateMyCharacter()
		// when
		_, err := r.GetCharacterToken(ctx, c.ID)
		// then
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})
}
