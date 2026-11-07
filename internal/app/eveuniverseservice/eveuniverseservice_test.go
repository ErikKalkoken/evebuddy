package eveuniverseservice_test

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/go-set"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestAddMissing(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ctx := context.Background()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
	t.Run("should resolve missing entities", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			"https://esi.evetech.net/universe/names",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{"id": 47, "name": "Erik", "category": "character"},
			}),
		)
		// when
		err := s.AddMissing(ctx, set.Of[int64](47), set.Of[int64](), set.Of[int64]())
		// then
		require.NoError(t, err)
		o, err := st.GetEveEntity(ctx, 47)
		require.NoError(t, err)
		xassert.Equal(t, "Erik", o.Name)
		xassert.Equal(t, app.EveEntityCharacter, o.Category)
	})
	t.Run("should do nothing when all sets are empty", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		// when
		err := s.AddMissing(ctx, set.Of[int64](), set.Of[int64](), set.Of[int64]())
		// then
		require.NoError(t, err)
		xassert.Equal(t, 0, httpmock.GetTotalCallCount())
	})
	t.Run("should do nothing when locations already exist", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		loc := factory.CreateEveLocationStation()
		// when
		err := s.AddMissing(ctx, set.Of[int64](), set.Of(loc.ID), set.Of[int64]())
		// then
		require.NoError(t, err)
		xassert.Equal(t, 0, httpmock.GetTotalCallCount())
	})
	t.Run("should resolve missing types", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		factory.CreateEveGroup(storage.CreateEveGroupParams{ID: 25})
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi.evetech.net/universe/types/\d+`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"description": "The Rifter is a...",
				"group_id":    25,
				"name":        "Rifter",
				"published":   true,
				"type_id":     587,
			}),
		)
		// when
		err := s.AddMissing(ctx, set.Of[int64](), set.Of[int64](), set.Of[int64](587))
		// then
		require.NoError(t, err)
		o, err := st.GetEveType(ctx, 587)
		require.NoError(t, err)
		xassert.Equal(t, "Rifter", o.Name)
	})
	t.Run("should do nothing when types already exist", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		et := factory.CreateEveType()
		// when
		err := s.AddMissing(ctx, set.Of[int64](), set.Of[int64](), set.Of(et.ID))
		// then
		require.NoError(t, err)
		xassert.Equal(t, 0, httpmock.GetTotalCallCount())
	})
}
