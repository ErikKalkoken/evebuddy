package eveuniverseservice_test

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/go-set"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestAddMissingEveEntitiesAndLocations(t *testing.T) {
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
		err := s.AddMissingEveEntitiesAndLocations(ctx, set.Of[int64](47), set.Of[int64]())
		// then
		require.NoError(t, err)
		o, err := st.GetEveEntity(ctx, 47)
		require.NoError(t, err)
		xassert.Equal(t, "Erik", o.Name)
		xassert.Equal(t, app.EveEntityCharacter, o.Category)
	})
	t.Run("should do nothing when both sets are empty", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		// when
		err := s.AddMissingEveEntitiesAndLocations(ctx, set.Of[int64](), set.Of[int64]())
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
		err := s.AddMissingEveEntitiesAndLocations(ctx, set.Of[int64](), set.Of(loc.ID))
		// then
		require.NoError(t, err)
		xassert.Equal(t, 0, httpmock.GetTotalCallCount())
	})
}
