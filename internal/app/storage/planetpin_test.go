package storage_test

import (
	"context"
	"maps"
	"testing"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
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
		require.NoError(t, err)
		c2, err := st.GetPlanetPin(ctx, planet.ID, 42)
		require.NoError(t, err)
		xassert.Equal(t, input, c2.Type)
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
		contentType1 := factory.CreateEveType(storage.CreateEveTypeParams{Name: "Alpha"})
		contentType2 := factory.CreateEveType(storage.CreateEveTypeParams{Name: "Bravo"})
		// when
		err := st.CreatePlanetPin(ctx, storage.CreatePlanetPinParams{
			CharacterPlanetID: planet.ID,
			Contents: map[int64]int64{
				contentType1.ID: 42,
				contentType2.ID: 7,
			},
			ExpiryTime:             optional.New(expiryTime),
			ExtractorCycleTime:     optional.New(30 * time.Minute),
			ExtractorHeadRadius:    optional.New(0.013),
			ExtractorNumHeads:      optional.New[int64](10),
			ExtractorProductTypeID: optional.New(productType.ID),
			ExtractorQtyPerCycle:   optional.New[int64](1081),
			FactorySchematicID:     optional.New(factorySchematic.ID),
			InstallTime:            optional.New(installTime),
			LastCycleStart:         optional.New(lastCycleStart),
			PinID:                  42,
			SchematicID:            optional.New(schematic.ID),
			TypeID:                 pinType.ID,
		})
		// then
		require.NoError(t, err)
		c2, err := st.GetPlanetPin(ctx, planet.ID, 42)
		require.NoError(t, err)
		xassert.Equal(t, pinType, c2.Type)
		xassert.EqualOptional(t, productType, c2.ExtractorProductType)
		xassert.EqualOptional(t, expiryTime, c2.ExpiryTime)
		xassert.EqualOptional(t, installTime, c2.InstallTime)
		xassert.EqualOptional(t, lastCycleStart, c2.LastCycleStart)
		xassert.EqualOptional(t, schematic, c2.Schematic)
		xassert.EqualOptional(t, factorySchematic, c2.FactorySchematic)
		xassert.EqualOptional(t, 30*time.Minute, c2.ExtractorCycleTime)
		xassert.EqualOptional(t, 0.013, c2.ExtractorHeadRadius)
		xassert.EqualOptional(t, 10, c2.ExtractorNumHeads)
		xassert.EqualOptional(t, 1081, c2.ExtractorQtyPerCycle)
		if assert.Len(t, c2.Contents, 2) {
			xassert.Equal(t, contentType1, c2.Contents[0].Type)
			xassert.Equal(t, 42, c2.Contents[0].Amount)
			xassert.Equal(t, contentType2, c2.Contents[1].Type)
			xassert.Equal(t, 7, c2.Contents[1].Amount)
		}
	})
	t.Run("can list pins", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		p := factory.CreateCharacterPlanet()
		product := factory.CreateEveType()
		x1 := factory.CreatePlanetPin(storage.CreatePlanetPinParams{
			CharacterPlanetID:      p.ID,
			ExtractorProductTypeID: optional.New(product.ID),
		})
		x2 := factory.CreatePlanetPin(storage.CreatePlanetPinParams{CharacterPlanetID: p.ID})
		// when
		oo, err := st.ListPlanetPins(ctx, p.ID)
		// then
		require.NoError(t, err)
		got := make(map[int64]*app.PlanetPin)
		for _, o := range oo {
			got[o.ID] = o
		}
		xassert.Equal(t, set.Of(x1.ID, x2.ID), set.Collect(maps.Keys(got)))
		xassert.EqualOptional(t, product, got[x1.ID].ExtractorProductType)
		assert.True(t, got[x2.ID].ExtractorProductType.IsEmpty())
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
		require.NoError(t, err)
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
	})
	t.Run("should delete contents with pins", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		planet := factory.CreateCharacterPlanet()
		contentType := factory.CreateEveType()
		factory.CreatePlanetPin(storage.CreatePlanetPinParams{
			CharacterPlanetID: planet.ID,
			Contents:          map[int64]int64{contentType.ID: 42},
		})
		// when
		err := st.DeletePlanetPins(ctx, planet.ID)
		// then
		require.NoError(t, err)
		var n int
		err = db.QueryRow("SELECT COUNT(*) FROM planet_pin_contents").Scan(&n)
		require.NoError(t, err)
		assert.Equal(t, 0, n)
	})
}
