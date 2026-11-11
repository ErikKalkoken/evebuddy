package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"github.com/ErikKalkoken/go-set"
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

func TestMissingIDs(t *testing.T) {
	t.Run("returns missing IDs across multiple chunks", func(t *testing.T) {
		// given
		st := &Storage{MaxIDsPerQuery: 1}
		stored := set.Of[int64](42, 43)
		var calls int
		listForIDs := func(_ context.Context, ids []int64) ([]int64, error) {
			calls++
			var found []int64
			for _, id := range ids {
				if stored.Contains(id) {
					found = append(found, id)
				}
			}
			return found, nil
		}
		// when
		got, err := st.missingIDs(t.Context(), set.Of[int64](42, 5, 43, 6), listForIDs)
		// then
		require.NoError(t, err)
		xassert.Equal(t, set.Of[int64](5, 6), got)
		xassert.Equal(t, 4, calls)
	})
	t.Run("makes no query for empty set", func(t *testing.T) {
		// given
		st := &Storage{MaxIDsPerQuery: 1}
		var calls int
		listForIDs := func(_ context.Context, _ []int64) ([]int64, error) {
			calls++
			return nil, nil
		}
		// when
		got, err := st.missingIDs(t.Context(), set.Set[int64]{}, listForIDs)
		// then
		require.NoError(t, err)
		xassert.Equal(t, 0, got.Size())
		xassert.Equal(t, 0, calls)
	})
	t.Run("returns wrapped query error", func(t *testing.T) {
		// given
		st := &Storage{MaxIDsPerQuery: 1}
		errQuery := errors.New("query error")
		listForIDs := func(_ context.Context, _ []int64) ([]int64, error) {
			return nil, errQuery
		}
		// when
		_, err := st.missingIDs(t.Context(), set.Of[int64](42, 43), listForIDs)
		// then
		assert.ErrorIs(t, err, errQuery)
		assert.ErrorContains(t, err, "1 ids:")
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
