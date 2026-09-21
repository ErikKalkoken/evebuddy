package eveuniverseservice_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
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

func TestEveuniverseservice_UpdateSectionAndRefreshIfNeeded_DoesNotReportSuccessOnFailedUpdate(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	sig := app.NewSignals()
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st, Signals: sig})
	ctx := context.Background()
	httpmock.Reset()
	httpmock.RegisterResponder(
		"GET",
		"https://esi.evetech.net/markets/prices",
		httpmock.NewErrorResponder(fmt.Errorf("failed")),
	)
	var updated bool
	sig.EveUniverseSectionUpdated.AddListener(func(ctx context.Context, arg app.EveUniverseSectionUpdated) {
		updated = true
	})
	// when
	s.UpdateSectionAndRefreshIfNeeded(ctx, app.SectionEveMarketPrices, false)
	// then
	status, err := st.GetGeneralSectionStatus(ctx, app.SectionEveMarketPrices)
	require.NoError(t, err)
	assert.True(t, status.HasError(), "expected section status to record the ESI failure")
	assert.False(t, updated, "EveUniverseSectionUpdated must not be emitted for a failed update")
}

func TestEveuniverseservice_UpdateSectionAndRefreshIfNeeded_ForceRefresh(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
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
		s.UpdateSectionAndRefreshIfNeeded(ctx, app.SectionEveMarketPrices, true)
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
		s.UpdateSectionAndRefreshIfNeeded(ctx, app.SectionEveMarketPrices, false)
		// then
		assert.False(t, gotForceRefresh)
	})
}
