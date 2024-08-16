package migrate_test

import (
	"testing"
	"testing/fstest"

	"github.com/ErikKalkoken/evebuddy/internal/migrate"
	"github.com/ErikKalkoken/evebuddy/pkg/set"
	"github.com/stretchr/testify/assert"
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
				names := set.NewFromSlice(tables)
				assert.True(t, names.Has("alpha"))
				assert.True(t, names.Has("bravo"))
			}
		}
	})
}
