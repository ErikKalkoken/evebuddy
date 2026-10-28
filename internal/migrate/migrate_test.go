package migrate_test

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/migrate"
)

func TestRun(t *testing.T) {
	migrations := fstest.MapFS{
		"migrations/0001_alpha.sql": &fstest.MapFile{
			Data: []byte("CREATE TABLE alpha(id INTEGER NOT NULL);"),
		},
		"migrations/0002_bravo.sql": &fstest.MapFile{
			Data: []byte("CREATE TABLE bravo(id INTEGER NOT NULL);"),
		},
	}
	t.Run("should run all migrations when new", func(t *testing.T) {
		// given
		db := migrate.CreateTestDB(t)
		// when
		err := migrate.Run(db, migrations)
		// then
		require.NoError(t, err)
		tables, err := migrate.ListTableNames(db)
		require.NoError(t, err)
		names := set.Of(tables...)
		assert.True(t, names.Contains("migrations"))
		assert.True(t, names.Contains("alpha"))
		assert.True(t, names.Contains("bravo"))
	})
	t.Run("should do nothing when run again", func(t *testing.T) {
		// given
		db := migrate.CreateTestDB(t)
		require.NoError(t, migrate.Run(db, migrations))
		// when
		err := migrate.Run(db, migrations)
		// then
		require.NoError(t, err)
	})
	t.Run("should apply only new migrations when run again", func(t *testing.T) {
		// given
		db := migrate.CreateTestDB(t)
		require.NoError(t, migrate.Run(db, migrations))
		migrations2 := fstest.MapFS{
			"migrations/0001_alpha.sql": migrations["migrations/0001_alpha.sql"],
			"migrations/0002_bravo.sql": migrations["migrations/0002_bravo.sql"],
			"migrations/0003_charlie.sql": &fstest.MapFile{
				Data: []byte("CREATE TABLE charlie(id INTEGER NOT NULL);"),
			},
		}
		// when
		err := migrate.Run(db, migrations2)
		// then
		require.NoError(t, err)
		tables, err := migrate.ListTableNames(db)
		require.NoError(t, err)
		assert.True(t, set.Of(tables...).Contains("charlie"))
	})
	t.Run("should run all migrations when db has other tables but no tracking table", func(t *testing.T) {
		// given
		db := migrate.CreateTestDB(t)
		_, err := db.Exec("CREATE TABLE other(id INTEGER NOT NULL);")
		require.NoError(t, err)
		// when
		err = migrate.Run(db, migrations)
		// then
		require.NoError(t, err)
		tables, err := migrate.ListTableNames(db)
		require.NoError(t, err)
		names := set.Of(tables...)
		assert.True(t, names.Contains("migrations"))
		assert.True(t, names.Contains("alpha"))
		assert.True(t, names.Contains("bravo"))
	})
	t.Run("should return error when migrations folder is missing", func(t *testing.T) {
		// given
		db := migrate.CreateTestDB(t)
		migrations := fstest.MapFS{
			"other/0001_alpha.sql": &fstest.MapFile{
				Data: []byte("CREATE TABLE alpha(id INTEGER NOT NULL);"),
			},
		}
		// when
		err := migrate.Run(db, migrations)
		// then
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})
	t.Run("should return error when db is closed", func(t *testing.T) {
		// given
		db := migrate.CreateTestDB(t)
		require.NoError(t, db.Close())
		// when
		err := migrate.Run(db, migrations)
		// then
		assert.Error(t, err)
	})
}
