package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

func TestCorporationStructure(t *testing.T) {
	db, r, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create minimal from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
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
		err := r.UpdateOrCreateCorporationStructure(ctx, arg)
		// then
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		got, err := r.GetCorporationStructure(ctx, arg.CorporationID, arg.StructureID)
		if assert.NoError(t, err) {
			assert.EqualValues(t, c.ID, got.CorporationID)
			assert.EqualValues(t, arg.Name, got.Name)
			assert.EqualValues(t, arg.ProfileID, got.ProfileID)
			assert.EqualValues(t, arg.State, got.State)
			assert.EqualValues(t, arg.StructureID, got.StructureID)
			assert.EqualValues(t, arg.SystemID, got.System.ID)
			assert.EqualValues(t, arg.TypeID, got.Type.ID)
		}
	})
	t.Run("can create full from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
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
		}
		err := r.UpdateOrCreateCorporationStructure(ctx, arg)
		// then
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		got, err := r.GetCorporationStructure(ctx, arg.CorporationID, arg.StructureID)
		if assert.NoError(t, err) {
			assert.EqualValues(t, c.ID, got.CorporationID)
			assert.EqualValues(t, arg.FuelExpires.MustValue(), got.FuelExpires.MustValue())
			assert.EqualValues(t, arg.Name, got.Name)
			assert.EqualValues(t, arg.NextReinforceApply.MustValue(), got.NextReinforceApply.MustValue())
			assert.EqualValues(t, arg.NextReinforceHour.MustValue(), got.NextReinforceHour.MustValue())
			assert.EqualValues(t, arg.ProfileID, got.ProfileID)
			assert.EqualValues(t, arg.ReinforceHour.MustValue(), got.ReinforceHour.MustValue())
			assert.EqualValues(t, arg.State, got.State)
			assert.EqualValues(t, arg.StateTimerEnd.MustValue(), got.StateTimerEnd.MustValue())
			assert.EqualValues(t, arg.StateTimerStart.MustValue(), got.StateTimerStart.MustValue())
			assert.EqualValues(t, arg.StructureID, got.StructureID)
			assert.EqualValues(t, arg.SystemID, got.System.ID)
			assert.EqualValues(t, arg.TypeID, got.Type.ID)
			assert.EqualValues(t, arg.UnanchorsAt.MustValue(), got.UnanchorsAt.MustValue())
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCorporation()
		x1 := factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{
			CorporationID: c.ID,
			State:         app.StructureStateAnchorVulnerable,
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
			StructureID:        x1.ProfileID,
			SystemID:           x1.System.ID,
			TypeID:             x1.Type.ID,
			UnanchorsAt:        optional.New(factory.RandomTime()),
		}
		err := r.UpdateOrCreateCorporationStructure(ctx, arg)
		// then
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		got, err := r.GetCorporationStructure(ctx, arg.CorporationID, arg.StructureID)
		if assert.NoError(t, err) {
			assert.EqualValues(t, c.ID, got.CorporationID)
			assert.EqualValues(t, arg.FuelExpires.MustValue(), got.FuelExpires.MustValue())
			assert.EqualValues(t, arg.Name, got.Name)
			assert.EqualValues(t, arg.NextReinforceApply.MustValue(), got.NextReinforceApply.MustValue())
			assert.EqualValues(t, arg.NextReinforceHour.MustValue(), got.NextReinforceHour.MustValue())
			assert.EqualValues(t, arg.ReinforceHour.MustValue(), got.ReinforceHour.MustValue())
			assert.EqualValues(t, arg.State, got.State)
			assert.EqualValues(t, arg.StateTimerEnd.MustValue(), got.StateTimerEnd.MustValue())
			assert.EqualValues(t, arg.StateTimerStart.MustValue(), got.StateTimerStart.MustValue())
			assert.EqualValues(t, arg.UnanchorsAt.MustValue(), got.UnanchorsAt.MustValue())
		}
	})
	t.Run("can list structure IDs for corporation", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCorporation()
		o1 := factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{
			CorporationID: c.ID,
		})
		o2 := factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{
			CorporationID: c.ID,
		})
		factory.CreateCorporationStructure()
		// when
		got, err := r.ListCorporationStructureIDs(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			want := set.Of(o1.StructureID, o2.StructureID)
			assert.Equal(t, want, got)
		}
	})
	t.Run("can list structures for corporation", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCorporation()
		o1 := factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{
			CorporationID: c.ID,
		})
		o2 := factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{
			CorporationID: c.ID,
		})
		factory.CreateCorporationStructure()
		// when
		xx, err := r.ListCorporationStructures(ctx, c.ID)
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
		testutil.TruncateTables(db)
		c := factory.CreateCorporation()
		o1 := factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{
			CorporationID: c.ID,
		})
		o2 := factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{
			CorporationID: c.ID,
		})
		// when
		err := r.DeleteCorporationStructures(ctx, c.ID, set.Of(o1.StructureID))
		// then
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		got, err := r.ListCorporationStructureIDs(ctx, c.ID)
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		want := set.Of(o2.StructureID)
		assert.Equal(t, want, got)
	})
}
