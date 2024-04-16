package repository_test

import (
	"context"
	"example/evebuddy/internal/repository"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMailCreate(t *testing.T) {
	db, r, factory := setUpDB()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		c := factory.CreateCharacter()
		f := factory.CreateEveEntity()
		recipient := factory.CreateEveEntity()
		// l := factory.CreateMailLabel(repository.MailLabel{CharacterID: c.ID})
		// when
		arg := repository.CreateMailParams{
			Body:         "body",
			CharacterID:  c.ID,
			FromID:       f.ID,
			IsRead:       false,
			MailID:       42,
			RecipientIDs: []int32{recipient.ID},
			Subject:      "subject",
			Timestamp:    time.Now(),
		}
		_, err := r.CreateMail(ctx, arg)
		// then
		if assert.NoError(t, err) {
			m, err := r.GetMail(ctx, c.ID, 42)
			assert.NoError(t, err)
			assert.Equal(t, int32(42), m.MailID)
			assert.Equal(t, "body", m.Body)
			assert.Equal(t, f, m.From)
			assert.Equal(t, c.ID, m.CharacterID)
			assert.Equal(t, "subject", m.Subject)
			assert.False(t, m.Timestamp.IsZero())
			assert.Equal(t, []repository.EveEntity{recipient}, m.Recipients)
			// assert.Equal(t, l.Name, m.Labels[0].Name)
			// assert.Equal(t, l.LabelID, m.Labels[0].LabelID)
		}
	})
}

func TestMail(t *testing.T) {
	db, r, factory := setUpDB()
	defer db.Close()
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
		err := r.UpdateOrCreateToken(ctx, &o)
		// then
		assert.NoError(t, err)
		r, err := r.GetToken(ctx, c.ID)
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
		repository.TruncateTables(db)
		c := factory.CreateCharacter()
		// when
		_, err := r.GetMail(ctx, c.ID, 99)
		// then
		assert.ErrorIs(t, err, repository.ErrNotFound)
	})
}
