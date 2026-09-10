package corporationservice_test

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

func TestCorporation_UpdateCorporations(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ctx := context.Background()
	s := testdouble.NewCorporationServiceFake(corporationservice.Params{Storage: st})
	t.Run("can delete corporations with no member character", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		character := factory.CreateCharacter()
		corp := factory.CreateCorporation(character.EveCharacter.Corporation.ID)
		factory.CreateCorporation()
		changed, err := s.UpdateCorporations(ctx)
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		assert.True(t, changed)
		want := set.Of(corp.ID)
		got, err := s.ListCorporationIDs(ctx)
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		xassert.Equal(t, want, got)
	})
	t.Run("report false when nothing deleted", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		character := factory.CreateCharacter()
		corp := factory.CreateCorporation(character.EveCharacter.Corporation.ID)
		changed, err := s.UpdateCorporations(ctx)
		if !assert.NoError(t, err) {
			t.Fatal()
		}

		assert.False(t, changed)
		want := set.Of(corp.ID)
		got, err := s.ListCorporationIDs(ctx)
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		xassert.Equal(t, want, got)
	})
	t.Run("report false when no corporations", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		changed, err := s.UpdateCorporations(ctx)
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		assert.False(t, changed)
		want := set.Of[int64]()
		got, err := s.ListCorporationIDs(ctx)
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		xassert.Equal(t, want, got)
	})
	t.Run("can add missing corporations", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		character := factory.CreateCharacter()
		factory.CreateEveCorporation(storage.UpdateOrCreateEveCorporationParams{
			ID: character.EveCharacter.Corporation.ID,
		})
		changed, err := s.UpdateCorporations(ctx)
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		assert.True(t, changed)
		got, err := s.ListCorporationIDs(ctx)
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		want := set.Of(character.EveCharacter.Corporation.ID)
		xassert.Equal(t, want, got)
	})
	t.Run("should not add missing NPC corp", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		cc := factory.CreateEveEntityCorporation(app.EveEntity{
			ID: 1000115,
		})
		ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
			CorporationID: cc.ID,
		})
		character := factory.CreateCharacter(storage.CreateCharacterParams{
			ID: ec.ID,
		})
		factory.CreateEveCorporation(storage.UpdateOrCreateEveCorporationParams{
			ID: character.EveCharacter.Corporation.ID,
		})
		changed, err := s.UpdateCorporations(ctx)
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		assert.False(t, changed)
		got, err := s.ListCorporationIDs(ctx)
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		xassert.Equal(t, 0, got.Size())
	})
}

func TestCorporation_GetCorporation(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ctx := context.Background()
	s := testdouble.NewCorporationServiceFake(corporationservice.Params{Storage: st})
	t.Run("can return existing corporation", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		got, err := s.GetCorporation(ctx, c.ID)
		if assert.NoError(t, err) {
			xassert.Equal(t, c.ID, got.ID)
		}
	})
	t.Run("should return error when corporation not found", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		_, err := s.GetCorporation(ctx, 42)
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
}

func TestCorporation_GetAnyCorporation(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ctx := context.Background()
	s := testdouble.NewCorporationServiceFake(corporationservice.Params{Storage: st})
	t.Run("can return a corporation", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		got, err := s.GetAnyCorporation(ctx)
		if assert.NoError(t, err) {
			xassert.Equal(t, c.ID, got.ID)
		}
	})
	t.Run("should return error when no corporation found", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		_, err := s.GetAnyCorporation(ctx)
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
}

func TestCorporation_HasCorporation(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ctx := context.Background()
	s := testdouble.NewCorporationServiceFake(corporationservice.Params{Storage: st})
	t.Run("reports true when corporation exists", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		got, err := s.HasCorporation(ctx, c.ID)
		if assert.NoError(t, err) {
			assert.True(t, got)
		}
	})
	t.Run("reports false when corporation does not exist", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		got, err := s.HasCorporation(ctx, 42)
		if assert.NoError(t, err) {
			assert.False(t, got)
		}
	})
	t.Run("reports false when id is zero", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		got, err := s.HasCorporation(ctx, 0)
		if assert.NoError(t, err) {
			assert.False(t, got)
		}
	})
}

func TestCorporation_ListCorporationsShort(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ctx := context.Background()
	s := testdouble.NewCorporationServiceFake(corporationservice.Params{Storage: st})
	t.Run("can list corporations in short form", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		c1 := factory.CreateCorporation()
		c2 := factory.CreateCorporation()
		got, err := s.ListCorporationsShort(ctx)
		if assert.NoError(t, err) {
			ids := set.Collect(xiter.MapSlice(got, func(x *app.EntityShort) int64 { return x.ID }))
			xassert.Equal(t, set.Of(c1.ID, c2.ID), ids)
		}
	})
}

func TestCorporation_CorporationNames(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ctx := context.Background()
	s := testdouble.NewCorporationServiceFake(corporationservice.Params{Storage: st})
	t.Run("returns a map of corporation names", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		got, err := s.CorporationNames(ctx)
		if assert.NoError(t, err) {
			ec, err := st.GetEveCorporation(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, ec.Name, got[c.ID])
			}
		}
	})
}

func TestCorporation_ListPrivilegedCorporations(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ctx := context.Background()
	s := testdouble.NewCorporationServiceFake(corporationservice.Params{Storage: st})
	t.Run("can list corporations with privileged access", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		character := factory.CreateCharacter()
		corp1 := factory.CreateCorporation(character.EveCharacter.Corporation.ID)
		factory.SetCharacterRoles(character.ID, app.CorporationSectionWalletJournal(app.Division1).Roles())
		factory.CreateCorporation()
		got, err := s.ListPrivilegedCorporations(ctx)
		if assert.NoError(t, err) {
			ids := set.Collect(xiter.MapSlice(got, func(x *app.EntityShort) int64 { return x.ID }))
			xassert.Equal(t, set.Of(corp1.ID), ids)
		}
	})
}
