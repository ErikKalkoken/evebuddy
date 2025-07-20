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
		c := factory.CreateCharacterFull()
		now := time.Now()
		arg := storage.UpdateOrCreateCharacterTokenParams{
			AccessToken:  "access",
			CharacterID:  c.ID,
			ExpiresAt:    now,
			RefreshToken: "refresh",
			Scopes:       set.Of("alpha", "bravo"),
			TokenType:    "xxx",
		}
		// when
		err := st.UpdateOrCreateCharacterToken(ctx, arg)
		// then
		if assert.NoError(t, err) {
			x, err := st.GetCharacterToken(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, arg.AccessToken, x.AccessToken)
				assert.Equal(t, c.ID, x.CharacterID)
				assert.Equal(t, arg.ExpiresAt.UTC(), x.ExpiresAt.UTC())
				assert.True(t, x.Scopes.Equal(arg.Scopes), "got %q, wanted %q", x.Scopes, arg.Scopes)
				assert.Equal(t, arg.TokenType, x.TokenType)
			}
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
		c := factory.CreateCharacterFull()
		o1 := factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		arg := storage.UpdateOrCreateCharacterTokenParamsFromToken(o1)
		arg.AccessToken = "changed"
		arg.Scopes = set.Of("alpha", "bravo")
		// when
		err := st.UpdateOrCreateCharacterToken(ctx, arg)
		// then
		assert.NoError(t, err)
		o2, err := st.GetCharacterToken(ctx, c.ID)
		if assert.NoError(t, err) {
			assert.Equal(t, "changed", o2.AccessToken)
			assert.Equal(t, c.ID, o2.CharacterID)
			assert.Equal(t, o1.ExpiresAt.UTC(), o2.ExpiresAt.UTC())
			assert.ElementsMatch(t, []string{"alpha", "bravo"}, o2.Scopes.Slice())
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
}

func TestListCharacterTokenForCorporation(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("return matching token only", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		corp1 := factory.CreateEveEntityCorporation()
		corp2 := factory.CreateEveEntityCorporation()

		// token with correct corp, role and scope
		ec1 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: corp1.ID})
		c1 := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec1.ID})
		if err := st.UpdateCharacterRoles(ctx, c1.ID, set.Of(app.RoleFactoryManager)); err != nil {
			panic(err)
		}
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c1.ID, Scopes: set.Of("alpha", "bravo")})

		// token with correct corp and wrong role
		ec2 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: corp1.ID})
		c2 := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec2.ID})
		if err := st.UpdateCharacterRoles(ctx, c2.ID, set.Of(app.RoleAccountant)); err != nil {
			panic(err)
		}
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c2.ID})

		// token with wrong corp and correct role
		ec3 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: corp2.ID})
		c3 := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec3.ID})
		if err := st.UpdateCharacterRoles(ctx, c3.ID, set.Of(app.RoleAccountant)); err != nil {
			panic(err)
		}
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c3.ID})

		// token with correct corp and role, but wrong scope
		ec4 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: corp1.ID})
		c4 := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec4.ID})
		if err := st.UpdateCharacterRoles(ctx, c1.ID, set.Of(app.RoleFactoryManager)); err != nil {
			panic(err)
		}
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c4.ID, Scopes: set.Of("bravo")})

		// when
		r, err := st.ListCharacterTokenForCorporation(
			ctx,
			c1.EveCharacter.Corporation.ID,
			set.Of(app.RoleFactoryManager, app.RoleAccountant),
			set.Of("alpha", "bravo"),
		)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, r, 1)
			assert.Equal(t, c1.ID, r[0].CharacterID)
		}
	})
	t.Run("matches any when no roles", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateEveEntityCorporation()

		// token with no role
		ec1 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: c.ID})
		c1 := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec1.ID})
		if err := st.UpdateCharacterRoles(ctx, c1.ID, set.Of[app.Role]()); err != nil {
			panic(err)
		}
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c1.ID, Scopes: set.Of("alpha", "bravo")})

		// token with other role
		ec2 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: c.ID})
		c2 := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec2.ID})
		if err := st.UpdateCharacterRoles(ctx, c2.ID, set.Of(app.RoleAuditor)); err != nil {
			panic(err)
		}
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c2.ID, Scopes: set.Of("alpha", "bravo")})

		// when
		r, err := st.ListCharacterTokenForCorporation(
			ctx,
			c1.EveCharacter.Corporation.ID,
			set.Of[app.Role](),
			set.Of("alpha", "bravo"),
		)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, r, 2)
		}
	})
	t.Run("list for corporation returns not found error when no token", func(t *testing.T) {
		testutil.TruncateTables(db)
		corp := factory.CreateCorporation()
		_, err := st.ListCharacterTokenForCorporation(ctx, corp.ID, set.Of(app.RoleFactoryManager), set.Of("alpha"))
		assert.Error(t, err, app.ErrNotFound)
	})
}
