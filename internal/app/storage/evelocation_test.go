package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestLocation(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create minimal", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		updatedAt := time.Now()
		arg := storage.UpdateOrCreateLocationParams{
			ID:        42,
			Name:      "Alpha",
			UpdatedAt: updatedAt,
		}
		// when
		err := st.UpdateOrCreateEveLocation(ctx, arg)
		// then
		if assert.NoError(t, err) {
			x, err := st.GetLocation(ctx, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, "Alpha", x.Name)
				assert.Equal(t, updatedAt.UTC(), x.UpdatedAt.UTC())
			}
		}
	})
	t.Run("can create new full", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		owner := factory.CreateEveEntityCorporation()
		system := factory.CreateEveSolarSystem()
		myType := factory.CreateEveType()
		updatedAt := time.Now()
		arg := storage.UpdateOrCreateLocationParams{
			ID:            42,
			SolarSystemID: optional.New(system.ID),
			TypeID:        optional.New(myType.ID),
			Name:          "Alpha",
			OwnerID:       optional.New(owner.ID),
			UpdatedAt:     updatedAt,
		}
		// when
		err := st.UpdateOrCreateEveLocation(ctx, arg)
		// then
		if assert.NoError(t, err) {
			x, err := st.GetLocation(ctx, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, "Alpha", x.Name)
				assert.Equal(t, owner, x.Owner)
				assert.Equal(t, system, x.SolarSystem)
				assert.Equal(t, myType, x.Type)
				assert.Equal(t, updatedAt.UTC(), x.UpdatedAt.UTC())
			}
		}
	})
	t.Run("can get existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 42, Name: "Alpha"})
		// when
		x, err := st.GetLocation(ctx, 42)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, "Alpha", x.Name)
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 42})
		owner := factory.CreateEveEntityCorporation()
		system := factory.CreateEveSolarSystem()
		myType := factory.CreateEveType()
		updatedAt := time.Now()
		arg := storage.UpdateOrCreateLocationParams{
			ID:            42,
			SolarSystemID: optional.New(system.ID),
			TypeID:        optional.New(myType.ID),
			Name:          "Alpha",
			OwnerID:       optional.New(owner.ID),
			UpdatedAt:     updatedAt,
		}
		// when
		err := st.UpdateOrCreateEveLocation(ctx, arg)
		// then
		if assert.NoError(t, err) {
			x, err := st.GetLocation(ctx, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, "Alpha", x.Name)
				assert.Equal(t, owner, x.Owner)
				assert.Equal(t, system, x.SolarSystem)
				assert.Equal(t, myType, x.Type)
				assert.Equal(t, updatedAt.UTC(), x.UpdatedAt.UTC())
			}
		}
	})
	t.Run("can list locations", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		l1 := factory.CreateEveLocationStructure()
		l2 := factory.CreateEveLocationStructure()
		// when
		got, err := st.ListEveLocation(ctx)
		if assert.NoError(t, err) {
			want := []*app.EveLocation{l1, l2}
			assert.Equal(t, want, got)
		}
	})
	t.Run("can list location IDs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		l1 := factory.CreateEveLocationStructure()
		l2 := factory.CreateEveLocationStructure()
		// when
		got, err := st.ListEveLocationIDs(ctx)
		if assert.NoError(t, err) {
			want := set.Of(l1.ID, l2.ID)
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
	t.Run("can list locations in solar system", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		s := factory.CreateEveSolarSystem()
		l1 := factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{SolarSystemID: optional.New(s.ID)})
		l2 := factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{SolarSystemID: optional.New(s.ID)})
		factory.CreateEveLocationStructure()
		// when
		got, err := st.ListEveLocationInSolarSystem(ctx, s.ID)
		if assert.NoError(t, err) {
			gotIDs := xslices.Map(got, func(x *app.EveLocation) int64 {
				return x.ID
			})
			assert.ElementsMatch(t, []int64{l1.ID, l2.ID}, gotIDs)
		}
	})
	t.Run("can return IDs of missing locations", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{
			ID: 42,
		})
		// when
		got, err := st.MissingEveLocations(ctx, set.Of[int64](42, 99))
		if assert.NoError(t, err) {
			want := set.Of[int64](99)
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
}
