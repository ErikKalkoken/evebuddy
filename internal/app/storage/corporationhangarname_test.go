package storage_test

import (
	"context"
	"maps"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

func TestCorporationHangarName(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create from scratch", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		// when
		err := st.UpdateOrCreateCorporationHangarName(ctx, storage.UpdateOrCreateCorporationHangarNameParams{
			CorporationID: c.ID,
			DivisionID:    3,
			Name:          "Alpha",
		})
		// then
		require.NoError(t, err)
		x, err := st.GetCorporationHangarName(ctx, storage.CorporationDivision{
			CorporationID: c.ID,
			Division:      3,
		})
		require.NoError(t, err)
		xassert.Equal(t, c.ID, x.CorporationID)
		xassert.Equal(t, 3, x.Division)
		xassert.Equal(t, "Alpha", x.Name)
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		x1 := factory.CreateCorporationHangarName()
		// when
		err := st.UpdateOrCreateCorporationHangarName(ctx, storage.UpdateOrCreateCorporationHangarNameParams{
			CorporationID: x1.CorporationID,
			DivisionID:    x1.Division.ID(),
			Name:          "Alpha",
		})
		// then
		require.NoError(t, err)
		x, err := st.GetCorporationHangarName(ctx, storage.CorporationDivision{
			CorporationID: x1.CorporationID,
			Division:      app.Division(x1.Division),
		})
		require.NoError(t, err)
		xassert.Equal(t, "Alpha", x.Name)
	})
	t.Run("can list existing entries", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		e1 := factory.CreateCorporationHangarName(storage.UpdateOrCreateCorporationHangarNameParams{
			CorporationID: c.ID,
			DivisionID:    1,
		})
		e2 := factory.CreateCorporationHangarName(storage.UpdateOrCreateCorporationHangarNameParams{
			CorporationID: c.ID,
			DivisionID:    2,
		})
		factory.CreateCorporationHangarName()
		// when
		oo, err := st.ListCorporationHangarNames(ctx, c.ID)
		// then
		require.NoError(t, err)
		got := maps.Collect(xiter.MapSlice2(oo, func(x *app.CorporationHangarName) (app.Division, string) {
			return x.Division, x.Name
		}))
		want := map[app.Division]string{
			e1.Division: e1.Name,
			e2.Division: e2.Name,
		}
		xassert.Equal(t, want, got)
	})
}
