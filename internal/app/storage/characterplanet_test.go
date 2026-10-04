package storage_test

import (
	"context"
	"errors"
	"slices"
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

func TestPlanet(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can list planets", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		p1 := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{CharacterID: c.ID})
		p2 := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{CharacterID: c.ID})
		p3 := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{CharacterID: c.ID})
		x1 := factory.CreatePlanetPin(storage.CreatePlanetPinParams{CharacterPlanetID: p1.ID})
		x2 := factory.CreatePlanetPin(storage.CreatePlanetPinParams{CharacterPlanetID: p1.ID})
		x3 := factory.CreatePlanetPin(storage.CreatePlanetPinParams{CharacterPlanetID: p2.ID})
		// when
		oo, err := st.ListCharacterPlanets(ctx, c.ID)
		// then
		require.NoError(t, err)
		assert.Len(t, oo, 3)
		assert.ElementsMatch(
			t,
			[]int64{p1.EvePlanet.ID, p2.EvePlanet.ID, p3.EvePlanet.ID},
			[]int64{oo[0].EvePlanet.ID, oo[1].EvePlanet.ID, oo[2].EvePlanet.ID},
		)
		got := pinIDsByPlanet(oo)
		xassert.Equal(t, set.Of(x1.ID, x2.ID), got[p1.ID])
		xassert.Equal(t, set.Of(x3.ID), got[p2.ID])
		xassert.Equal(t, set.Of[int64](), got[p3.ID])
	})
	t.Run("can delete planets", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		p1 := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{CharacterID: c.ID})
		p2 := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{CharacterID: c.ID})
		p3 := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{CharacterID: c.ID})
		// when
		err := st.DeleteCharacterPlanet(ctx, c.ID, set.Of(p1.EvePlanet.ID, p2.EvePlanet.ID))
		// then
		require.NoError(t, err)
		oo, err := st.ListCharacterPlanets(ctx, c.ID)
		if err != nil {
			t.Fatal(err)
		}
		assert.Len(t, oo, 1)
		assert.ElementsMatch(t, []int64{p3.EvePlanet.ID}, []int64{oo[0].EvePlanet.ID})
	})
	t.Run("can update last notified", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		planet := factory.CreateCharacterPlanet()
		lastNotified := factory.RandomTime()
		arg := storage.UpdateCharacterPlanetLastNotifiedParams{
			CharacterID:  planet.CharacterID,
			EvePlanetID:  planet.EvePlanet.ID,
			LastNotified: lastNotified,
		}
		// when
		err := st.UpdateCharacterPlanetLastNotified(ctx, arg)
		// then
		require.NoError(t, err)
		i, err := st.GetCharacterPlanet(ctx, planet.CharacterID, planet.EvePlanet.ID)
		require.NoError(t, err)
		xassert.EqualOptional(t, lastNotified, i.LastNotified)
	})
	t.Run("can list planets from all characters", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateCharacterFull()
		p1 := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{CharacterID: c1.ID})
		p2 := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{CharacterID: c1.ID})
		c2 := factory.CreateCharacterFull()
		p3 := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{CharacterID: c2.ID})
		x1 := factory.CreatePlanetPin(storage.CreatePlanetPinParams{CharacterPlanetID: p1.ID})
		x2 := factory.CreatePlanetPin(storage.CreatePlanetPinParams{CharacterPlanetID: p3.ID})
		x3 := factory.CreatePlanetPin(storage.CreatePlanetPinParams{CharacterPlanetID: p3.ID})
		// when
		oo, err := st.ListAllCharacterPlanets(ctx)
		// then
		require.NoError(t, err)
		assert.Len(t, oo, 3)
		assert.ElementsMatch(
			t,
			[]int64{p1.EvePlanet.ID, p2.EvePlanet.ID, p3.EvePlanet.ID},
			[]int64{oo[0].EvePlanet.ID, oo[1].EvePlanet.ID, oo[2].EvePlanet.ID},
		)
		got := pinIDsByPlanet(oo)
		xassert.Equal(t, set.Of(x1.ID), got[p1.ID])
		xassert.Equal(t, set.Of[int64](), got[p2.ID])
		xassert.Equal(t, set.Of(x2.ID, x3.ID), got[p3.ID])
	})
	t.Run("can list planets with extractor product types", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		p1 := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{CharacterID: c.ID})
		p2 := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{CharacterID: c.ID})
		product := factory.CreateEveType()
		x1 := factory.CreatePlanetPin(storage.CreatePlanetPinParams{
			CharacterPlanetID:      p1.ID,
			ExtractorProductTypeID: optional.New(product.ID),
		})
		x2 := factory.CreatePlanetPin(storage.CreatePlanetPinParams{
			CharacterPlanetID:      p2.ID,
			ExtractorProductTypeID: optional.New(product.ID),
		})
		x3 := factory.CreatePlanetPin(storage.CreatePlanetPinParams{CharacterPlanetID: p2.ID})
		// when
		oo, err := st.ListCharacterPlanets(ctx, c.ID)
		// then
		require.NoError(t, err)
		pins := make(map[int64]*app.PlanetPin)
		for _, p := range oo {
			for _, x := range p.Pins {
				pins[x.ID] = x
			}
		}
		require.Len(t, pins, 3)
		xassert.EqualOptional(t, product, pins[x1.ID].ExtractorProductType)
		xassert.EqualOptional(t, product, pins[x2.ID].ExtractorProductType)
		assert.True(t, pins[x3.ID].ExtractorProductType.IsEmpty())
	})
	t.Run("can list planets with chunking", func(t *testing.T) {
		// given
		old := st.MaxIDsPerQuery
		st.MaxIDsPerQuery = 1
		defer func() {
			st.MaxIDsPerQuery = old
		}()
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		p1 := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{CharacterID: c.ID})
		p2 := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{CharacterID: c.ID})
		p3 := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{CharacterID: c.ID})
		x1 := factory.CreatePlanetPin(storage.CreatePlanetPinParams{CharacterPlanetID: p1.ID})
		x2 := factory.CreatePlanetPin(storage.CreatePlanetPinParams{CharacterPlanetID: p2.ID})
		x3 := factory.CreatePlanetPin(storage.CreatePlanetPinParams{CharacterPlanetID: p3.ID})
		// when
		oo, err := st.ListCharacterPlanets(ctx, c.ID)
		// then
		require.NoError(t, err)
		got := pinIDsByPlanet(oo)
		xassert.Equal(t, set.Of(x1.ID), got[p1.ID])
		xassert.Equal(t, set.Of(x2.ID), got[p2.ID])
		xassert.Equal(t, set.Of(x3.ID), got[p3.ID])
	})
	t.Run("can get planet with pins", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		p1 := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{CharacterID: c.ID})
		p2 := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{CharacterID: c.ID})
		x1 := factory.CreatePlanetPin(storage.CreatePlanetPinParams{CharacterPlanetID: p1.ID})
		x2 := factory.CreatePlanetPin(storage.CreatePlanetPinParams{CharacterPlanetID: p1.ID})
		factory.CreatePlanetPin(storage.CreatePlanetPinParams{CharacterPlanetID: p2.ID})
		// when
		o, err := st.GetCharacterPlanet(ctx, c.ID, p1.EvePlanet.ID)
		// then
		require.NoError(t, err)
		got := pinIDsByPlanet([]*app.CharacterPlanet{o})
		xassert.Equal(t, set.Of(x1.ID, x2.ID), got[p1.ID])
	})
}

