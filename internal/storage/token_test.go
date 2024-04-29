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
		o := model.Token{
			AccessToken:  "access",
			CharacterID:  c.ID,
			ExpiresAt:    time.Now(),
			RefreshToken: "refresh",
			TokenType:    "xxx",
		}
		// when
		err := r.UpdateOrCreateToken(ctx, &o)
		// then
		assert.NoError(t, err)
		r, err := r.GetToken(ctx, c.ID)
		if assert.NoError(t, err) {
			assert.Equal(t, o.AccessToken, r.AccessToken)
			assert.Equal(t, c.ID, r.CharacterID)
		}
	})
	t.Run("can fetch existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateToken()
		// when
		r, err := r.GetToken(ctx, c.CharacterID)
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
		o := model.Token{
			AccessToken:  "access",
			CharacterID:  int32(c.ID),
			ExpiresAt:    time.Now(),
			RefreshToken: "refresh",
			TokenType:    "xxx",
		}
		if err := r.UpdateOrCreateToken(ctx, &o); err != nil {
			panic(err)
		}
		o.AccessToken = "changed"
		// when
		err := r.UpdateOrCreateToken(ctx, &o)
		// then
		assert.NoError(t, err)
		r, err := r.GetToken(ctx, c.ID)
		if assert.NoError(t, err) {
			assert.Equal(t, o.AccessToken, r.AccessToken)
			assert.Equal(t, c.ID, r.CharacterID)
		}
	})

	t.Run("should return correct error when not found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateMyCharacter()
		// when
		_, err := r.GetToken(ctx, c.ID)
		// then
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})
}
