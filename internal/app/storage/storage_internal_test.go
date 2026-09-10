package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestConvertNoRowsError(t *testing.T) {
	t.Run("converts no rows error", func(t *testing.T) {
		got := convertGetError(sql.ErrNoRows)
		xassert.Equal(t, app.ErrNotFound, got)
	})
	t.Run("passes through other errors", func(t *testing.T) {
		err := errors.New("random error")
		got := convertGetError(err)
		xassert.Equal(t, err, got)
	})
}

func TestDumpData(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()
	require.NoError(t, ApplyMigrations(db))
	st := New(db, db)
	t.Run("can dump all tables", func(t *testing.T) {
		// given
		_, err := st.CreateTag(t.Context(), "Alpha")
		require.NoError(t, err)
		// when
		got := st.DumpData()
		// then
		var world map[string]any
		require.NoError(t, json.Unmarshal([]byte(got), &world))
		assert.Contains(t, world, "character_tags")
	})
	t.Run("can dump specific tables only", func(t *testing.T) {
		// given
		_, err := st.CreateTag(t.Context(), "Bravo")
		require.NoError(t, err)
		// when
		got := st.DumpData("character_tags")
		// then
		var world map[string]any
		require.NoError(t, json.Unmarshal([]byte(got), &world))
		assert.Contains(t, world, "character_tags")
		assert.Len(t, world, 1)
	})
	t.Run("panics for unknown table", func(t *testing.T) {
		assert.Panics(t, func() {
			st.DumpData("does_not_exist")
		})
	})
}
