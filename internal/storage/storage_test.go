package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSchemaExists(t *testing.T) {
	t.Run("should return false when schema doesn't exists", func(t *testing.T) {
		// given
		db := createTestDB()
		// when
		r, err := schemaExists(db)
		// then
		if assert.NoError(t, err) {
			assert.False(t, r)
		}
	})
	t.Run("should return true when schema exists", func(t *testing.T) {
		// given
		db := createTestDB()
		err := runMigrations(db)
		if err != nil {
			t.Fatal("Failed to run migrations")
		}
		// when
		r, err := schemaExists(db)
		// then
		if assert.NoError(t, err) {
			assert.True(t, r)
		}
	})
}
