package storage_test

import (
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
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

func TestCorporationStructure(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()

	t.Run("can create minimal from scratch", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		system := factory.CreateEveSolarSystem()
		typ := factory.CreateEveType()
		// when
		arg := storage.UpdateOrCreateCorporationStructureParams{
			CorporationID: c.ID,
			Name:          optional.New("Alpha"),
			ProfileID:     42,
			State:         app.StructureStateShieldVulnerable,
			StructureID:   1234,
			SystemID:      system.ID,
			TypeID:        typ.ID,
		}
		err := st.UpdateOrCreateCorporationStructure(t.Context(), arg)
		// then
		require.NoError(t, err)
		got, err := st.GetCorporationStructure(t.Context(), arg.CorporationID, arg.StructureID)
		require.NoError(t, err)
		xassert.Equal(t, c.ID, got.CorporationID)
		xassert.Equal(t, arg.Name, got.Name)
		xassert.Equal(t, arg.ProfileID, got.ProfileID)
		xassert.Equal(t, arg.State, got.State)
		xassert.Equal(t, arg.StructureID, got.StructureID)
		xassert.Equal(t, arg.SystemID, got.System.ID)
		xassert.Equal(t, arg.TypeID, got.Type.ID)
		assert.Empty(t, got.Services)
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
			Name:               optional.New("Alpha"),
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
		err := st.UpdateOrCreateCorporationStructure(t.Context(), arg)
		// then
		require.NoError(t, err)
		got, err := st.GetCorporationStructure(t.Context(), arg.CorporationID, arg.StructureID)
		require.NoError(t, err)
		xassert.Equal(t, c.ID, got.CorporationID)
		xassert.EqualOptional(t, arg.FuelExpires.ValueOrZero(), got.FuelExpires)
		xassert.Equal(t, arg.Name, got.Name)
		xassert.EqualOptional(t, arg.NextReinforceApply.ValueOrZero(), got.NextReinforceApply)
		xassert.EqualOptional(t, arg.NextReinforceHour.ValueOrZero(), got.NextReinforceHour)
		xassert.Equal(t, arg.ProfileID, got.ProfileID)
		xassert.EqualOptional(t, arg.ReinforceHour.ValueOrZero(), got.ReinforceHour)
		xassert.Equal(t, arg.State, got.State)
		xassert.EqualOptional(t, arg.StateTimerEnd.ValueOrZero(), got.StateTimerEnd)
		xassert.EqualOptional(t, arg.StateTimerStart.ValueOrZero(), got.StateTimerStart)
		xassert.Equal(t, arg.StructureID, got.StructureID)
		xassert.Equal(t, arg.SystemID, got.System.ID)
		xassert.Equal(t, arg.TypeID, got.Type.ID)
		xassert.EqualOptional(t, arg.UnanchorsAt.ValueOrZero(), got.UnanchorsAt)
		xassert.Equal(t, "Jupiter", got.Services[0].Name)
		assert.EqualExportedValues(t, app.StructureServiceStateOnline, got.Services[0].State)
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
			Name:               optional.New("Alpha"),
			NextReinforceApply: optional.New(factory.RandomTime()),
			NextReinforceHour:  optional.New(int64(8)),
			ProfileID:          x1.ProfileID + 1,
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
		err := st.UpdateOrCreateCorporationStructure(t.Context(), arg)
		// then
		require.NoError(t, err)
		got, err := st.GetCorporationStructure(t.Context(), x1.CorporationID, x1.StructureID)
		require.NoError(t, err)
		xassert.Equal(t, c.ID, got.CorporationID)
		xassert.EqualOptional(t, arg.FuelExpires.ValueOrZero(), got.FuelExpires)
		xassert.Equal(t, arg.Name, got.Name)
		xassert.EqualOptional(t, arg.NextReinforceApply.ValueOrZero(), got.NextReinforceApply)
		xassert.EqualOptional(t, arg.NextReinforceHour.ValueOrZero(), got.NextReinforceHour)
		xassert.Equal(t, arg.ProfileID, got.ProfileID)
		xassert.EqualOptional(t, arg.ReinforceHour.ValueOrZero(), got.ReinforceHour)
		xassert.Equal(t, arg.State, got.State)
		xassert.EqualOptional(t, arg.StateTimerEnd.ValueOrZero(), got.StateTimerEnd)
		xassert.EqualOptional(t, arg.StateTimerStart.ValueOrZero(), got.StateTimerStart)
		xassert.EqualOptional(t, arg.UnanchorsAt.ValueOrZero(), got.UnanchorsAt)
		xassert.Equal(t, "Jupiter", got.Services[0].Name)
		assert.EqualExportedValues(t, app.StructureServiceStateOnline, got.Services[0].State)
	})

	t.Run("should keep existing structure unchanged when update fails", func(t *testing.T) {
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
		err := st.UpdateOrCreateCorporationStructure(t.Context(), storage.UpdateOrCreateCorporationStructureParams{
			CorporationID: c.ID,
			ProfileID:     x1.ProfileID,
			State:         app.StructureStateShieldVulnerable,
			StructureID:   x1.StructureID,
			SystemID:      x1.System.ID,
			TypeID:        x1.Type.ID,
			Services: []storage.StructureServiceParams{
				{
					Name:  "Jupiter",
					State: app.StructureServiceStateOnline,
				},
				{
					State: app.StructureServiceStateOnline,
				}, // invalid
			},
		})
		// then
		require.ErrorIs(t, err, app.ErrInvalid)
		got, err := st.GetCorporationStructure(t.Context(), x1.CorporationID, x1.StructureID)
		require.NoError(t, err)
		xassert.Equal(t, app.StructureStateAnchorVulnerable, got.State)
		services := serviceNamesByStructure([]*app.CorporationStructure{got})
		xassert.Equal(t, set.Of("Venus"), services[got.ID])
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
		got, err := st.ListCorporationStructureIDs(t.Context(), c.ID)
		// then
		require.NoError(t, err)
		want := set.Of(o1.StructureID, o2.StructureID)
		xassert.Equal(t, want, got)
	})

	t.Run("can list all structures", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		o1 := factory.CreateCorporationStructure()
		o2 := factory.CreateCorporationStructure()
		o3 := factory.CreateCorporationStructure()
		s1 := factory.CreateStructureService(storage.CreateStructureServiceParams{CorporationStructureID: o1.ID})
		s2 := factory.CreateStructureService(storage.CreateStructureServiceParams{CorporationStructureID: o1.ID})
		s3 := factory.CreateStructureService(storage.CreateStructureServiceParams{CorporationStructureID: o3.ID})
		// when
		xx, err := st.ListAllCorporationStructures(t.Context())
		// then
		require.NoError(t, err)
		got := set.Collect(xiter.MapSlice(xx, func(x *app.CorporationStructure) int64 {
			return x.StructureID
		}))
		want := set.Of(o1.StructureID, o2.StructureID, o3.StructureID)
		xassert.Equal(t, want, got)
		services := serviceNamesByStructure(xx)
		xassert.Equal(t, set.Of(s1.Name, s2.Name), services[o1.ID])
		xassert.Equal(t, set.Of[string](), services[o2.ID])
		xassert.Equal(t, set.Of(s3.Name), services[o3.ID])
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
		o3 := factory.CreateCorporationStructure()
		s1 := factory.CreateStructureService(storage.CreateStructureServiceParams{CorporationStructureID: o1.ID})
		factory.CreateStructureService(storage.CreateStructureServiceParams{CorporationStructureID: o3.ID})
		// when
		xx, err := st.ListCorporationStructures(t.Context(), c.ID)
		// then
		require.NoError(t, err)
		got := set.Collect(xiter.MapSlice(xx, func(x *app.CorporationStructure) int64 {
			return x.StructureID
		}))
		want := set.Of(o1.StructureID, o2.StructureID)
		xassert.Equal(t, want, got)
		services := serviceNamesByStructure(xx)
		xassert.Equal(t, set.Of(s1.Name), services[o1.ID])
		xassert.Equal(t, set.Of[string](), services[o2.ID])
	})

	t.Run("can list structures with chunking", func(t *testing.T) {
		// given
		old := st.MaxIDsPerQuery
		st.MaxIDsPerQuery = 1
		defer func() {
			st.MaxIDsPerQuery = old
		}()
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		arg := storage.UpdateOrCreateCorporationStructureParams{CorporationID: c.ID}
		o1 := factory.CreateCorporationStructure(arg)
		o2 := factory.CreateCorporationStructure(arg)
		o3 := factory.CreateCorporationStructure(arg)
		s1 := factory.CreateStructureService(storage.CreateStructureServiceParams{CorporationStructureID: o1.ID})
		s2 := factory.CreateStructureService(storage.CreateStructureServiceParams{CorporationStructureID: o2.ID})
		s3 := factory.CreateStructureService(storage.CreateStructureServiceParams{CorporationStructureID: o3.ID})
		// when
		xx, err := st.ListCorporationStructures(t.Context(), c.ID)
		// then
		require.NoError(t, err)
		services := serviceNamesByStructure(xx)
		xassert.Equal(t, set.Of(s1.Name), services[o1.ID])
		xassert.Equal(t, set.Of(s2.Name), services[o2.ID])
		xassert.Equal(t, set.Of(s3.Name), services[o3.ID])
	})

	t.Run("can get structure with services", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		arg := storage.UpdateOrCreateCorporationStructureParams{CorporationID: c.ID}
		o1 := factory.CreateCorporationStructure(arg)
		o2 := factory.CreateCorporationStructure(arg)
		s1 := factory.CreateStructureService(storage.CreateStructureServiceParams{CorporationStructureID: o1.ID})
		s2 := factory.CreateStructureService(storage.CreateStructureServiceParams{CorporationStructureID: o1.ID})
		factory.CreateStructureService(storage.CreateStructureServiceParams{CorporationStructureID: o2.ID})
		// when
		x, err := st.GetCorporationStructure(t.Context(), c.ID, o1.StructureID)
		// then
		require.NoError(t, err)
		services := serviceNamesByStructure([]*app.CorporationStructure{x})
		xassert.Equal(t, set.Of(s1.Name, s2.Name), services[o1.ID])
	})

	t.Run("can delete all structures of a corporation with their services", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateCorporation()
		c2 := factory.CreateCorporation()
		o1 := factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{
			CorporationID: c1.ID,
			Services: []storage.StructureServiceParams{
				{Name: "Jupiter", State: app.StructureServiceStateOnline},
			},
		})
		factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{
			CorporationID: c1.ID,
		})
		o3 := factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{
			CorporationID: c2.ID,
		})
		// when
		err := st.DeleteCorporationStructures(t.Context(), c1.ID)
		// then
		require.NoError(t, err)
		got1, err := st.ListCorporationStructureIDs(t.Context(), c1.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, got1.Size())
		services, err := st.ListStructureServices(t.Context(), o1.ID)
		require.NoError(t, err)
		assert.Empty(t, services)
		got2, err := st.ListCorporationStructureIDs(t.Context(), c2.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(o3.StructureID), got2)
	})

	t.Run("can count structures with state", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{
			CorporationID: c.ID,
			State:         app.StructureStateArmorReinforce,
		})
		factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{
			CorporationID: c.ID,
			State:         app.StructureStateHullReinforce,
		})
		factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{
			CorporationID: c.ID,
			State:         app.StructureStateShieldVulnerable,
		})
		// structure for a different corporation with a matching state must not be counted
		factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{
			State: app.StructureStateArmorReinforce,
		})
		// when
		got, err := st.CountCorporationStructuresWithState(t.Context(), c.ID, set.Of(app.StructureStateArmorReinforce, app.StructureStateHullReinforce))
		// then
		require.NoError(t, err)
		assert.Equal(t, int64(2), got)
	})

	t.Run("count returns error for empty states", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		// when
		_, err := st.CountCorporationStructuresWithState(t.Context(), c.ID, set.Set[app.StructureState]{})
		// then
		assert.ErrorIs(t, err, app.ErrInvalid)
	})
}

