package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

func TestCorporationStructure(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create minimal from scratch", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		system := factory.CreateEveSolarSystem()
		typ := factory.CreateEveType()
		// when
		arg := storage.UpdateOrCreateCorporationStructureParams{
			CorporationID: c.ID,
			Name:          "Alpha",
			ProfileID:     42,
			State:         app.StructureStateShieldVulnerable,
			StructureID:   1234,
			SystemID:      system.ID,
			TypeID:        typ.ID,
		}
		err := st.UpdateOrCreateCorporationStructure(ctx, arg)
		// then
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		got, err := st.GetCorporationStructure(ctx, arg.CorporationID, arg.StructureID)
		if assert.NoError(t, err) {
			assert.EqualValues(t, c.ID, got.CorporationID)
			assert.EqualValues(t, arg.Name, got.Name)
			assert.EqualValues(t, arg.ProfileID, got.ProfileID)
			assert.EqualValues(t, arg.State, got.State)
			assert.EqualValues(t, arg.StructureID, got.StructureID)
			assert.EqualValues(t, arg.SystemID, got.System.ID)
			assert.EqualValues(t, arg.TypeID, got.Type.ID)
			assert.Empty(t, got.Services)
		}
	})
	t.Run("can create full from scratch", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		system := factory.CreateEveSolarSystem()
		typ := factory.CreateEveType()
		// when
		arg := storage.UpdateOrCreateCorporationStructureParams{
			CorporationID:      c.ID,
			FuelExpires:        optional.New(factory.RandomTime()),
			Name:               "Alpha",
			NextReinforceApply: optional.New(factory.RandomTime()),
			NextReinforceHour:  optional.New(int64(8)),
			ProfileID:          42,
			ReinforceHour:      optional.New(int64(12)),
			State:              app.StructureStateShieldVulnerable,
			StateTimerEnd:      optional.New(factory.RandomTime()),
			StateTimerStart:    optional.New(factory.RandomTime()),
			StructureID:        1234,
			SystemID:           system.ID,
			TypeID:             typ.ID,
			UnanchorsAt:        optional.New(factory.RandomTime()),
			Services: []storage.StructureServiceParams{
				{
					Name:  "Jupiter",
					State: app.StructureServiceStateOnline,
				},
			},
		}
		err := st.UpdateOrCreateCorporationStructure(ctx, arg)
		// then
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		got, err := st.GetCorporationStructure(ctx, arg.CorporationID, arg.StructureID)
		if assert.NoError(t, err) {
			assert.EqualValues(t, c.ID, got.CorporationID)
			assert.EqualValues(t, arg.FuelExpires.ValueOrZero(), got.FuelExpires.ValueOrZero())
			assert.EqualValues(t, arg.Name, got.Name)
			assert.EqualValues(t, arg.NextReinforceApply.ValueOrZero(), got.NextReinforceApply.ValueOrZero())
			assert.EqualValues(t, arg.NextReinforceHour.ValueOrZero(), got.NextReinforceHour.ValueOrZero())
			assert.EqualValues(t, arg.ProfileID, got.ProfileID)
			assert.EqualValues(t, arg.ReinforceHour.ValueOrZero(), got.ReinforceHour.ValueOrZero())
			assert.EqualValues(t, arg.State, got.State)
			assert.EqualValues(t, arg.StateTimerEnd.ValueOrZero(), got.StateTimerEnd.ValueOrZero())
			assert.EqualValues(t, arg.StateTimerStart.ValueOrZero(), got.StateTimerStart.ValueOrZero())
			assert.EqualValues(t, arg.StructureID, got.StructureID)
			assert.EqualValues(t, arg.SystemID, got.System.ID)
			assert.EqualValues(t, arg.TypeID, got.Type.ID)
			assert.EqualValues(t, arg.UnanchorsAt.ValueOrZero(), got.UnanchorsAt.ValueOrZero())
			assert.EqualValues(t, "Jupiter", got.Services[0].Name)
			assert.EqualExportedValues(t, app.StructureServiceStateOnline, got.Services[0].State)
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		x1 := factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{
			CorporationID: c.ID,
			State:         app.StructureStateAnchorVulnerable,
			Services: []storage.StructureServiceParams{
				{
					Name:  "Venus",
					State: app.StructureServiceStateOffline,
				},
			},
		})
		// when
		arg := storage.UpdateOrCreateCorporationStructureParams{
			CorporationID:      c.ID,
			FuelExpires:        optional.New(factory.RandomTime()),
			Name:               "Alpha",
			NextReinforceApply: optional.New(factory.RandomTime()),
			NextReinforceHour:  optional.New(int64(8)),
			ProfileID:          x1.ProfileID,
			ReinforceHour:      optional.New(int64(12)),
			State:              app.StructureStateShieldVulnerable,
			StateTimerEnd:      optional.New(factory.RandomTime()),
			StateTimerStart:    optional.New(factory.RandomTime()),
			StructureID:        x1.StructureID,
			SystemID:           x1.System.ID,
			TypeID:             x1.Type.ID,
			UnanchorsAt:        optional.New(factory.RandomTime()),
			Services: []storage.StructureServiceParams{
				{
					Name:  "Jupiter",
					State: app.StructureServiceStateOnline,
				},
			},
		}
		err := st.UpdateOrCreateCorporationStructure(ctx, arg)
		// then
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		got, err := st.GetCorporationStructure(ctx, x1.CorporationID, x1.StructureID)
		if assert.NoError(t, err) {
			assert.EqualValues(t, c.ID, got.CorporationID)
			assert.EqualValues(t, arg.FuelExpires.ValueOrZero(), got.FuelExpires.ValueOrZero())
			assert.EqualValues(t, arg.Name, got.Name)
			assert.EqualValues(t, arg.NextReinforceApply.ValueOrZero(), got.NextReinforceApply.ValueOrZero())
			assert.EqualValues(t, arg.NextReinforceHour.ValueOrZero(), got.NextReinforceHour.ValueOrZero())
			assert.EqualValues(t, arg.ReinforceHour.ValueOrZero(), got.ReinforceHour.ValueOrZero())
			assert.EqualValues(t, arg.State, got.State)
			assert.EqualValues(t, arg.StateTimerEnd.ValueOrZero(), got.StateTimerEnd.ValueOrZero())
			assert.EqualValues(t, arg.StateTimerStart.ValueOrZero(), got.StateTimerStart.ValueOrZero())
			assert.EqualValues(t, arg.UnanchorsAt.ValueOrZero(), got.UnanchorsAt.ValueOrZero())
			assert.EqualValues(t, "Jupiter", got.Services[0].Name)
			assert.EqualExportedValues(t, app.StructureServiceStateOnline, got.Services[0].State)
		}
	})
	t.Run("can list structure IDs for corporation", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		o1 := factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{
			CorporationID: c.ID,
		})
		o2 := factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{
			CorporationID: c.ID,
		})
		factory.CreateCorporationStructure()
		// when
		got, err := st.ListCorporationStructureIDs(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			want := set.Of(o1.StructureID, o2.StructureID)
			assert.Equal(t, want, got)
		}
	})
	t.Run("can list structures for corporation", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		o1 := factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{
			CorporationID: c.ID,
		})
		o2 := factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{
			CorporationID: c.ID,
		})
		factory.CreateCorporationStructure()
		// when
		xx, err := st.ListCorporationStructures(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			got := set.Collect(xiter.MapSlice(xx, func(x *app.CorporationStructure) int64 {
				return x.StructureID
			}))
			want := set.Of(o1.StructureID, o2.StructureID)
			assert.Equal(t, want, got)
		}
	})
	t.Run("can delete structures for a corporation", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		o1 := factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{
			CorporationID: c.ID,
		})
		o2 := factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{
			CorporationID: c.ID,
		})
		// when
		err := st.DeleteCorporationStructures(ctx, c.ID, set.Of(o1.StructureID))
		// then
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		got, err := st.ListCorporationStructureIDs(ctx, c.ID)
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		want := set.Of(o2.StructureID)
		assert.Equal(t, want, got)
	})
}

