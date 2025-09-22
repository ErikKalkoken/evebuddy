package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

// NewDBInMemory creates and returns a database in memory for tests.
// Important: This variant is not suitable for DB code that runs in goroutines.
func NewDBInMemory() (*sql.DB, *storage.Storage, Factory) {
	// in-memory DB for faster running tests
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}
	if err := storage.ApplyMigrations(db); err != nil {
		panic(err)
	}
	r := storage.New(db, db)
	factory := NewFactory(r, db)
	return db, r, factory
}

// NewDBOnDisk creates and returns a new temporary database on disk for tests.
// The database is automatically removed once the tests have concluded.
func NewDBOnDisk(t testing.TB) (*sql.DB, *storage.Storage, Factory) {
	// real DB for more thorough tests
	p := filepath.Join(t.TempDir(), "evebuddy_test.sqlite")
	dbRW, dbRO, err := storage.InitDB("file:" + p)
	if err != nil {
		panic(err)
	}
	r := storage.New(dbRW, dbRO)
	factory := NewFactory(r, dbRO)
	return dbRW, r, factory
}

// func New() (*sql.DB, *storage.Storage, Factory) {

// }

// TruncateTables will purge data from all tables. This is meant for tests.
func TruncateTables(dbRW *sql.DB) {
	_, err := dbRW.Exec("PRAGMA foreign_keys = 0")
	if err != nil {
		panic(err)
	}
	sql := `SELECT name FROM sqlite_master WHERE type = "table"`
	rows, err := dbRW.Query(sql)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			panic(err)
		}
		tables = append(tables, name)
	}
	for _, n := range tables {
		sql := fmt.Sprintf("DELETE FROM %s;", n)
		_, err := dbRW.Exec(sql)
		if err != nil {
			panic(err)
		}
	}
	for _, n := range tables {
		sql := fmt.Sprintf("DELETE FROM SQLITE_SEQUENCE WHERE name='%s'", n)
		_, err := dbRW.Exec(sql)
		if err != nil {
			panic(err)
		}
	}
	_, err = dbRW.Exec("PRAGMA foreign_keys = 1")
	if err != nil {
		panic(err)
	}
}

// ErrGroupDebug represents a replacement for errgroup.Group with the same API,
// but it runs the callbacks without Goroutines, which makes debugging much easier.
type ErrGroupDebug struct {
	ff []func() error
}

func (g *ErrGroupDebug) Go(f func() error) {
	if g.ff == nil {
		g.ff = make([]func() error, 0)
	}
	g.ff = append(g.ff, f)
}

func (g *ErrGroupDebug) Wait() error {
	for _, f := range g.ff {
		err := f()
		if err != nil {
			return err
		}
	}
	return nil
}

type EveUniverseServiceFake struct{}

func (s *EveUniverseServiceFake) GetOrCreateEntityESI(ctx context.Context, id int32) (*app.EveEntity, error) {
	o := &app.EveEntity{
		ID:       id,
		Name:     fmt.Sprintf("Entity%d", id),
		Category: app.EveEntityUnknown,
	}
	return o, nil
}

func (s *EveUniverseServiceFake) GetOrCreateLocationESI(ctx context.Context, id int64) (*app.EveLocation, error) {
	ss, _ := s.GetOrCreateSolarSystemESI(ctx, 30002537)
	owner, _ := s.GetOrCreateEntityESI(ctx, 42)
	o := &app.EveLocation{
		ID:          id,
		Name:        fmt.Sprintf("Location%d", id),
		Owner:       owner,
		SolarSystem: ss,
		UpdatedAt:   time.Now(),
		Type: &app.EveType{
			ID:   35835,
			Name: fmt.Sprintf("Type%d", id),
			Group: &app.EveGroup{
				ID:   1406,
				Name: "Refinery",
				Category: &app.EveCategory{
					ID:   65,
					Name: "Structure",
				},
			},
		},
	}
	return o, nil
}

func (s *EveUniverseServiceFake) GetOrCreateMoonESI(ctx context.Context, id int32) (*app.EveMoon, error) {
	ss, nil := s.GetOrCreateSolarSystemESI(ctx, 30002537)
	o := &app.EveMoon{
		ID:          id,
		Name:        fmt.Sprintf("Moon%d", id),
		SolarSystem: ss,
	}
	return o, nil
}

func (s *EveUniverseServiceFake) GetOrCreatePlanetESI(ctx context.Context, id int32) (*app.EvePlanet, error) {
	ss, _ := s.GetOrCreateSolarSystemESI(ctx, 30002537)
	et, _ := s.GetOrCreateTypeESI(ctx, 5)
	o := &app.EvePlanet{
		ID:          id,
		Name:        fmt.Sprintf("Planet%d", id),
		SolarSystem: ss,
		Type:        et,
	}
	return o, nil
}

func (s *EveUniverseServiceFake) GetOrCreateSolarSystemESI(ctx context.Context, id int32) (*app.EveSolarSystem, error) {
	o := &app.EveSolarSystem{
		ID:   id,
		Name: fmt.Sprintf("System%d", id),
		Constellation: &app.EveConstellation{
			ID:   20000372,
			Name: "Constellation",
			Region: &app.EveRegion{
				ID:   10000030,
				Name: "Region",
			},
		},
	}
	return o, nil
}

func (s *EveUniverseServiceFake) GetOrCreateTypeESI(ctx context.Context, id int32) (*app.EveType, error) {
	o := &app.EveType{
		ID:   id,
		Name: fmt.Sprintf("Type%d", id),
		Group: &app.EveGroup{
			ID:   420,
			Name: "Group",
			Category: &app.EveCategory{
				ID:   5,
				Name: "Category",
			},
		},
	}
	return o, nil
}

func (s *EveUniverseServiceFake) ToEntities(ctx context.Context, ids set.Set[int32]) (map[int32]*app.EveEntity, error) {
	m := make(map[int32]*app.EveEntity)
	for id := range ids.All() {
		o, _ := s.GetOrCreateEntityESI(ctx, id)
		m[id] = o
	}
	return m, nil
}
