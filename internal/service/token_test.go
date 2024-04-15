package service_test

import (
	"context"
	"example/evebuddy/internal/repository"
	"example/evebuddy/internal/service"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestToken(t *testing.T) {
	db, q, factory := setUpDB()
	defer db.Close()
	s := service.NewService(q)
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		c := factory.CreateCharacter()
		o := service.Token{
			AccessToken:  "access",
			CharacterID:  int32(c.ID),
			ExpiresAt:    time.Now(),
			RefreshToken: "refresh",
			TokenType:    "xxx",
		}
		// when
		err := s.UpdateOrCreateToken(&o)
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
		o := service.Token{
			AccessToken:  "access",
			CharacterID:  int32(c.ID),
			ExpiresAt:    time.Now(),
			RefreshToken: "refresh",
			TokenType:    "xxx",
		}
		if err := s.UpdateOrCreateToken(&o); err != nil {
			panic(err)
		}
		o.AccessToken = "changed"
		// when
		err := s.UpdateOrCreateToken(&o)
		// then
		assert.NoError(t, err)
		r, err := q.GetToken(ctx, c.ID)
		if assert.NoError(t, err) {
			assert.Equal(t, o.AccessToken, r.AccessToken)
			assert.Equal(t, c.ID, r.CharacterID)
		}
	})
}