func TestStructureService(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can get and create minimal", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		structure := factory.CreateCorporationStructure()
		arg := storage.CreateStructureServiceParams{
			CorporationStructureID: structure.ID,
			Name:                   "Alpha",
			State:                  app.StructureServiceStateOnline,
		}
		// when
		err := st.CreateStructureService(ctx, arg)
		// then
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		got, err := st.GetStructureService(ctx, structure.ID, "Alpha")
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		assert.Equal(t, "Alpha", got.Name)
		assert.Equal(t, app.StructureServiceStateOnline, got.State)
	})
	t.Run("can list services", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		s := factory.CreateCorporationStructure()
		x1 := factory.CreateStructureService(storage.CreateStructureServiceParams{CorporationStructureID: s.ID})
		x2 := factory.CreateStructureService(storage.CreateStructureServiceParams{CorporationStructureID: s.ID})
		// when
		oo, err := st.ListStructureServices(ctx, s.ID)
		// then
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		got := set.Collect(xiter.MapSlice(oo, func(x *app.StructureService) string {
			return x.Name
		}))
		want := set.Of(x1.Name, x2.Name)
		xassert.EqualSet(t, want, got)
	})
	t.Run("can delete services", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		structure := factory.CreateCorporationStructure()
		factory.CreateStructureService(storage.CreateStructureServiceParams{CorporationStructureID: structure.ID})
		factory.CreateStructureService(storage.CreateStructureServiceParams{CorporationStructureID: structure.ID})
		x := factory.CreateStructureService()
		// when
		err := st.DeleteStructureServices(ctx, structure.ID)
		// then
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		oo1, err := st.ListStructureServices(ctx, structure.ID)
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		assert.Empty(t, oo1)
		_, err2 := st.GetStructureService(ctx, x.CorporationStructureID, x.Name)
		if !assert.NoError(t, err2) {
			t.Fatal()
		}
	})
}