func TestStructureService(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()

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
		err := st.CreateStructureService(t.Context(), arg)
		// then
		require.NoError(t, err)
		got, err := st.GetStructureService(t.Context(), structure.ID, "Alpha")
		require.NoError(t, err)
		xassert.Equal(t, "Alpha", got.Name)
		xassert.Equal(t, app.StructureServiceStateOnline, got.State)
	})

	t.Run("can list services", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		s := factory.CreateCorporationStructure()
		x1 := factory.CreateStructureService(storage.CreateStructureServiceParams{CorporationStructureID: s.ID})
		x2 := factory.CreateStructureService(storage.CreateStructureServiceParams{CorporationStructureID: s.ID})
		// when
		oo, err := st.ListStructureServices(t.Context(), s.ID)
		// then
		require.NoError(t, err)
		got := set.Collect(xiter.MapSlice(oo, func(x *app.StructureService) string {
			return x.Name
		}))
		want := set.Of(x1.Name, x2.Name)
		xassert.Equal(t, want, got)
	})

	t.Run("returns not found error", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		_, err := st.GetStructureService(t.Context(), 1, "Alpha")
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
}

func serviceNamesByStructure(structures []*app.CorporationStructure) map[int64]set.Set[string] {
	m := make(map[int64]set.Set[string])
	for _, s := range structures {
		names := set.Of[string]()
		for _, x := range s.Services {
			names.Add(x.Name)
		}
		m[s.ID] = names
	}
	return m
}

