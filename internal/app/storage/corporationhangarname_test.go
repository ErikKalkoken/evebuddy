package storage_test

import (
	"context"
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
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
		if assert.NoError(t, err) {
			x, err := st.GetCorporationHangarName(ctx, storage.CorporationDivision{
				CorporationID: c.ID,
				DivisionID:    3,
			})
			if assert.NoError(t, err) {
				assert.EqualValues(t, c.ID, x.CorporationID)
				assert.EqualValues(t, 3, x.DivisionID)
				assert.EqualValues(t, "Alpha", x.Name)
			}
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		x1 := factory.CreateCorporationHangarName()
		// when
		err := st.UpdateOrCreateCorporationHangarName(ctx, storage.UpdateOrCreateCorporationHangarNameParams{
			CorporationID: x1.CorporationID,
			DivisionID:    x1.DivisionID,
			Name:          "Alpha",
		})
		// then
		if assert.NoError(t, err) {
			x, err := st.GetCorporationHangarName(ctx, storage.CorporationDivision{
				CorporationID: x1.CorporationID,
				DivisionID:    x1.DivisionID,
			})
			if assert.NoError(t, err) {
				assert.EqualValues(t, "Alpha", x.Name)
			}
		}
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
		if assert.NoError(t, err) {
			got := maps.Collect(xiter.MapSlice2(oo, func(x *app.CorporationHangarName) (int32, string) {
				return x.DivisionID, x.Name
			}))
			want := map[int32]string{
				e1.DivisionID: e1.Name,
				e2.DivisionID: e2.Name,
			}
			assert.Equal(t, want, got)
		}
	})
}