func pinIDsByPlanet(planets []*app.CharacterPlanet) map[int64]set.Set[int64] {
	m := make(map[int64]set.Set[int64])
	for _, p := range planets {
		ids := set.Of[int64]()
		for _, x := range p.Pins {
			ids.Add(x.ID)
		}
		m[p.ID] = ids
	}
	return m
}

func TestListCharacterPlanetIDs(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("should return EVE planet IDs of a character's colonies", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		p1 := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{CharacterID: c.ID})
		p2 := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{CharacterID: c.ID})
		factory.CreateCharacterPlanet() // other character
		got, err := st.ListCharacterPlanetIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(p1.EvePlanet.ID, p2.EvePlanet.ID), got)
	})
	t.Run("should return empty set for character without colonies", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		got, err := st.ListCharacterPlanetIDs(ctx, c.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, got.Size())
	})
}

func TestReplaceCharacterPlanet(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	pinIDs := func(p *app.CharacterPlanet) []int64 {
		var s []int64
		for _, x := range p.Pins {
			s = append(s, x.ID)
		}
		return s
	}
	routeIDs := func(p *app.CharacterPlanet) []int64 {
		var s []int64
		for _, x := range p.Routes {
			s = append(s, x.RouteID)
		}
		return s
	}
	t.Run("should create new colony with pins and routes", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		evePlanet := factory.CreateEvePlanet()
		pinType := factory.CreateEveType()
		product := factory.CreateEveType()
		lastUpdate := time.Now().UTC()
		_, err := st.ReplaceCharacterPlanet(ctx, storage.ReplaceCharacterPlanetParams{
			CharacterID:  c.ID,
			EvePlanetID:  evePlanet.ID,
			LastUpdate:   lastUpdate,
			UpgradeLevel: 3,
			Pins: []storage.CreatePlanetPinParams{
				{PinID: 1, TypeID: pinType.ID, Contents: map[int64]int64{product.ID: 42}},
				{PinID: 2, TypeID: pinType.ID},
			},
			Routes: []storage.CreatePlanetRouteParams{
				{RouteID: 7, SourcePinID: 1, DestinationPinID: 2, ContentTypeID: product.ID, Quantity: 100},
			},
		})
		require.NoError(t, err)
		p, err := st.GetCharacterPlanet(ctx, c.ID, evePlanet.ID)
		require.NoError(t, err)
		xassert.Equal(t, c.ID, p.CharacterID)
		xassert.Equal(t, evePlanet, p.EvePlanet)
		xassert.Equal(t, lastUpdate, p.LastUpdate)
		xassert.Equal(t, 3, p.UpgradeLevel)
		assert.ElementsMatch(t, []int64{1, 2}, pinIDs(p))
		assert.Equal(t, []int64{7}, routeIDs(p))
		for _, x := range p.Pins {
			if x.ID == 1 {
				require.Len(t, x.Contents, 1)
				xassert.Equal(t, 42, x.Contents[0].Amount)
			}
		}
	})
	t.Run("should replace pins and routes of existing colony", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		lastNotified := time.Now().Add(-5 * time.Minute).UTC()
		cp := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{
			LastUpdate:   time.Now().Add(-time.Hour).UTC(),
			LastNotified: lastNotified,
		})
		factory.CreatePlanetPin(storage.CreatePlanetPinParams{CharacterPlanetID: cp.ID, PinID: 1})
		factory.CreatePlanetRoute(storage.CreatePlanetRouteParams{CharacterPlanetID: cp.ID, RouteID: 1, SourcePinID: 1, DestinationPinID: 1})
		pinType := factory.CreateEveType()
		lastUpdate := time.Now().UTC()
		id, err := st.ReplaceCharacterPlanet(ctx, storage.ReplaceCharacterPlanetParams{
			CharacterID:  cp.CharacterID,
			EvePlanetID:  cp.EvePlanet.ID,
			LastUpdate:   lastUpdate,
			UpgradeLevel: 4,
			Pins:         []storage.CreatePlanetPinParams{{PinID: 2, TypeID: pinType.ID}},
		})
		require.NoError(t, err)
		xassert.Equal(t, cp.ID, id)
		p, err := st.GetCharacterPlanet(ctx, cp.CharacterID, cp.EvePlanet.ID)
		require.NoError(t, err)
		xassert.Equal(t, lastUpdate, p.LastUpdate)
		xassert.EqualOptional(t, lastNotified, p.LastNotified)
		xassert.Equal(t, 4, p.UpgradeLevel)
		assert.Equal(t, []int64{2}, pinIDs(p))
		assert.Empty(t, p.Routes)
	})
	t.Run("should keep existing colony unchanged when replacing fails", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		oldUpdate := time.Now().Add(-time.Hour).UTC()
		cp := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{LastUpdate: oldUpdate})
		factory.CreatePlanetPin(storage.CreatePlanetPinParams{CharacterPlanetID: cp.ID, PinID: 1})
		factory.CreatePlanetRoute(storage.CreatePlanetRouteParams{CharacterPlanetID: cp.ID, RouteID: 1, SourcePinID: 1, DestinationPinID: 1})
		pinType := factory.CreateEveType()
		_, err := st.ReplaceCharacterPlanet(ctx, storage.ReplaceCharacterPlanetParams{
			CharacterID: cp.CharacterID,
			EvePlanetID: cp.EvePlanet.ID,
			LastUpdate:  time.Now().UTC(),
			Pins: []storage.CreatePlanetPinParams{
				{PinID: 2, TypeID: pinType.ID},
				{PinID: 3}, // invalid
			},
		})
		require.ErrorIs(t, err, app.ErrInvalid)
		p, err := st.GetCharacterPlanet(ctx, cp.CharacterID, cp.EvePlanet.ID)
		require.NoError(t, err)
		xassert.Equal(t, oldUpdate, p.LastUpdate)
		assert.Equal(t, []int64{1}, pinIDs(p))
		assert.Equal(t, []int64{1}, routeIDs(p))
	})
}