func TestCorporationStructure_ConsistentReads(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	t.Run("should never mix old and new data of a structure while it is written", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		old := st.MaxIDsPerQuery
		st.MaxIDsPerQuery = 1 // one statement per structure, so a mixed read has more chances
		defer func() {
			st.MaxIDsPerQuery = old
		}()
		type version struct {
			state    app.StructureState
			services set.Set[string]
		}
		versionA := version{state: app.StructureStateAnchorVulnerable, services: set.Of("Alpha", "Bravo")}
		versionB := version{state: app.StructureStateShieldVulnerable, services: set.Of("Charlie", "Delta")}
		c1 := factory.CreateCorporation()
		c2 := factory.CreateCorporation()
		s1 := factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{CorporationID: c1.ID}) // changes
		s2 := factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{CorporationID: c1.ID}) // never changes
		s3 := factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{CorporationID: c2.ID}) // changes, other corporation
		params := func(s *app.CorporationStructure, v version) storage.CorporationStructureParams {
			arg := storage.CorporationStructureParams{
				ProfileID:   s.ProfileID,
				State:       v.state,
				StructureID: s.StructureID,
				SystemID:    s.System.ID,
				TypeID:      s.Type.ID,
			}
			for name := range v.services.All() {
				arg.Services = append(arg.Services, storage.StructureServiceParams{
					Name:  name,
					State: app.StructureServiceStateOnline,
				})
			}
			return arg
		}
		replace := func(s *app.CorporationStructure, v version) error {
			arg := params(s, v)
			return st.UpdateOrCreateCorporationStructure(t.Context(), storage.UpdateOrCreateCorporationStructureParams{
				CorporationID: s.CorporationID,
				ProfileID:     arg.ProfileID,
				Services:      arg.Services,
				State:         arg.State,
				StructureID:   arg.StructureID,
				SystemID:      arg.SystemID,
				TypeID:        arg.TypeID,
			})
		}
		require.NoError(t, replace(s1, versionA))
		require.NoError(t, replace(s2, versionA))
		require.NoError(t, replace(s3, versionA))

		// cycle the changing structures through update, delete and re-create
		done := make(chan struct{})
		writerErr := make(chan error, 1)
		go func() {
			defer close(writerErr)
			for {
				for _, step := range []func(s *app.CorporationStructure) error{
					func(s *app.CorporationStructure) error { return replace(s, versionB) },
					func(s *app.CorporationStructure) error { return replace(s, versionA) },
					func(s *app.CorporationStructure) error {
						// delete s, but keep the stable structure
						var args []storage.CorporationStructureParams
						if s.CorporationID == s2.CorporationID {
							args = append(args, params(s2, versionA))
						}
						_, err := st.ReplaceCorporationStructures(t.Context(), s.CorporationID, args)
						return err
					},
					func(s *app.CorporationStructure) error { return replace(s, versionA) },
				} {
					select {
					case <-done:
						return
					default:
					}
					for _, s := range []*app.CorporationStructure{s1, s3} {
						if err := step(s); err != nil {
							writerErr <- err
							return
						}
					}
				}
			}
		}()

		// assertConsistent reports whether all data of a structure belongs to one version.
		assertConsistent := func(t *testing.T, s *app.CorporationStructure) bool {
			t.Helper()
			var v version
			switch s.State {
			case versionA.state:
				v = versionA
			case versionB.state:
				v = versionB
			default:
				return assert.Fail(t, "unknown state", "%v", s.State)
			}
			got := set.Collect(xiter.MapSlice(s.Services, func(x *app.StructureService) string {
				return x.Name
			}))
			return assert.True(t, v.services.Equal(got), "services %v for state %v", got, s.State)
		}
		// check checks structures of a list: the stable one must be present, the changing ones may be absent.
		check := func(t *testing.T, oo []*app.CorporationStructure, stable int64, changing ...int64) bool {
			t.Helper()
			var hasStable bool
			for _, s := range oo {
				if s.StructureID == stable {
					hasStable = true
				} else if !slices.Contains(changing, s.StructureID) {
					continue
				}
				if !assertConsistent(t, s) {
					return false
				}
			}
			return assert.True(t, hasStable, "stable structure missing")
		}
		var seenA, seenB, seenDeleted bool
		ok := true
		// Read until all versions were seen, so the test does not depend on scheduling.
		deadline := time.Now().Add(10 * time.Second)
		for i := 0; ok && (i < 300 || !(seenA && seenB && seenDeleted)) && time.Now().Before(deadline); i++ {
			s, err := st.GetCorporationStructure(t.Context(), c1.ID, s1.StructureID)
			if errors.Is(err, app.ErrNotFound) {
				seenDeleted = true
			} else if ok = assert.NoError(t, err) && assertConsistent(t, s); !ok {
				break
			} else if s.State == versionA.state {
				seenA = true
			} else {
				seenB = true
			}
			oo, err := st.ListCorporationStructures(t.Context(), c1.ID)
			if ok = assert.NoError(t, err) && check(t, oo, s2.StructureID, s1.StructureID); !ok {
				break
			}
			oo, err = st.ListAllCorporationStructures(t.Context())
			ok = assert.NoError(t, err) && check(t, oo, s2.StructureID, s1.StructureID, s3.StructureID)
		}
		close(done)
		assert.NoError(t, <-writerErr)
		assert.True(t, seenA && seenB && seenDeleted, "reads overlapped all writes: A %v, B %v, deleted %v", seenA, seenB, seenDeleted)
	})
}

