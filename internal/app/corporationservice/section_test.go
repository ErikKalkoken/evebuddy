package corporationservice_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
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
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
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
	t.Run("keeps data when role exists but scope is missing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		corporation := factory.CreateCorporation()
		const section = app.SectionCorporationIndustryJobs
		factory.CreateCorporationToken(corporation.ID, section.Roles(), set.Of("other-scope"))
		j1 := factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			CorporationID: corporation.ID,
		})
		// when
		err := s.RemoveSectionDataWhenPermissionLost(ctx, corporation.ID)
		// then
		require.NoError(t, err)
		_, err = st.GetCorporationIndustryJob(ctx, j1.CorporationID, j1.JobID)
		assert.NoError(t, err)
	})
	t.Run("returns not found when corporation does not exist", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		err := s.RemoveSectionDataWhenPermissionLost(ctx, 42)
		// then
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
	// On disk, because Init starts a background job, and a second connection to an in-memory DB sees an empty DB.
	db, st, factory := testutil.NewDBOnDisk(t)
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
	defer s.Stop() // stops the background job started by Init
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
	// On disk, because Stop cancels queries, which wipes an in-memory DB.
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ctx := context.Background()
	jobExists := func(j *app.CorporationIndustryJob) bool {
		_, err := st.GetCorporationIndustryJob(ctx, j.CorporationID, j.JobID)
		return !errors.Is(err, app.ErrNotFound)
	}
	t.Run("removes data no character has permission for", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		corporation := factory.CreateCorporation()
		j1 := factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			CorporationID: corporation.ID,
		})
		s := testdouble.NewCorporationServiceFake(corporationservice.Params{Storage: st})
		defer s.Stop()
		// when
		err := s.Init(ctx)
		// then
		require.NoError(t, err)
		assert.Eventually(t, func() bool {
			return !jobExists(j1)
		}, 2*time.Second, 10*time.Millisecond)
	})
	t.Run("keeps data with permission", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		corporation := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(corporation.ID, app.SectionCorporationIndustryJobs)
		j1 := factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			CorporationID: corporation.ID,
		})
		s := testdouble.NewCorporationServiceFake(corporationservice.Params{Storage: st})
		defer s.Stop()
		// when
		err := s.Init(ctx)
		// then
		require.NoError(t, err)
		assert.Never(t, func() bool {
			return !jobExists(j1)
		}, 200*time.Millisecond, 10*time.Millisecond)
	})
}

func TestCorporationService_StartAfterInit(t *testing.T) {
	if xgoesi.IsDailyDowntime() {
		t.Skip("the scheduler skips updates during daily downtime")
	}
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	signals := app.NewSignals()
	s := testdouble.NewCorporationServiceFake(corporationservice.Params{Storage: st, Signals: signals})
	started := make(chan struct{}, 1)
	signals.UpdateStarted.AddListener(func(_ context.Context, _ string) {
		select {
		case started <- struct{}{}:
		default:
		}
	})
	// when
	require.NoError(t, s.Init(t.Context()))
	s.Start(time.Hour)
	defer s.Stop()
	// then
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("update scheduler did not start after Init")
	}
}
