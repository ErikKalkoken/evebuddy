package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestPlanetPin(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can get and create minimal", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		planet := factory.CreateCharacterPlanet()
		input := factory.CreateEveType()
		arg := storage.CreatePlanetPinParams{
			CharacterPlanetID: planet.ID,
			TypeID:            input.ID,
			PinID:             42,
		}
		// when
		err := st.CreatePlanetPin(ctx, arg)
		// then
		if assert.NoError(t, err) {
			c2, err := st.GetPlanetPin(ctx, planet.ID, 42)
			if assert.NoError(t, err) {
				xassert.Equal(t, input, c2.Type)
			}
		}
	})
	t.Run("can get and create complete", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		planet := factory.CreateCharacterPlanet()
		pinType := factory.CreateEveType()
		productType := factory.CreateEveType()
		expiryTime := time.Now()
		installTime := time.Now()
		lastCycleStart := time.Now()
		schematic := factory.CreateEveSchematic()
		factorySchematic := factory.CreateEveSchematic()
		// when
		err := st.CreatePlanetPin(ctx, storage.CreatePlanetPinParams{
			CharacterPlanetID:      planet.ID,
			ExpiryTime:             optional.New(expiryTime),
			ExtractorProductTypeID: optional.New(productType.ID),
			FactorySchematicID:     optional.New(factorySchematic.ID),
			InstallTime:            optional.New(installTime),
			LastCycleStart:         optional.New(lastCycleStart),
			PinID:                  42,
			SchematicID:            optional.New(schematic.ID),
			TypeID:                 pinType.ID,
		})
		// then
		if assert.NoError(t, err) {
			c2, err := st.GetPlanetPin(ctx, planet.ID, 42)
			if assert.NoError(t, err) {
				xassert.Equal(t, pinType, c2.Type)
				xassert.Equal(t, productType, c2.ExtractorProductType.MustValue())
				xassert.Equal2(t, expiryTime, c2.ExpiryTime.MustValue())
				xassert.Equal2(t, installTime, c2.InstallTime.MustValue())
				xassert.Equal2(t, lastCycleStart, c2.LastCycleStart.MustValue())
				xassert.Equal(t, schematic, c2.Schematic.MustValue())
				xassert.Equal(t, factorySchematic, c2.FactorySchematic.MustValue())
			}
		}
	})
	t.Run("can list pins", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		p := factory.CreateCharacterPlanet()
		x1 := factory.CreatePlanetPin(storage.CreatePlanetPinParams{CharacterPlanetID: p.ID})
		x2 := factory.CreatePlanetPin(storage.CreatePlanetPinParams{CharacterPlanetID: p.ID})
		// when
		oo, err := st.ListPlanetPins(ctx, p.ID)
		// then
		if assert.NoError(t, err) {
			got := set.Of[int64]()
			for _, o := range oo {
				got.Add(o.ID)
			}
			want := set.Of(x1.ID, x2.ID)
			xassert.Equal2(t, want, got)
		}
	})
	t.Run("can delete pins", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		planet1 := factory.CreateCharacterPlanet()
		factory.CreatePlanetPin(storage.CreatePlanetPinParams{CharacterPlanetID: planet1.ID})
		factory.CreatePlanetPin(storage.CreatePlanetPinParams{CharacterPlanetID: planet1.ID})
		planet2 := factory.CreateCharacterPlanet()
		factory.CreatePlanetPin(storage.CreatePlanetPinParams{CharacterPlanetID: planet2.ID})
		// when
		err := st.DeletePlanetPins(ctx, planet1.ID)
		// then
		if assert.NoError(t, err) {
			oo1, err := st.ListPlanetPins(ctx, planet1.ID)
			if err != nil {
				t.Fatal(err)
			}
			assert.Len(t, oo1, 0)
			oo2, err := st.ListPlanetPins(ctx, planet2.ID)
			if err != nil {
				t.Fatal(err)
			}
			assert.Len(t, oo2, 1)
		}
	})
}
