package characterservice_test

import (
	"context"
	"slices"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/stretchr/testify/assert"
)

func TestHasTokenWithScopes(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := characterservice.NewFake(st)
	ctx := context.Background()
	t.Run("should return true when token has same scopes", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID, Scopes: app.Scopes().Slice()})
		// when
		x, err := s.HasTokenWithScopes(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.True(t, x)
		}
	})
	t.Run("should return false when token is missing scopes", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterFull()
		esiScopes2 := []string{"esi-assets.read_assets.v1"}
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID, Scopes: esiScopes2})
		// when
		x, err := s.HasTokenWithScopes(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.False(t, x)
		}
	})
	t.Run("should return true when token has at least requested scopes", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID, Scopes: slices.Concat(app.Scopes().Slice(), []string{"extra"})})
		// when
		x, err := s.HasTokenWithScopes(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.True(t, x)
		}
	})
}

func TestValidCharacterTokenForCorporation(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	s := characterservice.NewFake(st)
	ctx := context.Background()
	t.Run("should return matching token when exists", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		factory.CreateCharacterToken()
		c := factory.CreateCharacterMinimal()
		o1 := factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{
			CharacterID: c.ID,
			Scopes:      app.Scopes().Slice(),
		})
		if err := st.UpdateCharacterRoles(ctx, c.ID, set.Of(app.RoleAccountant)); err != nil {
			t.Fatal(err)
		}
		// when
		o2, err := s.ValidCharacterTokenForCorporation(ctx, c.EveCharacter.Corporation.ID, app.RoleAccountant)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, o1.ID, o2.ID)
		}
	})
	t.Run("should report not found when token exists and role not matching", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterMinimal()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{
			CharacterID: c.ID,
			Scopes:      app.Scopes().Slice(),
		})
		if err := st.UpdateCharacterRoles(ctx, c.ID, set.Of(app.RoleBrandManager)); err != nil {
			t.Fatal(err)
		}
		// when
		_, err := s.ValidCharacterTokenForCorporation(ctx, c.EveCharacter.Corporation.ID, app.RoleAccountant)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
}
