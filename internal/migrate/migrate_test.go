package migrate_test

import (
	"testing"
	"testing/fstest"

	"github.com/ErikKalkoken/kx/set"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/migrate"
)

func TestMigrate(t *testing.T) {
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
		db := migrate.CreateTestDB()
		// when
		err := migrate.Run(db, migrations)
		// then
		if assert.NoError(t, err) {
			tables, err := migrate.ListTableNames(db)
			if assert.NoError(t, err) {
				names := set.Of(tables...)
				assert.True(t, names.Contains("alpha"))
				assert.True(t, names.Contains("bravo"))
			}
		}
	})
}
