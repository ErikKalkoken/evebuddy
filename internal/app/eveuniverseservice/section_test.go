package eveuniverseservice_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/antihax/goesi"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
)

func TestEveuniverseservice_HasSection(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.New(eveuniverseservice.Params{
		Storage:   st,
		ESIClient: goesi.NewAPIClient(nil, ""),
	})
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
