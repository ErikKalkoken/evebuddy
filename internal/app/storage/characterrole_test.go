package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

func TestCharacterRole(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can update roles from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		r1 := set.Of(app.RoleAccountant, app.RoleAuditor)
		// when
		err := r.UpdateCharacterRoles(ctx, c.ID, r1)
		// then
		if assert.NoError(t, err) {
			r2, err := r.ListCharacterRoles(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, r1, r2)
			}
		}
	})
	t.Run("can add roles", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		if err := r.UpdateCharacterRoles(ctx, c.ID, set.Of(app.RoleBrandManager)); err != nil {
			panic(err)
		}
		want := set.Of(app.RoleDiplomat, app.RoleBrandManager)
		// when
		err := r.UpdateCharacterRoles(ctx, c.ID, want)
		// then
		if assert.NoError(t, err) {
			got, err := r.ListCharacterRoles(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
			}
		}
	})
	t.Run("can remove roles", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		if err := r.UpdateCharacterRoles(ctx, c.ID, set.Of(app.RoleDiplomat, app.RoleBrandManager)); err != nil {
			panic(err)
		}
		want := set.Of(app.RoleDiplomat)
		// when
		err := r.UpdateCharacterRoles(ctx, c.ID, want)
		// then
		if assert.NoError(t, err) {
			got, err := r.ListCharacterRoles(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
			}
		}
	})
}
