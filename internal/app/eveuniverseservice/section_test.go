package eveuniverseservice_test

import (
	"context"
	"database/sql"
	"net/http"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
)

func TestEveuniverseservice_HasSection(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
	section := app.SectionEveTypes
	ctx := context.Background()
	t.Run("should report true when exists", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
			Section: section,
		})
		// when
		got, err := s.HasSection(ctx, section)
		// then
		require.NoError(t, err)
		assert.True(t, got)
	})
	t.Run("should report false when not exists", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		// when
		got, err := s.HasSection(ctx, section)
		// then
		require.NoError(t, err)
		assert.False(t, got)
	})
	t.Run("should report false when exist, but incomplete", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
			Section: section,
		})
		_, err := st.UpdateOrCreateGeneralSectionStatus(ctx, storage.UpdateOrCreateGeneralSectionStatusParams{
			Section:     section,
			CompletedAt: &sql.NullTime{},
		})
		require.NoError(t, err)
		// when
		got, err := s.HasSection(ctx, section)
		// then
		require.NoError(t, err)
		assert.False(t, got)
	})
}

func TestEveuniverseservice_AsyncUpdates(t *testing.T) {
	_, st, _ := testutil.NewDBOnDisk(t)
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	const section = app.SectionEveMarketPrices

	t.Run("Stop cancels and waits for a running update", func(t *testing.T) {
		httpmock.Reset()
		s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
		entered := make(chan struct{})
		canceled := make(chan struct{})
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/markets/prices",
			func(req *http.Request) (*http.Response, error) {
				close(entered)
				<-req.Context().Done()
				close(canceled)
				return nil, req.Context().Err()
			},
		)
		s.UpdateSectionAndRefreshIfNeededAsync(section, true)
		select {
		case <-entered:
		case <-time.After(time.Second):
			t.Fatal("update never reached the HTTP call")
		}
		// when
		stopped := make(chan struct{})
		go func() {
			s.Stop()
			close(stopped)
		}()
		// then
		select {
		case <-stopped:
		case <-time.After(time.Second):
			t.Fatal("Stop did not return")
		}
		x, err := st.GetGeneralSectionStatus(context.Background(), section)
		require.NoError(t, err)
		assert.True(t, x.StartedAt.IsZero(), "Stop must wait for the update to finish")
		assert.False(t, x.HasError(), "a canceled update must not be recorded as an error")
		// httpmock can return ctx.Err() before the responder sees the cancel
		select {
		case <-canceled:
		case <-time.After(time.Second):
			t.Fatal("request was not canceled")
		}
	})
	t.Run("skips updates after Stop", func(t *testing.T) {
		httpmock.Reset()
		s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
		s.Stop()
		// when
		s.UpdateSectionsIfNeededAsync(true)
		s.UpdateSectionAndRefreshIfNeededAsync(section, true)
		s.Stop()
		// then
		assert.Zero(t, httpmock.GetTotalCallCount())
	})
}

func TestEveuniverseservice_Init(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	o := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
		Section: app.SectionEveTypes,
	})
	require.False(t, o.StartedAt.IsZero())
	scs := new(statuscache.StatusCache)
	require.NoError(t, scs.Init(ctx, st))
	x0, _ := scs.EveUniverseSection(app.SectionEveTypes)
	require.True(t, x0.IsRunning())
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st, StatusCacheService: scs})
	// when
	err := s.Init(ctx)
	// then
	require.NoError(t, err)
	got, err := st.GetGeneralSectionStatus(ctx, app.SectionEveTypes)
	require.NoError(t, err)
	assert.True(t, got.StartedAt.IsZero())
	x, ok := scs.EveUniverseSection(app.SectionEveTypes)
	require.True(t, ok)
	assert.False(t, x.IsRunning())
}
