package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestToken(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
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
				xassert.Equal(t, arg.AccessToken, x.AccessToken)
				xassert.Equal(t, c.ID, x.CharacterID)
				xassert.Equal(t, arg.ExpiresAt.UTC(), x.ExpiresAt.UTC())
				assert.True(t, x.Scopes.Equal(arg.Scopes), "got %q, wanted %q", x.Scopes, arg.Scopes)
				xassert.Equal(t, arg.TokenType, x.TokenType)
			}
		}
	})
	t.Run("can fetch existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterToken()
		// when
		r, err := st.GetCharacterToken(ctx, c.CharacterID)
		// then
		if assert.NoError(t, err) {
			xassert.Equal(t, c.AccessToken, r.AccessToken)
			xassert.Equal(t, c.CharacterID, r.CharacterID)
			xassert.Equal(t, c.ExpiresAt.UTC(), c.ExpiresAt.UTC())
			xassert.Equal(t, c.RefreshToken, r.RefreshToken)
			xassert.Equal(t, c.TokenType, r.TokenType)
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
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
			xassert.Equal(t, "changed", o2.AccessToken)
			xassert.Equal(t, c.ID, o2.CharacterID)
			xassert.Equal(t, o1.ExpiresAt.UTC(), o2.ExpiresAt.UTC())
			xassert.Equal(t, set.Of("alpha", "bravo"), o2.Scopes)
			xassert.Equal(t, o1.TokenType, o2.TokenType)
		}
	})

	t.Run("should return correct error when not found", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
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
		testutil.MustTruncateTables(db)
		corp1 := factory.CreateEveEntityCorporation()
		corp2 := factory.CreateEveEntityCorporation()

		// token with correct corp, role and scope
		ec1 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: corp1.ID})
		c1 := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec1.ID})
		err := st.UpdateCharacterRoles(ctx, c1.ID, set.Of(app.RoleFactoryManager))
		require.NoError(t, err)
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{
			CharacterID: c1.ID,
			Scopes:      set.Of("alpha", "bravo"),
		})

		// token with correct corp and wrong role
		ec2 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: corp1.ID})
		c2 := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec2.ID})
		err = st.UpdateCharacterRoles(ctx, c2.ID, set.Of(app.RoleAccountant))
		require.NoError(t, err)
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c2.ID})

		// token with wrong corp and correct role
		ec3 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: corp2.ID})
		c3 := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec3.ID})
		err = st.UpdateCharacterRoles(ctx, c3.ID, set.Of(app.RoleAccountant))
		require.NoError(t, err)
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c3.ID})

		// token with correct corp and role, but wrong scope
		ec4 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: corp1.ID})
		c4 := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec4.ID})
		err = st.UpdateCharacterRoles(ctx, c1.ID, set.Of(app.RoleFactoryManager))
		require.NoError(t, err)
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{
			CharacterID: c4.ID,
			Scopes:      set.Of("bravo"),
		})

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
			xassert.Equal(t, c1.ID, r[0].CharacterID)
		}
	})
	t.Run("matches any when no roles", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateEveEntityCorporation()

		// token with no role
		ec1 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: c.ID})
		c1 := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec1.ID})
		err := st.UpdateCharacterRoles(ctx, c1.ID, set.Of[app.Role]())
		require.NoError(t, err)
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c1.ID, Scopes: set.Of("alpha", "bravo")})

		// token with other role
		ec2 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: c.ID})
		c2 := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec2.ID})
		err = st.UpdateCharacterRoles(ctx, c2.ID, set.Of(app.RoleAuditor))
		require.NoError(t, err)
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
	t.Run("returns empty when no tokens found", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		corp := factory.CreateCorporation()
		got, err := st.ListCharacterTokenForCorporation(ctx, corp.ID, set.Of(app.RoleFactoryManager), set.Of("alpha"))
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}
