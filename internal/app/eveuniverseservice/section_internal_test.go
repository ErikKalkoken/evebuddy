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
