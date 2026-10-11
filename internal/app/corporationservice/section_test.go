package corporationservice_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestRemoveSectionDataWhenPermissionLost(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := testdouble.NewCorporationServiceFake(corporationservice.Params{Storage: st})
	ctx := context.Background()
	t.Run("should do nothing when permission exists", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		corporation := factory.CreateCorporation()
		const section = app.SectionCorporationIndustryJobs
		factory.CreateCorporationTokenForSection(corporation.ID, section)
		j1 := factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			CorporationID: corporation.ID,
		})
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: corporation.ID,
			Section:       section,
		})
		// when
		err := s.RemoveSectionDataWhenPermissionLost(ctx, corporation.ID)
		// then
		require.NoError(t, err)
		j2, err := st.GetCorporationIndustryJob(ctx, j1.CorporationID, j1.JobID)
		require.NoError(t, err)
		xassert.Equal(t, j1.StartDate, j2.StartDate)
		status, err := st.GetCorporationSectionStatus(
			ctx,
			corporation.ID,
			app.SectionCorporationIndustryJobs,
		)
		require.NoError(t, err)
		assert.True(t, status.HasContent())
	})
	t.Run("should delete secion when permission does not exit", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		corporation := factory.CreateCorporation()
		j1 := factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			CorporationID: corporation.ID,
		})
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: corporation.ID,
			Section:       app.SectionCorporationIndustryJobs,
		})
		// when
		err := s.RemoveSectionDataWhenPermissionLost(ctx, corporation.ID)
		// then
		require.NoError(t, err)
		_, err = st.GetCorporationIndustryJob(ctx, j1.CorporationID, j1.JobID)
		assert.ErrorIs(t, err, app.ErrNotFound)
		status, err := st.GetCorporationSectionStatus(
			ctx,
			corporation.ID,
			app.SectionCorporationIndustryJobs,
		)
		require.NoError(t, err)
		assert.False(t, status.HasContent())
	})
	t.Run("should delete structures when permission does not exist", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		corporation := factory.CreateCorporation()
		o := factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{
			CorporationID: corporation.ID,
		})
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: corporation.ID,
			Section:       app.SectionCorporationStructures,
		})
		// when
		err := s.RemoveSectionDataWhenPermissionLost(ctx, corporation.ID)
		// then
		require.NoError(t, err)
		_, err = st.GetCorporationStructure(ctx, o.CorporationID, o.StructureID)
		assert.ErrorIs(t, err, app.ErrNotFound)
		status, err := st.GetCorporationSectionStatus(
			ctx,
			corporation.ID,
			app.SectionCorporationStructures,
		)
		require.NoError(t, err)
		assert.False(t, status.HasContent())
	})
	t.Run("completes removal when context is canceled", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		corporation := factory.CreateCorporation()
		j1 := factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			CorporationID: corporation.ID,
		})
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: corporation.ID,
			Section:       app.SectionCorporationIndustryJobs,
		})
		ctx, cancel := context.WithCancel(ctx)
		cancel()
		// when
		err := s.RemoveSectionDataWhenPermissionLost(ctx, corporation.ID)
		// then
		require.NoError(t, err)
		_, err = st.GetCorporationIndustryJob(context.Background(), j1.CorporationID, j1.JobID)
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
}

func TestCorporationService_PermittedSections(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := testdouble.NewCorporationServiceFake(corporationservice.Params{Storage: st})
	ctx := context.Background()
	t.Run("should return section when matching token exists", func(t *testing.T) {
		// given
		section := app.SectionCorporationStructures
		testutil.MustTruncateTables(db)
		corporation := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(corporation.ID, section)
		// when
		got, err := s.PermittedSections(ctx, corporation.ID)
		// then
		require.NoError(t, err)
		want := set.Of(section)
		xassert.Equal(t, want, got)
	})
	t.Run("should return no sections when permission does not exist", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		corporation := factory.CreateCorporation()
		ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
			CorporationID: corporation.ID,
		})
		character := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec.ID})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{
			CharacterID: character.ID,
			Scopes:      app.SectionCharacterAssets.Scopes(),
		})
		// when
		got, err := s.PermittedSections(ctx, corporation.ID)
		// then
		require.NoError(t, err)
		want := set.Of[app.CorporationSection]()
		xassert.Equal(t, want, got)
	})
}

func TestCorporationService_Init(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	c := factory.CreateCorporation()
	o := factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
		CorporationID: c.ID,
		Section:       app.SectionCorporationIndustryJobs,
	})
	require.False(t, o.StartedAt.IsZero())
	scs := new(statuscache.StatusCache)
	require.NoError(t, scs.Init(ctx, st))
	x0, _ := scs.CorporationSection(c.ID, app.SectionCorporationIndustryJobs)
	require.True(t, x0.IsRunning())
	s := testdouble.NewCorporationServiceFake(corporationservice.Params{Storage: st, StatusCacheService: scs})
	// when
	err := s.Init(ctx)
	// then
	require.NoError(t, err)
	got, err := st.GetCorporationSectionStatus(ctx, c.ID, app.SectionCorporationIndustryJobs)
	require.NoError(t, err)
	assert.True(t, got.StartedAt.IsZero())
	x, ok := scs.CorporationSection(c.ID, app.SectionCorporationIndustryJobs)
	require.True(t, ok)
	assert.False(t, x.IsRunning())
}

func TestCorporationService_InitRemovesDataWithoutPermission(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	s := testdouble.NewCorporationServiceFake(corporationservice.Params{Storage: st})
	ctx := context.Background()
	t.Run("removes data no character has permission for", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		corporation := factory.CreateCorporation()
		j1 := factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			CorporationID: corporation.ID,
		})
		// when
		err := s.Init(ctx)
		// then
		require.NoError(t, err)
		_, err = st.GetCorporationIndustryJob(ctx, j1.CorporationID, j1.JobID)
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
	t.Run("keeps data with permission", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		corporation := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(corporation.ID, app.SectionCorporationIndustryJobs)
		j1 := factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			CorporationID: corporation.ID,
		})
		// when
		err := s.Init(ctx)
		// then
		require.NoError(t, err)
		_, err = st.GetCorporationIndustryJob(ctx, j1.CorporationID, j1.JobID)
		assert.NoError(t, err)
	})
}
