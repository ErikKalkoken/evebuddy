package migrate

import (
	"database/sql"
	"errors"
	"io/fs"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestApplyNewMigrations(t *testing.T) {
	migrations := fstest.MapFS{
		"migrations/0001_alpha.sql": &fstest.MapFile{
			Data: []byte("CREATE TABLE alpha(id INTEGER NOT NULL);"),
		},
		"migrations/0002_bravo.sql": &fstest.MapFile{
			Data: []byte("CREATE TABLE bravo(id INTEGER NOT NULL);"),
		},
	}
	t.Run("should apply all migrations from scratch", func(t *testing.T) {
		// given
		db := CreateTestDB(t)
		require.NoError(t, createMigrationTracking(db))
		// when
		err := applyNewMigrations(db, migrations)
		// then
		require.NoError(t, err)
		applied, err := listMigrationNames(db)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"0001_alpha", "0002_bravo"}, applied)
		tables, err := ListTableNames(db)
		require.NoError(t, err)
		names := set.Of(tables...)
		assert.True(t, names.Contains("alpha"))
		assert.True(t, names.Contains("bravo"))
	})
	t.Run("should apply new migrations only", func(t *testing.T) {
		// given
		db := CreateTestDB(t)
		require.NoError(t, createMigrationTracking(db))
		require.NoError(t, recordMigration(db, "0001_alpha"))
		// when
		err := applyNewMigrations(db, migrations)
		// then
		require.NoError(t, err)
		applied, err := listMigrationNames(db)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"0001_alpha", "0002_bravo"}, applied)
		tables, err := ListTableNames(db)
		require.NoError(t, err)
		names := set.Of(tables...)
		assert.False(t, names.Contains("alpha"))
		assert.True(t, names.Contains("bravo"))
	})
	t.Run("should do nothing when no new migrations", func(t *testing.T) {
		// given
		db := CreateTestDB(t)
		require.NoError(t, createMigrationTracking(db))
		require.NoError(t, recordMigration(db, "0001_alpha"))
		require.NoError(t, recordMigration(db, "0002_bravo"))
		// when
		err := applyNewMigrations(db, migrations)
		// then
		require.NoError(t, err)
		applied, err := listMigrationNames(db)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"0001_alpha", "0002_bravo"}, applied)
		tables, err := ListTableNames(db)
		require.NoError(t, err)
		names := set.Of(tables...)
		assert.False(t, names.Contains("alpha"))
		assert.False(t, names.Contains("bravo"))
	})
	t.Run("should ignore files without sql extension", func(t *testing.T) {
		// given
		db := CreateTestDB(t)
		require.NoError(t, createMigrationTracking(db))
		migrations := fstest.MapFS{
			"migrations/0001_alpha.sql": &fstest.MapFile{
				Data: []byte("CREATE TABLE alpha(id INTEGER NOT NULL);"),
			},
			"migrations/0002_bravo.txt": &fstest.MapFile{
				Data: []byte("CREATE TABLE bravo(id INTEGER NOT NULL);"),
			},
			"migrations/0003_charlie.sql.bak": &fstest.MapFile{
				Data: []byte("CREATE TABLE charlie(id INTEGER NOT NULL);"),
			},
			"migrations/README": &fstest.MapFile{
				Data: []byte("not a migration"),
			},
		}
		// when
		err := applyNewMigrations(db, migrations)
		// then
		require.NoError(t, err)
		applied, err := listMigrationNames(db)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"0001_alpha"}, applied)
		tables, err := ListTableNames(db)
		require.NoError(t, err)
		names := set.Of(tables...)
		assert.True(t, names.Contains("alpha"))
		assert.False(t, names.Contains("bravo"))
		assert.False(t, names.Contains("charlie"))
	})
	t.Run("should apply migrations in name order when directory is unsorted", func(t *testing.T) {
		// given
		db := CreateTestDB(t)
		require.NoError(t, createMigrationTracking(db))
		migrations := fakeFS{
			MapFS: fstest.MapFS{
				"migrations/0001_alpha.sql": &fstest.MapFile{
					Data: []byte("CREATE TABLE alpha(id INTEGER NOT NULL);"),
				},
				"migrations/0002_bravo.sql": &fstest.MapFile{
					Data: []byte("CREATE TABLE bravo(id INTEGER NOT NULL);"),
				},
				"migrations/0003_charlie.sql": &fstest.MapFile{
					Data: []byte("ALTER TABLE bravo ADD COLUMN name TEXT;"),
				},
			},
			reverseReadDir: true,
		}
		// when
		err := applyNewMigrations(db, migrations)
		// then
		require.NoError(t, err)
		rows, err := db.Query("SELECT name FROM migrations ORDER BY id;")
		require.NoError(t, err)
		defer rows.Close()
		var applied []string
		for rows.Next() {
			var n string
			require.NoError(t, rows.Scan(&n))
			applied = append(applied, n)
		}
		require.NoError(t, rows.Err())
		xassert.Equal(t, []string{"0001_alpha", "0002_bravo", "0003_charlie"}, applied)
	})
	t.Run("should apply migration with multiple statements", func(t *testing.T) {
		// given
		db := CreateTestDB(t)
		require.NoError(t, createMigrationTracking(db))
		migrations := fstest.MapFS{
			"migrations/0001_alpha.sql": &fstest.MapFile{
				Data: []byte(`
					CREATE TABLE alpha(id INTEGER NOT NULL);
					CREATE TABLE bravo(id INTEGER NOT NULL);
					INSERT INTO alpha(id) VALUES(1);`),
			},
		}
		// when
		err := applyNewMigrations(db, migrations)
		// then
		require.NoError(t, err)
		tables, err := ListTableNames(db)
		require.NoError(t, err)
		names := set.Of(tables...)
		assert.True(t, names.Contains("alpha"))
		assert.True(t, names.Contains("bravo"))
		var count int
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM alpha;").Scan(&count))
		xassert.Equal(t, 1, count)
	})
	t.Run("should stop and roll back failed migration", func(t *testing.T) {
		// given
		db := CreateTestDB(t)
		require.NoError(t, createMigrationTracking(db))
		migrations := fstest.MapFS{
			"migrations/0001_alpha.sql": &fstest.MapFile{
				Data: []byte("CREATE TABLE alpha(id INTEGER NOT NULL);"),
			},
			"migrations/0002_bravo.sql": &fstest.MapFile{
				Data: []byte(`
					CREATE TABLE bravo(id INTEGER NOT NULL);
					INVALID SQL;`),
			},
			"migrations/0003_charlie.sql": &fstest.MapFile{
				Data: []byte("CREATE TABLE charlie(id INTEGER NOT NULL);"),
			},
		}
		// when
		err := applyNewMigrations(db, migrations)
		// then
		require.Error(t, err)
		applied, err := listMigrationNames(db)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"0001_alpha"}, applied)
		tables, err := ListTableNames(db)
		require.NoError(t, err)
		names := set.Of(tables...)
		assert.True(t, names.Contains("alpha"))
		assert.False(t, names.Contains("bravo"))
		assert.False(t, names.Contains("charlie"))
	})
	t.Run("should roll back when recording the migration fails", func(t *testing.T) {
		// given
		db := CreateTestDB(t)
		require.NoError(t, createMigrationTracking(db))
		migrations := fstest.MapFS{
			"migrations/0001_alpha.sql": &fstest.MapFile{
				Data: []byte("INSERT INTO migrations(name) VALUES('0001_alpha');"),
			},
		}
		// when
		err := applyNewMigrations(db, migrations)
		// then
		require.Error(t, err)
		applied, err := listMigrationNames(db)
		require.NoError(t, err)
		assert.Empty(t, applied)
	})
	t.Run("should return error when migrations folder is missing", func(t *testing.T) {
		// given
		db := CreateTestDB(t)
		require.NoError(t, createMigrationTracking(db))
		migrations := fstest.MapFS{
			"other/0001_alpha.sql": &fstest.MapFile{
				Data: []byte("CREATE TABLE alpha(id INTEGER NOT NULL);"),
			},
		}
		// when
		err := applyNewMigrations(db, migrations)
		// then
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})
	t.Run("should return error when migration file can not be read", func(t *testing.T) {
		// given
		db := CreateTestDB(t)
		require.NoError(t, createMigrationTracking(db))
		errRead := errors.New("read error")
		migrations := fakeFS{
			MapFS: fstest.MapFS{
				"migrations/0001_alpha.sql": &fstest.MapFile{
					Data: []byte("CREATE TABLE alpha(id INTEGER NOT NULL);"),
				},
			},
			readFileErr: errRead,
		}
		// when
		err := applyNewMigrations(db, migrations)
		// then
		assert.ErrorIs(t, err, errRead)
		applied, err := listMigrationNames(db)
		require.NoError(t, err)
		assert.Empty(t, applied)
	})
	t.Run("should return error when tracking table is missing", func(t *testing.T) {
		// given
		db := CreateTestDB(t)
		// when
		err := applyNewMigrations(db, migrations)
		// then
		assert.ErrorContains(t, err, "list applied migrations")
	})
}

