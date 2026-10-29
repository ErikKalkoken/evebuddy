package corporationservice_test

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
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
		require.NoError(t, err)
		assert.True(t, changed)
		want := set.Of(corp.ID)
		got, err := s.ListCorporationIDs(ctx)
		require.NoError(t, err)
		xassert.Equal(t, want, got)
	})
	t.Run("removes deleted corporations from the status cache", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		character := factory.CreateCharacter()
		corp := factory.CreateCorporation(character.EveCharacter.Corporation.ID)
		obsolete := factory.CreateCorporation()
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: obsolete.ID,
			Section:       app.CorporationSections[0],
		})
		scs := new(statuscache.StatusCache)
		require.NoError(t, scs.Init(ctx, st))
		s := testdouble.NewCorporationServiceFake(corporationservice.Params{Storage: st, StatusCacheService: scs})
		// when
		_, err := s.UpdateCorporations(ctx)
		// then
		require.NoError(t, err)
		var got []int64
		for _, x := range scs.ListCorporations() {
			got = append(got, x.ID)
		}
		assert.Equal(t, []int64{corp.ID}, got)
		assert.False(t, scs.HasCorporationSection(obsolete.ID, app.CorporationSections[0]))
	})
	t.Run("report false when nothing deleted", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		character := factory.CreateCharacter()
		corp := factory.CreateCorporation(character.EveCharacter.Corporation.ID)
		changed, err := s.UpdateCorporations(ctx)
		require.NoError(t, err)

		assert.False(t, changed)
		want := set.Of(corp.ID)
		got, err := s.ListCorporationIDs(ctx)
		require.NoError(t, err)
		xassert.Equal(t, want, got)
	})
	t.Run("report false when no corporations", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		changed, err := s.UpdateCorporations(ctx)
		require.NoError(t, err)
		assert.False(t, changed)
		want := set.Of[int64]()
		got, err := s.ListCorporationIDs(ctx)
		require.NoError(t, err)
		xassert.Equal(t, want, got)
	})
	t.Run("can add missing corporations", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		character := factory.CreateCharacter()
		factory.CreateEveCorporation(storage.UpdateOrCreateEveCorporationParams{
			ID: character.EveCharacter.Corporation.ID,
		})
		changed, err := s.UpdateCorporations(ctx)
		require.NoError(t, err)
		assert.True(t, changed)
		got, err := s.ListCorporationIDs(ctx)
		require.NoError(t, err)
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
		require.NoError(t, err)
		assert.False(t, changed)
		got, err := s.ListCorporationIDs(ctx)
		require.NoError(t, err)
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
		require.NoError(t, err)
		xassert.Equal(t, c.ID, got.ID)
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
		require.NoError(t, err)
		xassert.Equal(t, c.ID, got.ID)
	})
	t.Run("should return error when no corporation found", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		_, err := s.GetAnyCorporation(ctx)
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
}

func TestCorporation_ListCorporations(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ctx := context.Background()
	s := testdouble.NewCorporationServiceFake(corporationservice.Params{Storage: st})
	t.Run("can list corporations", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		c1 := factory.CreateCorporation()
		c2 := factory.CreateCorporation()
		got, err := s.ListCorporations(ctx)
		require.NoError(t, err)
		ids := set.Collect(xiter.MapSlice(got, func(x *app.Corporation) int64 { return x.ID }))
		xassert.Equal(t, set.Of(c1.ID, c2.ID), ids)
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
		require.NoError(t, err)
		assert.True(t, got)
	})
	t.Run("reports false when corporation does not exist", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		got, err := s.HasCorporation(ctx, 42)
		require.NoError(t, err)
		assert.False(t, got)
	})
	t.Run("reports false when id is zero", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		got, err := s.HasCorporation(ctx, 0)
		require.NoError(t, err)
		assert.False(t, got)
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
		require.NoError(t, err)
		ids := set.Collect(xiter.MapSlice(got, func(x *app.EntityShort) int64 { return x.ID }))
		xassert.Equal(t, set.Of(c1.ID, c2.ID), ids)
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
		require.NoError(t, err)
		ec, err := st.GetEveCorporation(ctx, c.ID)
		require.NoError(t, err)
		assert.Equal(t, ec.Name, got[c.ID])
	})
}

func TestCorporation_ListCorporationsForSelection(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ctx := context.Background()
	s := testdouble.NewCorporationServiceFake(corporationservice.Params{Storage: st})

	t.Run("should list all corporations of characters", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c1.ID, app.CorporationSectionWalletJournal(app.Division1))
		c2 := factory.CreateCorporation()
		factory.CreateCorporationToken(c2.ID, set.Of[app.Role](), set.Of[string]())
		// when
		corporations, err := s.ListCorporationsForSelection(ctx, false)
		// then
		require.NoError(t, err)
		got := set.Collect(xiter.MapSlice(corporations, func(x *app.EntityShort) int64 { return x.ID }))
		xassert.Equal(t, set.Of(c1.ID, c2.ID), got)
	})

	t.Run("should list corporations where character has privileged access when limited ones are hidden", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		character := factory.CreateCharacter()
		corp1 := factory.CreateCorporation(character.EveCharacter.Corporation.ID)
		factory.SetCharacterRoles(character.ID, set.Of(app.RoleDirector))
		factory.CreateCorporation()
		// when
		corporations, err := s.ListCorporationsForSelection(ctx, true)
		// then
		require.NoError(t, err)
		got := set.Collect(xiter.MapSlice(corporations, func(x *app.EntityShort) int64 { return x.ID }))
		want := set.Of(corp1.ID)
		xassert.Equal(t, want, got)
	})

	t.Run("should not list corporations where character has no privileged access when limited ones are hidden", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		character := factory.CreateCharacter()
		factory.CreateCorporation(character.EveCharacter.Corporation.ID)
		factory.CreateCorporation()
		// when
		corporations, err := s.ListCorporationsForSelection(ctx, true)
		// then
		require.NoError(t, err)
		got := set.Collect(xiter.MapSlice(corporations, func(x *app.EntityShort) int64 { return x.ID }))
		xassert.Equal(t, 0, got.Size())
	})
}
