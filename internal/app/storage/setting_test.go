package storage_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestGetSetting(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()

	t.Run("should return value of existing setting", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		err := st.SetSetting(t.Context(), "alpha", "value")
		require.NoError(t, err)
		// when
		got, err := st.GetSetting(t.Context(), "alpha")
		// then
		require.NoError(t, err)
		xassert.Equal(t, "value", got)
	})

	t.Run("should return not found error when setting does not exist", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		_, err := st.GetSetting(t.Context(), "alpha")
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})

	t.Run("should return invalid error when key is empty", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		_, err := st.GetSetting(t.Context(), "")
		// then
		assert.ErrorIs(t, err, app.ErrInvalid)
	})
}

func TestSetSetting(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()

	t.Run("should create new setting", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		err := st.SetSetting(t.Context(), "alpha", "value")
		// then
		require.NoError(t, err)
		got, err := st.GetSetting(t.Context(), "alpha")
		require.NoError(t, err)
		xassert.Equal(t, "value", got)
	})

	t.Run("should overwrite existing setting", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		err := st.SetSetting(t.Context(), "alpha", "old")
		require.NoError(t, err)
		// when
		err = st.SetSetting(t.Context(), "alpha", "new")
		// then
		require.NoError(t, err)
		got, err := st.GetSetting(t.Context(), "alpha")
		require.NoError(t, err)
		xassert.Equal(t, "new", got)
	})

	t.Run("should return invalid error when key is empty", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		err := st.SetSetting(t.Context(), "", "value")
		// then
		assert.ErrorIs(t, err, app.ErrInvalid)
	})
}

func TestListSettings(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()

	t.Run("should return all settings", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		err := st.SetSetting(t.Context(), "alpha", "one")
		require.NoError(t, err)
		err = st.SetSetting(t.Context(), "bravo", "two")
		require.NoError(t, err)
		// when
		got, err := st.ListSettings(t.Context())
		// then
		require.NoError(t, err)
		want := []queries.Setting{
			{Key: "alpha", Value: "one"},
			{Key: "bravo", Value: "two"},
		}
		assert.ElementsMatch(t, want, got)
	})

	t.Run("should return empty list when there are no settings", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		got, err := st.ListSettings(t.Context())
		// then
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}

func TestDeleteSetting(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()

	t.Run("should delete existing setting", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		err := st.SetSetting(t.Context(), "alpha", "one")
		require.NoError(t, err)
		err = st.SetSetting(t.Context(), "bravo", "two")
		require.NoError(t, err)
		// when
		err = st.DeleteSetting(t.Context(), "alpha")
		// then
		require.NoError(t, err)
		got, err := st.ListSettings(t.Context())
		require.NoError(t, err)
		xassert.Equal(t, []queries.Setting{{Key: "bravo", Value: "two"}}, got)
	})

	t.Run("should not return error when setting does not exist", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		err := st.DeleteSetting(t.Context(), "alpha")
		// then
		assert.NoError(t, err)
	})

	t.Run("should return invalid error when key is empty", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		err := st.DeleteSetting(t.Context(), "")
		// then
		assert.ErrorIs(t, err, app.ErrInvalid)
	})
}
