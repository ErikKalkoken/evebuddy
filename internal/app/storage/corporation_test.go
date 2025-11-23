package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

func TestCorporation(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		ec := factory.CreateEveCorporation()
		// when
		err := st.CreateCorporation(ctx, ec.ID)
		// then
		if assert.NoError(t, err) {
			r, err := st.GetCorporation(ctx, ec.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, ec.Name, r.EveCorporation.Name)
			}
		}
	})
	t.Run("raise specfic error when tyring to re-create existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		// when
		err := st.CreateCorporation(ctx, c.ID)
		// then
		assert.ErrorIs(t, err, app.ErrAlreadyExists)
	})
	t.Run("can fetch by ID with minimal fields populated only", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateCorporation()
		// when
		c2, err := st.GetEveCorporation(ctx, c1.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, c1.EveCorporation.Name, c2.Name)
		}
	})
	t.Run("can create when not exists", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		ec := factory.CreateEveCorporation()
		// when
		c, err := st.GetOrCreateCorporation(ctx, ec.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, ec.Name, c.EveCorporation.Name)
		}
	})
	t.Run("can get when exists", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateCorporation()
		// when
		c2, err := st.GetOrCreateCorporation(ctx, c1.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, c2, c1)
		}
	})
}

func TestListCorporations(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can list corporation IDs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateCorporation()
		c2 := factory.CreateCorporation()
		// when
		got, err := st.ListCorporationIDs(ctx)
		// then
		if assert.NoError(t, err) {
			want := set.Of(c1.ID, c2.ID)
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
	t.Run("can list corporations in short form", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateCorporation()
		c2 := factory.CreateCorporation()
		// when
		xx, err := st.ListCorporationsShort(ctx)
		// then
		if assert.NoError(t, err) {
			want := set.Of(c1.ID, c2.ID)
			got := set.Collect(xiter.MapSlice(xx, func(x *app.EntityShort[int32]) int32 {
				return x.ID
			}))
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
	t.Run("can list priviledged corporations", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		corp1 := factory.CreateCorporation(c.EveCharacter.Corporation.ID)
		factory.SetCharacterRoles(c.ID, set.Of(app.RoleBrandManager))
		factory.CreateCorporation()
		// when
		xx, err := st.ListPrivilegedCorporationsShort(ctx, set.Of(app.RoleBrandManager, app.RoleAccountant))
		// then
		if assert.NoError(t, err) {
			want := set.Of(corp1.ID)
			got := set.Collect(xiter.MapSlice(xx, func(x *app.EntityShort[int32]) int32 {
				return x.ID
			}))
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
}

func TestListOrphanedCorporationIDs(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("orphaned corporation exists", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		ec := factory.CreateEveCorporation()
		factory.CreateCorporation(ec.ID)
		factory.CreateEveEntityWithCategory(app.EveEntityCorporation, app.EveEntity{ID: ec.ID})
		x := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: ec.ID})
		factory.CreateCharacterFull(storage.CreateCharacterParams{ID: x.ID})
		corporation2 := factory.CreateCorporation()
		// when
		got, err := st.ListOrphanedCorporationIDs(ctx)
		// then
		if assert.NoError(t, err) {
			want := set.Of(corporation2.ID)
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
	t.Run("orphaned corporation does not exist", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		ec := factory.CreateEveCorporation()
		factory.CreateCorporation(ec.ID)
		factory.CreateEveEntityWithCategory(app.EveEntityCorporation, app.EveEntity{ID: ec.ID})
		x := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: ec.ID})
		factory.CreateCharacterFull(storage.CreateCharacterParams{ID: x.ID})
		// when
		got, err := st.ListOrphanedCorporationIDs(ctx)
		// then
		if assert.NoError(t, err) {
			want := set.Of[int32]()
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
}

func TestGetAnyCorporation(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("should return a character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateCorporation()
		c2 := factory.CreateCorporation()
		// when
		c, err := st.GetAnyCorporation(ctx)
		// then
		if assert.NoError(t, err) {
			assert.Contains(t, []int32{c1.ID, c2.ID}, c.ID)
		}
	})
	t.Run("should return correct error when not found", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		_, err := st.GetAnyCorporation(ctx)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
}