func TestListMigrationNames(t *testing.T) {
	t.Run("should return recorded migrations", func(t *testing.T) {
		// given
		db := CreateTestDB(t)
		require.NoError(t, createMigrationTracking(db))
		require.NoError(t, recordMigration(db, "test1"))
		require.NoError(t, recordMigration(db, "test2"))
		// when
		names, err := listMigrationNames(db)
		// then
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"test1", "test2"}, names)
	})
	t.Run("should return empty when no migrations recorded", func(t *testing.T) {
		// given
		db := CreateTestDB(t)
		require.NoError(t, createMigrationTracking(db))
		// when
		names, err := listMigrationNames(db)
		// then
		require.NoError(t, err)
		assert.Empty(t, names)
	})
	t.Run("should return error when db is closed", func(t *testing.T) {
		// given
		db := CreateTestDB(t)
		require.NoError(t, createMigrationTracking(db))
		require.NoError(t, db.Close())
		// when
		_, err := listMigrationNames(db)
		// then
		assert.Error(t, err)
	})
}

func TestCreateMigrationTracking(t *testing.T) {
	t.Run("should keep recorded migrations when tracking table already exists", func(t *testing.T) {
		// given
		db := CreateTestDB(t)
		require.NoError(t, createMigrationTracking(db))
		require.NoError(t, recordMigration(db, "test1"))
		// when
		err := createMigrationTracking(db)
		// then
		require.NoError(t, err)
		names, err := listMigrationNames(db)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"test1"}, names)
	})
}

func recordMigration(db *sql.DB, name string) error {
	_, err := db.Exec(`INSERT INTO migrations(name) VALUES(?);`, name)
	return err
}

// CreateTestDB returns a new in-memory database, which is closed when the test ends.
func CreateTestDB(t testing.TB) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1) // every connection to :memory: opens a separate database
	t.Cleanup(func() {
		db.Close()
	})
	return db
}

// ListTableNames returns the names of all objects in the database.
func ListTableNames(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT name from sqlite_master;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		names = append(names, n)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return names, nil
}

// fakeFS is a [MigrateFS] which can return unsorted directory entries and fail reading files.
type fakeFS struct {
	fstest.MapFS
	reverseReadDir bool
	readFileErr    error
}

func (f fakeFS) ReadDir(name string) ([]fs.DirEntry, error) {
	entries, err := f.MapFS.ReadDir(name)
	if err != nil {
		return nil, err
	}
	if f.reverseReadDir {
		slices.Reverse(entries)
	}
	return entries, nil
}

func (f fakeFS) ReadFile(name string) ([]byte, error) {
	if f.readFileErr != nil {
		return nil, f.readFileErr
	}
	return f.MapFS.ReadFile(name)
}
