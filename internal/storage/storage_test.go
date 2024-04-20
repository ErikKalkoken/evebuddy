package storage

import (
	"database/sql"
	"example/evebuddy/internal/storage/sqlc"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSchemaExists(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if assert.NoError(t, err) {
		t.Run("should return false when schema doesn't exists", func(t *testing.T) {
			// when
			r, err := schemaExists(db)
			// then
			if assert.NoError(t, err) {
				assert.False(t, r)
			}
		})
		t.Run("should return true when schema exists", func(t *testing.T) {
			// given
			_, err = db.Exec(sqlc.Schema())
			if assert.NoError(t, err) {
				// when
				r, err := schemaExists(db)
				// then
				if assert.NoError(t, err) {
					assert.True(t, r)
				}
			}
		})
	}
}
