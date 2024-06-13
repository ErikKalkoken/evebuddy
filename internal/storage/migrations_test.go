package storage

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMigration(t *testing.T) {
	t.Run("can record migrations", func(t *testing.T) {
		db := createTestDB()
		createMigrationTracking(db)
		recordMigration(db, "test1")
		recordMigration(db, "test2")
		names, err := listMigrations(db)
		if assert.NoError(t, err) {
			assert.Equal(t, []string{"test1", "test2"}, names)
		}
	})
}

func createTestDB() *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}
	return db
}