func TestCharacterPlanet_ConsistentReads(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ctx := context.Background()
	t.Run("should return not found for missing colony", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		_, err := st.GetCharacterPlanet(ctx, c.ID, 42)
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
	t.Run("should return empty list for character without colonies", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		oo, err := st.ListCharacterPlanets(ctx, c.ID)
		require.NoError(t, err)
		assert.Empty(t, oo)
	})
	t.Run("should never mix old and new data of a colony while it is written", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		old := st.MaxIDsPerQuery
		st.MaxIDsPerQuery = 1 // one statement per pin, so a mixed read has more chances
		defer func() {
			st.MaxIDsPerQuery = old
		}()
		pinType := factory.CreateEveType()
		contentType := factory.CreateEveType()
		t0 := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
		type version struct {
			lastUpdate time.Time
			pinIDs     set.Set[int64]
			amount     int64
			routeID    int64
		}
		versionA := version{lastUpdate: t0, pinIDs: set.Of[int64](1, 2), amount: 100, routeID: 1}
		versionB := version{lastUpdate: t0.Add(time.Hour), pinIDs: set.Of[int64](3, 4), amount: 200, routeID: 2}
		replace := func(characterID, planetID int64, v version) error {
			arg := storage.ReplaceCharacterPlanetParams{
				CharacterID:  characterID,
				EvePlanetID:  planetID,
				LastUpdate:   v.lastUpdate,
				UpgradeLevel: 1,
			}
			for id := range v.pinIDs.All() {
				arg.Pins = append(arg.Pins, storage.CreatePlanetPinParams{
					PinID:    id,
					TypeID:   pinType.ID,
					Contents: map[int64]int64{contentType.ID: v.amount},
				})
			}
			pins := slices.Sorted(v.pinIDs.All())
			arg.Routes = append(arg.Routes, storage.CreatePlanetRouteParams{
				ContentTypeID:    contentType.ID,
				DestinationPinID: pins[1],
				Quantity:         10,
				RouteID:          v.routeID,
				SourcePinID:      pins[0],
			})
			_, err := st.ReplaceCharacterPlanet(ctx, arg)
			return err
		}
		c1 := factory.CreateCharacter()
		c2 := factory.CreateCharacter()
		p1 := factory.CreateEvePlanet().ID // changes
		p2 := factory.CreateEvePlanet().ID // never changes
		p3 := factory.CreateEvePlanet().ID // changes, other character
		require.NoError(t, replace(c1.ID, p1, versionA))
		require.NoError(t, replace(c1.ID, p2, versionA))
		require.NoError(t, replace(c2.ID, p3, versionA))

		// cycle the changing colonies through update, delete and re-create
		done := make(chan struct{})
		writerErr := make(chan error, 1)
		go func() {
			defer close(writerErr)
			for {
				for _, step := range []func(characterID, planetID int64) error{
					func(c, p int64) error { return replace(c, p, versionB) },
					func(c, p int64) error { return replace(c, p, versionA) },
					func(c, p int64) error { return st.DeleteCharacterPlanet(ctx, c, set.Of(p)) },
					func(c, p int64) error { return replace(c, p, versionA) },
				} {
					select {
					case <-done:
						return
					default:
					}
					for _, x := range [][2]int64{{c1.ID, p1}, {c2.ID, p3}} {
						if err := step(x[0], x[1]); err != nil {
							writerErr <- err
							return
						}
					}
				}
			}
		}()

		// assertConsistent reports whether all data of a colony belongs to one version.
		assertConsistent := func(t *testing.T, p *app.CharacterPlanet) bool {
			t.Helper()
			var v version
			switch {
			case p.LastUpdate.Equal(versionA.lastUpdate):
				v = versionA
			case p.LastUpdate.Equal(versionB.lastUpdate):
				v = versionB
			default:
				return assert.Fail(t, "unknown last update", "%v", p.LastUpdate)
			}
			var pinIDs set.Set[int64]
			for _, pin := range p.Pins {
				pinIDs.Add(pin.ID)
				if !assert.Len(t, pin.Contents, 1, "pin %d", pin.ID) || !assert.Equal(t, v.amount, pin.Contents[0].Amount, "pin %d", pin.ID) {
					return false
				}
			}
			if !assert.True(t, v.pinIDs.Equal(pinIDs), "pins %v for last update %v", pinIDs, p.LastUpdate) {
				return false
			}
			return assert.Len(t, p.Routes, 1) && assert.Equal(t, v.routeID, p.Routes[0].RouteID)
		}
		// check checks colonies of a list: the stable one must be present, the changing ones may be absent.
		check := func(t *testing.T, oo []*app.CharacterPlanet, stable int64, changing ...int64) bool {
			t.Helper()
			var hasStable bool
			for _, p := range oo {
				if p.EvePlanet.ID == stable {
					hasStable = true
				} else if !slices.Contains(changing, p.EvePlanet.ID) {
					continue
				}
				if !assertConsistent(t, p) {
					return false
				}
			}
			return assert.True(t, hasStable, "stable colony missing")
		}
		var seenA, seenB, seenDeleted bool
		ok := true
		for i := 0; ok && i < 300; i++ {
			p, err := st.GetCharacterPlanet(ctx, c1.ID, p1)
			if errors.Is(err, app.ErrNotFound) {
				seenDeleted = true
			} else if ok = assert.NoError(t, err) && assertConsistent(t, p); !ok {
				break
			} else if p.LastUpdate.Equal(versionA.lastUpdate) {
				seenA = true
			} else {
				seenB = true
			}
			oo, err := st.ListCharacterPlanets(ctx, c1.ID)
			if ok = assert.NoError(t, err) && check(t, oo, p2, p1); !ok {
				break
			}
			oo, err = st.ListAllCharacterPlanets(ctx)
			ok = assert.NoError(t, err) && check(t, oo, p2, p1, p3)
		}
		close(done)
		assert.NoError(t, <-writerErr)
		assert.True(t, seenA && seenB && seenDeleted, "reads overlapped all writes: A %v, B %v, deleted %v", seenA, seenB, seenDeleted)
	})
}
