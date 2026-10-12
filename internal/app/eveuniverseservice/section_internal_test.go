package eveuniverseservice

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	goesi "github.com/fnt-eve/goesi-openapi"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
)

func newTestService(st *storage.Storage, scs StatusCache) *EVEUniverseService {
	return New(Params{
		ESIClient: goesi.NewESIClientWithOptions(http.DefaultClient, goesi.ClientOptions{
			UserAgent: "MyApp/1.0 (contact@example.com)",
		}),
		Signals:            app.NewSignals(),
		StatusCacheService: scs,
		Storage:            st,
	})
}

func TestRecordUpdateFailed(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	scs := new(statuscache.StatusCache)
	s := newTestService(st, scs)
	const section = app.SectionEveMarketPrices
	recordStarted := func(t *testing.T) {
		t.Helper()
		_, err := st.UpdateOrCreateGeneralSectionStatus(context.Background(), storage.UpdateOrCreateGeneralSectionStatusParams{
			Section:   section,
			StartedAt: new(optional.New(time.Now())),
		})
		require.NoError(t, err)
	}
	arg := eveUniverseSectionUpdateParams{section: section}
	t.Run("should clear started but keep status when canceled", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		completedAt := time.Now().Add(-6 * time.Hour)
		factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
			Section:      section,
			CompletedAt:  completedAt,
			ErrorMessage: "old error",
		})
		recordStarted(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		// when
		s.recordUpdateFailed(ctx, arg, fmt.Errorf("%w: %w", app.ErrCanceled, context.Canceled))
		// then
		x, err := st.GetGeneralSectionStatus(context.Background(), section)
		require.NoError(t, err)
		assert.True(t, x.StartedAt.IsZero())
		xassert.Equal(t, "old error", x.ErrorMessage)
		assert.WithinDuration(t, completedAt, x.CompletedAt, time.Second)
		y, ok := scs.EveUniverseSection(section)
		require.True(t, ok)
		assert.False(t, y.IsRunning())
	})
	t.Run("should record error when failed", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		recordStarted(t)
		// when
		s.recordUpdateFailed(context.Background(), arg, fmt.Errorf("dummy"))
		// then
		x, err := st.GetGeneralSectionStatus(context.Background(), section)
		require.NoError(t, err)
		assert.True(t, x.StartedAt.IsZero())
		xassert.Equal(t, "dummy", x.ErrorMessage)
	})
}

func TestUpdateSectionIfNeeded_ReturnsOriginalErrorWhenErrorPersistFails(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := newTestService(st, new(statuscache.StatusCache))
	httpmock.RegisterResponder(
		"GET",
		"https://esi.evetech.net/markets/prices",
		func(req *http.Request) (*http.Response, error) {
			// close the DB so recording the failure also fails
			db.Close()
			return nil, fmt.Errorf("esi failed")
		},
	)
	// when
	_, err := s.updateSectionIfNeeded(context.Background(), eveUniverseSectionUpdateParams{
		section:     app.SectionEveMarketPrices,
		forceUpdate: true,
	})
	// then
	require.Error(t, err)
	assert.ErrorContains(t, err, "esi failed")
}

func TestUpdateSectionAndRefreshIfNeeded_DoesNotReportSuccessOnFailedUpdate(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := newTestService(st, new(statuscache.StatusCache))
	ctx := context.Background()
	httpmock.Reset()
	httpmock.RegisterResponder(
		"GET",
		"https://esi.evetech.net/markets/prices",
		httpmock.NewErrorResponder(fmt.Errorf("failed")),
	)
	var updated bool
	s.signals.EveUniverseSectionUpdated.AddListener(func(ctx context.Context, arg app.EveUniverseSectionUpdated) {
		updated = true
	})
	// when
	s.updateSectionAndRefreshIfNeeded(ctx, app.SectionEveMarketPrices, false)
	// then
	status, err := st.GetGeneralSectionStatus(ctx, app.SectionEveMarketPrices)
	require.NoError(t, err)
	assert.True(t, status.HasError(), "expected section status to record the ESI failure")
	assert.False(t, updated, "EveUniverseSectionUpdated must not be emitted for a failed update")
}

func TestUpdateSectionAndRefreshIfNeeded_SingleflightFollowerNotCanceled(t *testing.T) {
	// on disk: concurrent callers would get separate, empty in-memory DBs
	_, st, _ := testutil.NewDBOnDisk(t)
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := newTestService(st, new(statuscache.StatusCache))
	const section = app.SectionEveMarketPrices

	entered := make(chan struct{})
	release := make(chan struct{})
	httpmock.RegisterResponder(
		"GET",
		"https://esi.evetech.net/markets/prices",
		func(req *http.Request) (*http.Response, error) {
			close(entered)
			select {
			case <-req.Context().Done():
				return nil, req.Context().Err()
			case <-release:
				return httpmock.NewJsonResponse(200, []map[string]any{})
			}
		},
	)

	// leader: cancelable ctx, e.g. tied to the update scheduler
	ctxLeader, cancelLeader := context.WithCancel(context.Background())
	leaderDone := make(chan struct{})
	go func() {
		defer close(leaderDone)
		s.updateSectionAndRefreshIfNeeded(ctxLeader, section, false)
	}()
	select {
	case <-entered:
		// the leader's HTTP call is now in flight
	case <-time.After(time.Second):
		t.Fatal("leader never reached the HTTP call")
	}

	// follower: independent, never-canceled ctx, e.g. from a manual reload
	followerDone := make(chan struct{})
	go func() {
		defer close(followerDone)
		s.updateSectionAndRefreshIfNeeded(context.Background(), section, false)
	}()
	time.Sleep(20 * time.Millisecond) // let the follower join the in-flight singleflight call

	// when
	cancelLeader()
	<-leaderDone
	<-followerDone

	// then
	status, err := st.GetGeneralSectionStatus(context.Background(), section)
	require.NoError(t, err)
	assert.False(t, status.HasError(), "a canceled leader must not be recorded as a real error via the follower's uncanceled ctx")
	assert.True(t, status.StartedAt.IsZero(), "a canceled update must not leave the section as running")
}

func TestUpdateSectionAndRefreshIfNeeded_ForceRefresh(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := newTestService(st, new(statuscache.StatusCache))
	ctx := context.Background()
	newResponder := func(gotForceRefresh *bool) httpmock.Responder {
		return func(req *http.Request) (*http.Response, error) {
			*gotForceRefresh = xgoesi.IsForceRefresh(req.Context())
			return httpmock.NewJsonResponse(200, []any{})
		}
	}
	t.Run("should mark context for force refresh only when forced", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		var gotForceRefresh bool
		httpmock.RegisterResponder("GET", "https://esi.evetech.net/markets/prices", newResponder(&gotForceRefresh))
		// when
		s.updateSectionAndRefreshIfNeeded(ctx, app.SectionEveMarketPrices, true)
		// then
		assert.True(t, gotForceRefresh)
	})
	t.Run("should not mark context for force refresh when not forced", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		var gotForceRefresh bool
		httpmock.RegisterResponder("GET", "https://esi.evetech.net/markets/prices", newResponder(&gotForceRefresh))
		// when
		s.updateSectionAndRefreshIfNeeded(ctx, app.SectionEveMarketPrices, false)
		// then
		assert.False(t, gotForceRefresh)
	})
}
