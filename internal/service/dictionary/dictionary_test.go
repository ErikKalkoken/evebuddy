package dictionary_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/service/dictionary"
)

func TestDictionary(t *testing.T) {
	db, r, _ := testutil.New()
	defer db.Close()
	s := dictionary.New(r)
	t.Run("can use string entries", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		err := s.SetString("alpha", "john")
		// then
		if assert.NoError(t, err) {
			v, ok, err := s.GetString("alpha")
			if assert.NoError(t, err) {
				assert.True(t, ok)
				assert.Equal(t, "john", v)
			}
		}
	})
	t.Run("can use int entries", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		err := s.SetInt("alpha", 42)
		// then
		if assert.NoError(t, err) {
			v, ok, err := s.GetInt("alpha")
			if assert.NoError(t, err) {
				assert.True(t, ok)
				assert.Equal(t, 42, v)
			}
		}
	})
	t.Run("can use float entries", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		err := s.SetFloat32("alpha", 1.23)
		// then
		if assert.NoError(t, err) {
			v, ok, err := s.GetFloat32("alpha")
			if assert.NoError(t, err) {
				assert.True(t, ok)
				assert.Equal(t, float32(1.23), v)
			}
		}
	})
	t.Run("can use time entries", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		v := time.Now()
		// when
		err := s.SetTime("alpha", v)
		// then
		if assert.NoError(t, err) {
			v2, ok, err := s.GetTime("alpha")
			if assert.NoError(t, err) {
				assert.True(t, ok)
				assert.Equal(t, v.UnixMicro(), v2.UnixMicro())
			}
		}
	})
	t.Run("can update existing entry", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		err := s.SetString("alpha", "john")
		if err != nil {
			t.Fatal(err)
		}
		// when
		err = s.SetString("alpha", "peter")
		// then
		if assert.NoError(t, err) {
			v, _, err := s.GetString("alpha")
			if assert.NoError(t, err) {
				assert.Equal(t, "peter", v)
			}
		}
	})
	t.Run("should return nok when not found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		_, ok, err := s.GetString("alpha")
		// then
		if assert.NoError(t, err) {
			assert.False(t, ok)
		}
	})
	t.Run("should return nok when not found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		_, ok, err := s.GetInt("alpha")
		// then
		if assert.NoError(t, err) {
			assert.False(t, ok)
		}
	})
	t.Run("should return nok when not found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		_, ok, err := s.GetTime("alpha")
		// then
		if assert.NoError(t, err) {
			assert.False(t, ok)
		}
	})
	t.Run("can delete existing key", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		err := s.SetString("alpha", "abc")
		if err != nil {
			t.Fatal(err)
		}
		// when
		err = s.Delete("alpha")
		// then
		if assert.NoError(t, err) {
			v, _, err := s.GetString("alpha")
			if assert.NoError(t, err) {
				assert.Equal(t, "", v)
			}
		}
	})
	t.Run("can delete not existing key", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		err := s.Delete("alpha")
		// then
		if assert.NoError(t, err) {
			v, _, err := s.GetString("alpha")
			if assert.NoError(t, err) {
				assert.Equal(t, "", v)
			}
		}
	})
	t.Run("should return false when key doesn't exist", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		v, err := s.Exists("alpha")
		// then
		if assert.NoError(t, err) {
			assert.False(t, v)
		}
	})
	t.Run("should return true when key exists", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		if err := s.SetString("alpha", "abc"); err != nil {
			t.Fatal(err)
		}
		// when
		v, err := s.Exists("alpha")
		// then
		if assert.NoError(t, err) {
			assert.True(t, v)
		}
	})
	t.Run("should return fallback when key does not exist", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		v, err := s.GetIntWithFallback("alpha", 7)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, 7, v)
		}
	})
	t.Run("should return value when key does exists", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		if err := s.SetInt("alpha", 42); err != nil {
			t.Fatal(err)
		}
		// when
		v, err := s.GetIntWithFallback("alpha", 7)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, 42, v)
		}
	})
}