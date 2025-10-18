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
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can update roles from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterFull()
		r1 := set.Of(app.RoleAccountant, app.RoleAuditor)
		// when
		err := st.UpdateCharacterRoles(ctx, c.ID, r1)
		// then
		if assert.NoError(t, err) {
			r2, err := st.ListCharacterRoles(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, r1, r2)
			}
		}
	})
	t.Run("can add roles", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterFull()
		if err := st.UpdateCharacterRoles(ctx, c.ID, set.Of(app.RoleBrandManager)); err != nil {
			panic(err)
		}
		want := set.Of(app.RoleDiplomat, app.RoleBrandManager)
		// when
		err := st.UpdateCharacterRoles(ctx, c.ID, want)
		// then
		if assert.NoError(t, err) {
			got, err := st.ListCharacterRoles(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
			}
		}
	})
	t.Run("can remove roles", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterFull()
		if err := st.UpdateCharacterRoles(ctx, c.ID, set.Of(app.RoleDiplomat, app.RoleBrandManager)); err != nil {
			panic(err)
		}
		want := set.Of(app.RoleDiplomat)
		// when
		err := st.UpdateCharacterRoles(ctx, c.ID, want)
		// then
		if assert.NoError(t, err) {
			got, err := st.ListCharacterRoles(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
			}
		}
	})
}
