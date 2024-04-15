package repository_test

import (
	"context"
	"example/evebuddy/internal/repository"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestToken(t *testing.T) {
	db, q, factory := setUpDB()
	defer db.Close()
	s := repository.New(db)
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		c := factory.CreateCharacter()
		o := repository.Token{
			AccessToken:  "access",
			CharacterID:  int32(c.ID),
			ExpiresAt:    time.Now(),
			RefreshToken: "refresh",
			TokenType:    "xxx",
		}
		// when
		err := s.UpdateOrCreateToken(ctx, &o)
		// then
		assert.NoError(t, err)
		r, err := q.GetToken(ctx, c.ID)
		if assert.NoError(t, err) {
			assert.Equal(t, o.AccessToken, r.AccessToken)
			assert.Equal(t, c.ID, r.CharacterID)
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		c := factory.CreateCharacter()
		o := repository.Token{
			AccessToken:  "access",
			CharacterID:  int32(c.ID),
			ExpiresAt:    time.Now(),
			RefreshToken: "refresh",
			TokenType:    "xxx",
		}
		if err := s.UpdateOrCreateToken(ctx, &o); err != nil {
			panic(err)
		}
		o.AccessToken = "changed"
		// when
		err := s.UpdateOrCreateToken(ctx, &o)
		// then
		assert.NoError(t, err)
		r, err := q.GetToken(ctx, c.ID)
		if assert.NoError(t, err) {
			assert.Equal(t, o.AccessToken, r.AccessToken)
			assert.Equal(t, c.ID, r.CharacterID)
		}
	})
}
