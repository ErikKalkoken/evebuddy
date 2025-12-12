package characterservice_test

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/stretchr/testify/assert"
)

func TestHasTokenWithScopes(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := characterservice.NewFake(st)
	ctx := context.Background()
	t.Run("should return true when token has same scopes", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{
			CharacterID: c.ID,
			Scopes:      set.Of("alpha", "bravo"),
		})
		// when
		x, err := s.HasTokenWithScopes(ctx, c.ID, set.Of("alpha", "bravo"))
		// then
		if assert.NoError(t, err) {
			assert.True(t, x)
		}
	})
	t.Run("should return false when token is missing scopes", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{
			CharacterID: c.ID,
			Scopes:      set.Of("alpha"),
		})
		// when
		x, err := s.HasTokenWithScopes(ctx, c.ID, set.Of("alpha", "bravo"))
		// then
		if assert.NoError(t, err) {
			assert.False(t, x)
		}
	})
	t.Run("should return true when token has at least requested scopes", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{
			CharacterID: c.ID,
			Scopes:      set.Of("alpha", "bravo", "charlie"),
		})
		// when
		x, err := s.HasTokenWithScopes(ctx, c.ID, set.Of("alpha", "bravo"))
		// then
		if assert.NoError(t, err) {
			assert.True(t, x)
		}
	})
}

func TestMissingScopes(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := characterservice.NewFake(st)
	ctx := context.Background()
	t.Run("should return empty when token has all scopes", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{
			CharacterID: c.ID,
			Scopes:      set.Of("alpha", "bravo"),
		})
		// when
		got, err := s.MissingScopes(ctx, c.ID, set.Of("alpha", "bravo"))
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, 0, got.Size())
		}
	})
	t.Run("should return scopes that are missing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{
			CharacterID: c.ID,
			Scopes:      set.Of("alpha"),
		})
		// when
		got, err := s.MissingScopes(ctx, c.ID, set.Of("alpha", "bravo"))
		// then
		if assert.NoError(t, err) {
			want := set.Of("bravo")
			xassert.EqualSet(t, want, got)
		}
	})
	t.Run("when no token found all scopes are missing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		// when
		got, err := s.MissingScopes(ctx, c.ID, set.Of("alpha", "bravo"))
		// then
		if assert.NoError(t, err) {
			want := set.Of("alpha", "bravo")
			xassert.EqualSet(t, want, got)
		}
	})
}

func TestCharacterTokenForCorporation(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	s := characterservice.NewFake(st)
	ctx := context.Background()
	t.Run("should return matching token when exists", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateCharacterToken()
		c := factory.CreateCharacter()
		o1 := factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{
			CharacterID: c.ID,
			Scopes:      app.Scopes(),
		})
		if err := st.UpdateCharacterRoles(ctx, c.ID, set.Of(app.RoleAccountant)); err != nil {
			t.Fatal(err)
		}
		// when
		o2, err := s.CharacterTokenForCorporation(ctx, c.EveCharacter.Corporation.ID, set.Of(app.RoleAccountant), set.Set[string]{}, false)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, o1.ID, o2.ID)
		}
	})
	t.Run("should report not found when token exists and role not matching", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{
			CharacterID: c.ID,
			Scopes:      app.Scopes(),
		})
		if err := st.UpdateCharacterRoles(ctx, c.ID, set.Of(app.RoleBrandManager)); err != nil {
			t.Fatal(err)
		}
		// when
		_, err := s.CharacterTokenForCorporation(ctx, c.EveCharacter.Corporation.ID, set.Of(app.RoleAccountant), set.Set[string]{}, false)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
}
