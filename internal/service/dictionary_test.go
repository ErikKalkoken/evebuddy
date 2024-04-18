package service_test

import (
	"example/evebuddy/internal/service"
	"example/evebuddy/internal/testutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDictionary(t *testing.T) {
	db, r, _ := testutil.New()
	defer db.Close()
	s := service.NewService(r)
	t.Run("can create new string", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		err := s.SetDictKeyString("alpha", "john")
		// then
		if assert.NoError(t, err) {
			v, err := s.GetDictKeyString("alpha")
			if assert.NoError(t, err) {
				assert.Equal(t, "john", v)
			}
		}
	})
	t.Run("can create new int", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		err := s.SetDictKeyInt("alpha", 42)
		// then
		if assert.NoError(t, err) {
			v, err := s.GetDictKeyInt("alpha")
			if assert.NoError(t, err) {
				assert.Equal(t, 42, v)
			}
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		err := s.SetDictKeyString("alpha", "john")
		if err != nil {
			panic(err)
		}
		// when
		err = s.SetDictKeyString("alpha", "peter")
		// then
		if assert.NoError(t, err) {
			v, err := s.GetDictKeyString("alpha")
			if assert.NoError(t, err) {
				assert.Equal(t, "peter", v)
			}
		}
	})
	t.Run("should return empty string if not found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		v, err := s.GetDictKeyString("alpha")
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, "", v)
		}
	})
	t.Run("should return 0 if not found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		v, err := s.GetDictKeyInt("alpha")
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, 0, v)
		}
	})
	t.Run("can delete existing key", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		err := s.SetDictKeyString("alpha", "abc")
		if err != nil {
			panic(err)
		}
		// when
		err = s.DeleteDictKey("alpha")
		// then
		if assert.NoError(t, err) {
			v, err := s.GetDictKeyString("alpha")
			if assert.NoError(t, err) {
				assert.Equal(t, "", v)
			}
		}
	})
	t.Run("can delete not existing key", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		err := s.DeleteDictKey("alpha")
		// then
		if assert.NoError(t, err) {
			v, err := s.GetDictKeyString("alpha")
			if assert.NoError(t, err) {
				assert.Equal(t, "", v)
			}
		}
	})
	t.Run("should return false when key doesn't exist", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		v, err := s.ExistsDictKey("alpha")
		// then
		if assert.NoError(t, err) {
			assert.False(t, v)
		}
	})
	t.Run("should return true when key exists", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		if err := s.SetDictKeyString("alpha", "abc"); err != nil {
			panic(err)
		}
		// when
		v, err := s.ExistsDictKey("alpha")
		// then
		if assert.NoError(t, err) {
			assert.True(t, v)
		}
	})
}