func TestReplaceCorporationStructures(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()

	t.Run("should create new structures with services", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		system := factory.CreateEveSolarSystem()
		typ := factory.CreateEveType()
		arg := storage.CorporationStructureParams{
			Name:        optional.New("Alpha"),
			ProfileID:   42,
			State:       app.StructureStateShieldVulnerable,
			StructureID: 1234,
			SystemID:    system.ID,
			TypeID:      typ.ID,
			Services: []storage.StructureServiceParams{
				{Name: "Jupiter", State: app.StructureServiceStateOnline},
			},
		}
		// when
		changed, err := st.ReplaceCorporationStructures(t.Context(), c.ID, []storage.CorporationStructureParams{arg})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		got, err := st.GetCorporationStructure(t.Context(), c.ID, 1234)
		require.NoError(t, err)
		xassert.Equal(t, arg.Name, got.Name)
		xassert.Equal(t, 42, got.ProfileID)
		xassert.Equal(t, app.StructureStateShieldVulnerable, got.State)
		xassert.Equal(t, system.ID, got.System.ID)
		xassert.Equal(t, typ.ID, got.Type.ID)
		require.Len(t, got.Services, 1)
		xassert.Equal(t, "Jupiter", got.Services[0].Name)
		xassert.Equal(t, app.StructureServiceStateOnline, got.Services[0].State)
	})

	t.Run("should report no change when structures are unchanged", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		system := factory.CreateEveSolarSystem()
		typ := factory.CreateEveType()
		now := time.Now().In(time.FixedZone("test", 3600)) // differs from UTC after the DB round trip
		arg := storage.CorporationStructureParams{
			FuelExpires:        optional.New(now.Add(48 * time.Hour)),
			Name:               optional.New("Alpha"),
			NextReinforceApply: optional.New(now.Add(24 * time.Hour)),
			NextReinforceHour:  optional.New[int64](8),
			ProfileID:          42,
			ReinforceHour:      optional.New[int64](12),
			State:              app.StructureStateArmorReinforce,
			StateTimerEnd:      optional.New(now.Add(time.Hour)),
			StateTimerStart:    optional.New(now.Add(-time.Hour)),
			StructureID:        1234,
			SystemID:           system.ID,
			TypeID:             typ.ID,
			UnanchorsAt:        optional.New(now.Add(72 * time.Hour)),
			Services: []storage.StructureServiceParams{
				{Name: "Jupiter", State: app.StructureServiceStateOnline},
				{Name: "Venus", State: app.StructureServiceStateOffline},
			},
		}
		_, err := st.ReplaceCorporationStructures(t.Context(), c.ID, []storage.CorporationStructureParams{arg})
		require.NoError(t, err)
		slices.Reverse(arg.Services) // order of services does not matter
		// when
		changed, err := st.ReplaceCorporationStructures(t.Context(), c.ID, []storage.CorporationStructureParams{arg})
		// then
		require.NoError(t, err)
		assert.False(t, changed)
	})

	t.Run("should store a structure when a changeable column has changed", func(t *testing.T) {
		now := time.Now().UTC()
		cases := []struct {
			name   string
			change func(arg *storage.CorporationStructureParams)
		}{
			{"fuel expires", func(arg *storage.CorporationStructureParams) {
				arg.FuelExpires = optional.New(now.Add(time.Hour))
			}},
			{"name", func(arg *storage.CorporationStructureParams) {
				arg.Name = optional.New("Bravo")
			}},
			{"next reinforce apply", func(arg *storage.CorporationStructureParams) {
				arg.NextReinforceApply = optional.New(now.Add(time.Hour))
			}},
			{"next reinforce hour", func(arg *storage.CorporationStructureParams) {
				arg.NextReinforceHour = optional.New[int64](3)
			}},
			{"profile ID", func(arg *storage.CorporationStructureParams) {
				arg.ProfileID = 43
			}},
			{"reinforce hour", func(arg *storage.CorporationStructureParams) {
				arg.ReinforceHour = optional.New[int64](3)
			}},
			{"state", func(arg *storage.CorporationStructureParams) {
				arg.State = app.StructureStateHullReinforce
			}},
			{"state timer end", func(arg *storage.CorporationStructureParams) {
				arg.StateTimerEnd = optional.New(now.Add(time.Hour))
			}},
			{"state timer start", func(arg *storage.CorporationStructureParams) {
				arg.StateTimerStart = optional.New(now.Add(-time.Hour))
			}},
			{"unanchors at", func(arg *storage.CorporationStructureParams) {
				arg.UnanchorsAt = optional.New(now.Add(time.Hour))
			}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// given
				testutil.MustTruncateTables(db)
				c := factory.CreateCorporation()
				system := factory.CreateEveSolarSystem()
				typ := factory.CreateEveType()
				arg := storage.CorporationStructureParams{
					Name:        optional.New("Alpha"),
					ProfileID:   42,
					State:       app.StructureStateShieldVulnerable,
					StructureID: 1234,
					SystemID:    system.ID,
					TypeID:      typ.ID,
				}
				_, err := st.ReplaceCorporationStructures(t.Context(), c.ID, []storage.CorporationStructureParams{arg})
				require.NoError(t, err)
				tc.change(&arg)
				// when
				changed, err := st.ReplaceCorporationStructures(t.Context(), c.ID, []storage.CorporationStructureParams{arg})
				// then
				require.NoError(t, err)
				assert.True(t, changed)
				got, err := st.GetCorporationStructure(t.Context(), c.ID, 1234)
				require.NoError(t, err)
				xassert.Equal(t, arg.FuelExpires, got.FuelExpires)
				xassert.Equal(t, arg.Name, got.Name)
				xassert.Equal(t, arg.NextReinforceApply, got.NextReinforceApply)
				xassert.Equal(t, arg.NextReinforceHour, got.NextReinforceHour)
				xassert.Equal(t, arg.ProfileID, got.ProfileID)
				xassert.Equal(t, arg.ReinforceHour, got.ReinforceHour)
				xassert.Equal(t, arg.State, got.State)
				xassert.Equal(t, arg.StateTimerEnd, got.StateTimerEnd)
				xassert.Equal(t, arg.StateTimerStart, got.StateTimerStart)
				xassert.Equal(t, arg.UnanchorsAt, got.UnanchorsAt)
			})
		}
	})

	t.Run("should store a structure when its services have changed", func(t *testing.T) {
		cases := []struct {
			name     string
			services []storage.StructureServiceParams
		}{
			{"service added", []storage.StructureServiceParams{
				{Name: "Jupiter", State: app.StructureServiceStateOnline},
				{Name: "Venus", State: app.StructureServiceStateOnline},
			}},
			{"service removed", nil},
			{"service state changed", []storage.StructureServiceParams{
				{Name: "Jupiter", State: app.StructureServiceStateOffline},
			}},
			{"service renamed", []storage.StructureServiceParams{
				{Name: "Venus", State: app.StructureServiceStateOnline},
			}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// given
				testutil.MustTruncateTables(db)
				c := factory.CreateCorporation()
				system := factory.CreateEveSolarSystem()
				typ := factory.CreateEveType()
				arg := storage.CorporationStructureParams{
					ProfileID:   42,
					State:       app.StructureStateShieldVulnerable,
					StructureID: 1234,
					SystemID:    system.ID,
					TypeID:      typ.ID,
					Services: []storage.StructureServiceParams{
						{Name: "Jupiter", State: app.StructureServiceStateOnline},
					},
				}
				_, err := st.ReplaceCorporationStructures(t.Context(), c.ID, []storage.CorporationStructureParams{arg})
				require.NoError(t, err)
				arg.Services = tc.services
				// when
				changed, err := st.ReplaceCorporationStructures(t.Context(), c.ID, []storage.CorporationStructureParams{arg})
				// then
				require.NoError(t, err)
				assert.True(t, changed)
				got, err := st.GetCorporationStructure(t.Context(), c.ID, 1234)
				require.NoError(t, err)
				want := make(map[string]app.StructureServiceState)
				for _, s := range tc.services {
					want[s.Name] = s.State
				}
				services := make(map[string]app.StructureServiceState)
				for _, s := range got.Services {
					services[s.Name] = s.State
				}
				xassert.Equal(t, want, services)
			})
		}
	})

	t.Run("should ignore changes to columns which can not change", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		system := factory.CreateEveSolarSystem()
		typ := factory.CreateEveType()
		otherType := factory.CreateEveType()
		arg := storage.CorporationStructureParams{
			ProfileID:   42,
			State:       app.StructureStateShieldVulnerable,
			StructureID: 1234,
			SystemID:    system.ID,
			TypeID:      typ.ID,
		}
		_, err := st.ReplaceCorporationStructures(t.Context(), c.ID, []storage.CorporationStructureParams{arg})
		require.NoError(t, err)
		arg.TypeID = otherType.ID
		// when
		changed, err := st.ReplaceCorporationStructures(t.Context(), c.ID, []storage.CorporationStructureParams{arg})
		// then
		require.NoError(t, err)
		assert.False(t, changed)
		got, err := st.GetCorporationStructure(t.Context(), c.ID, 1234)
		require.NoError(t, err)
		xassert.Equal(t, typ.ID, got.Type.ID)
	})

	t.Run("should delete structures missing in args with their services", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		kept := factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{
			CorporationID: c.ID,
		})
		vanished := factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{
			CorporationID: c.ID,
			Services: []storage.StructureServiceParams{
				{Name: "Jupiter", State: app.StructureServiceStateOnline},
			},
		})
		arg := storage.CorporationStructureParams{
			FuelExpires:   kept.FuelExpires,
			Name:          kept.Name,
			ProfileID:     kept.ProfileID,
			ReinforceHour: kept.ReinforceHour,
			State:         kept.State,
			StructureID:   kept.StructureID,
			SystemID:      kept.System.ID,
			TypeID:        kept.Type.ID,
		}
		// when
		changed, err := st.ReplaceCorporationStructures(t.Context(), c.ID, []storage.CorporationStructureParams{arg})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		got, err := st.ListCorporationStructureIDs(t.Context(), c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(kept.StructureID), got)
		services, err := st.ListStructureServices(t.Context(), vanished.ID)
		require.NoError(t, err)
		assert.Empty(t, services)
	})

	t.Run("should report no change when there are no structures", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		// when
		changed, err := st.ReplaceCorporationStructures(t.Context(), c.ID, nil)
		// then
		require.NoError(t, err)
		assert.False(t, changed)
	})

	t.Run("should not touch structures of other corporations", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateCorporation()
		c2 := factory.CreateCorporation()
		other := factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{
			CorporationID: c2.ID,
		})
		// when
		changed, err := st.ReplaceCorporationStructures(t.Context(), c1.ID, nil)
		// then
		require.NoError(t, err)
		assert.False(t, changed)
		got, err := st.ListCorporationStructureIDs(t.Context(), c2.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(other.StructureID), got)
	})

	t.Run("should keep structures unchanged when replace fails", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		existing := factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{
			CorporationID: c.ID,
		})
		system := factory.CreateEveSolarSystem()
		arg := storage.CorporationStructureParams{
			ProfileID:   42,
			State:       app.StructureStateShieldVulnerable,
			StructureID: existing.StructureID + 1,
			SystemID:    system.ID,
			TypeID:      42, // has no eve type
		}
		// when
		_, err := st.ReplaceCorporationStructures(t.Context(), c.ID, []storage.CorporationStructureParams{arg})
		// then
		require.Error(t, err)
		got, err := st.ListCorporationStructureIDs(t.Context(), c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(existing.StructureID), got)
	})

	t.Run("should return error when args are invalid", func(t *testing.T) {
		cases := []struct {
			name          string
			corporationID int64
			args          []storage.CorporationStructureParams
		}{
			{"zero corporation ID", 0, nil},
			{"zero structure ID", 1, []storage.CorporationStructureParams{{
				State:    app.StructureStateShieldVulnerable,
				SystemID: 2,
				TypeID:   3,
			}}},
			{"service without name", 1, []storage.CorporationStructureParams{{
				State:       app.StructureStateShieldVulnerable,
				StructureID: 4,
				SystemID:    2,
				TypeID:      3,
				Services: []storage.StructureServiceParams{
					{State: app.StructureServiceStateOnline},
				},
			}}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// when
				_, err := st.ReplaceCorporationStructures(t.Context(), tc.corporationID, tc.args)
				// then
				assert.ErrorIs(t, err, app.ErrInvalid)
			})
		}
	})
}
