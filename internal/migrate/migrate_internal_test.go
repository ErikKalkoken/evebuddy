package migrate

import (
	"database/sql"
	"testing"
	"testing/fstest"

	"github.com/ErikKalkoken/evebuddy/pkg/set"
	"github.com/stretchr/testify/assert"
)

func TestApplyMigrations(t *testing.T) {
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
		db := CreateTestDB()
		createMigrationTracking(db)
		// when
		err := applyNewMigrations(db, migrations)
		// then
		if assert.NoError(t, err) {
			applied, err := listMigrationNames(db)
			if assert.NoError(t, err) {
				assert.Equal(t, []string{"0001_alpha", "0002_bravo"}, applied)
			}
			tables, err := ListTableNames(db)
			if assert.NoError(t, err) {
				names := set.NewFromSlice(tables)
				assert.True(t, names.Contains("alpha"))
				assert.True(t, names.Contains("bravo"))
			}
		}
	})
	t.Run("should apply new migrations only", func(t *testing.T) {
		// given
		db := CreateTestDB()
		createMigrationTracking(db)
		recordMigration(db, "0001_alpha")
		// when
		err := applyNewMigrations(db, migrations)
		// then
		if assert.NoError(t, err) {
			applied, err := listMigrationNames(db)
			if assert.NoError(t, err) {
				assert.Equal(t, []string{"0001_alpha", "0002_bravo"}, applied)
			}
			tables, err := ListTableNames(db)
			if assert.NoError(t, err) {
				names := set.NewFromSlice(tables)
				assert.False(t, names.Contains("alpha"))
				assert.True(t, names.Contains("bravo"))
			}
		}
	})
	t.Run("should do nothing when no new migrations", func(t *testing.T) {
		// given
		db := CreateTestDB()
		createMigrationTracking(db)
		recordMigration(db, "0001_alpha")
		recordMigration(db, "0002_bravo")
		// when
		err := applyNewMigrations(db, migrations)
		// then
		if assert.NoError(t, err) {
			applied, err := listMigrationNames(db)
			if assert.NoError(t, err) {
				assert.Equal(t, []string{"0001_alpha", "0002_bravo"}, applied)
			}
			tables, err := ListTableNames(db)
			if assert.NoError(t, err) {
				names := set.NewFromSlice(tables)
				assert.False(t, names.Contains("alpha"))
				assert.False(t, names.Contains("bravo"))
			}
		}
	})
}

func TestMigrate(t *testing.T) {
	t.Run("can record migrations", func(t *testing.T) {
		db := CreateTestDB()
		createMigrationTracking(db)
		recordMigration(db, "test1")
		recordMigration(db, "test2")
		names, err := listMigrationNames(db)
		if assert.NoError(t, err) {
			assert.Equal(t, []string{"test1", "test2"}, names)
		}
	})
	t.Run("should return true when db is empty", func(t *testing.T) {
		// given
		db := CreateTestDB()
		// when
		r, err := isEmpty(db)
		// then
		if assert.NoError(t, err) {
			assert.True(t, r)
		}
	})
	t.Run("should return false when db is not empty", func(t *testing.T) {
		// given
		db := CreateTestDB()
		if err := createMigrationTracking(db); err != nil {
			t.Fatalf("Failed to create tables: %s", err)
		}
		// when
		r, err := isEmpty(db)
		// then
		if assert.NoError(t, err) {
			assert.False(t, r)
		}
	})
}

func CreateTestDB() *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}
	return db
}

// isEmpty reports wether the database is empty.
func ListTableNames(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT name from sqlite_master;")
	if err != nil {
		return nil, err
	}
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
