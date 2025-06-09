package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

func TestToken(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		now := time.Now()
		o1 := app.CharacterToken{
			AccessToken:  "access",
			CharacterID:  c.ID,
			ExpiresAt:    now,
			RefreshToken: "refresh",
			Scopes:       []string{"alpha", "bravo"},
			TokenType:    "xxx",
		}
		// when
		err := st.UpdateOrCreateCharacterToken(ctx, &o1)
		// then
		assert.NoError(t, err)
		assert.Equal(t, "access", o1.AccessToken)
		assert.Equal(t, c.ID, o1.CharacterID)
		assert.Equal(t, now.UTC(), o1.ExpiresAt.UTC())
		assert.Equal(t, []string{"alpha", "bravo"}, o1.Scopes)
		assert.Equal(t, "xxx", o1.TokenType)
		o2, err := st.GetCharacterToken(ctx, c.ID)
		if assert.NoError(t, err) {
			assert.Equal(t, o1.AccessToken, o2.AccessToken)
			assert.Equal(t, c.ID, o2.CharacterID)
			assert.Equal(t, o1.ExpiresAt.UTC(), o2.ExpiresAt.UTC())
			assert.Equal(t, o1.Scopes, o2.Scopes)
			assert.Equal(t, o1.TokenType, o2.TokenType)
		}
	})
	t.Run("can fetch existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterToken()
		// when
		r, err := st.GetCharacterToken(ctx, c.CharacterID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, c.AccessToken, r.AccessToken)
			assert.Equal(t, c.CharacterID, r.CharacterID)
			assert.Equal(t, c.ExpiresAt.UTC(), c.ExpiresAt.UTC())
			assert.Equal(t, c.RefreshToken, r.RefreshToken)
			assert.Equal(t, c.TokenType, r.TokenType)
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		o1 := factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		o1.AccessToken = "changed"
		o1.Scopes = []string{"alpha", "bravo"}
		// when
		err := st.UpdateOrCreateCharacterToken(ctx, o1)
		// then
		assert.NoError(t, err)
		o2, err := st.GetCharacterToken(ctx, c.ID)
		if assert.NoError(t, err) {
			assert.Equal(t, o1.AccessToken, o2.AccessToken)
			assert.Equal(t, c.ID, o2.CharacterID)
			assert.Equal(t, o1.ExpiresAt.UTC(), o2.ExpiresAt.UTC())
			assert.Equal(t, o1.Scopes, o2.Scopes)
			assert.Equal(t, o1.TokenType, o2.TokenType)
		}
	})

	t.Run("should return correct error when not found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		// when
		_, err := st.GetCharacterToken(ctx, c.ID)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
	t.Run("list for corporation", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		corp1 := factory.CreateEveEntityCorporation()
		corp2 := factory.CreateEveEntityCorporation()

		// token with correct corp and role
		ec1 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: corp1.ID})
		c1 := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec1.ID})
		if err := st.UpdateCharacterRoles(ctx, c1.ID, set.Of(app.RoleFactoryManager)); err != nil {
			panic(err)
		}
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c1.ID})

		// token with correct corp and wrong role
		ec2 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: corp1.ID})
		c2 := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec2.ID})
		if err := st.UpdateCharacterRoles(ctx, c2.ID, set.Of(app.RoleAccountant)); err != nil {
			panic(err)
		}
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c2.ID})

		// token with wrong corp and correct role
		ec3 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: corp2.ID})
		c3 := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec3.ID})
		if err := st.UpdateCharacterRoles(ctx, c3.ID, set.Of(app.RoleAccountant)); err != nil {
			panic(err)
		}
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c3.ID})

		// when
		r, err := st.ListCharacterTokenForCorporation(ctx, c1.EveCharacter.Corporation.ID, app.RoleFactoryManager)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, r, 1)
			assert.Equal(t, c1.ID, r[0].CharacterID)
		}
	})
	t.Run("list for corporation returns not found error when no token", func(t *testing.T) {
		testutil.TruncateTables(db)
		corp := factory.CreateCorporation()
		_, err := st.ListCharacterTokenForCorporation(ctx, corp.ID, app.RoleFactoryManager)
		assert.Error(t, err, app.ErrNotFound)
	})
}
