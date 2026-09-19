package eveuniverseservice_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
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

func TestEveuniverseservice_UpdateTicker_StopWithoutStart(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
	// when
	done := make(chan struct{})
	go func() {
		s.StopUpdateTicker()
		close(done)
	}()
	// then
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("StopUpdateTicker did not return")
	}
}

func TestEveuniverseservice_UpdateTicker_StartThenStop(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
	s.StartUpdateTicker(10 * time.Millisecond)
	time.Sleep(50 * time.Millisecond) // let at least one tick fire
	// when
	done := make(chan struct{})
	go func() {
		s.StopUpdateTicker()
		close(done)
	}()
	// then
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("StopUpdateTicker did not return within timeout")
	}
}

func TestEveuniverseservice_UpdateTicker_StopIsIdempotent(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
	s.StartUpdateTicker(10 * time.Millisecond)
	s.StopUpdateTicker()
	// when/then
	assert.NotPanics(t, func() {
		s.StopUpdateTicker()
	})
}
