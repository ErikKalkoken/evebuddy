package service_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/service"
)

func TestDictionary(t *testing.T) {
	db, r, _ := testutil.New()
	defer db.Close()
	s := service.NewService(r)
	t.Run("can use string entries", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		err := s.DictionarySetString("alpha", "john")
		// then
		if assert.NoError(t, err) {
			v, ok, err := s.DictionaryString("alpha")
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
		err := s.DictionarySetInt("alpha", 42)
		// then
		if assert.NoError(t, err) {
			v, ok, err := s.DictionaryInt("alpha")
			if assert.NoError(t, err) {
				assert.True(t, ok)
				assert.Equal(t, 42, v)
			}
		}
	})
	t.Run("can use time entries", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		v := time.Now()
		// when
		err := s.DictionarySetTime("alpha", v)
		// then
		if assert.NoError(t, err) {
			v2, ok, err := s.DictionaryTime("alpha")
			if assert.NoError(t, err) {
				assert.True(t, ok)
				assert.Equal(t, v.UnixMicro(), v2.UnixMicro())
			}
		}
	})
	t.Run("can update existing entry", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		err := s.DictionarySetString("alpha", "john")
		if err != nil {
			t.Fatal(err)
		}
		// when
		err = s.DictionarySetString("alpha", "peter")
		// then
		if assert.NoError(t, err) {
			v, _, err := s.DictionaryString("alpha")
			if assert.NoError(t, err) {
				assert.Equal(t, "peter", v)
			}
		}
	})
	t.Run("should return nok when not found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		_, ok, err := s.DictionaryString("alpha")
		// then
		if assert.NoError(t, err) {
			assert.False(t, ok)
		}
	})
	t.Run("should return nok when not found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		_, ok, err := s.DictionaryInt("alpha")
		// then
		if assert.NoError(t, err) {
			assert.False(t, ok)
		}
	})
	t.Run("should return nok when not found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		_, ok, err := s.DictionaryTime("alpha")
		// then
		if assert.NoError(t, err) {
			assert.False(t, ok)
		}
	})
	t.Run("can delete existing key", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		err := s.DictionarySetString("alpha", "abc")
		if err != nil {
			t.Fatal(err)
		}
		// when
		err = s.DictionaryDelete("alpha")
		// then
		if assert.NoError(t, err) {
			v, _, err := s.DictionaryString("alpha")
			if assert.NoError(t, err) {
				assert.Equal(t, "", v)
			}
		}
	})
	t.Run("can delete not existing key", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		err := s.DictionaryDelete("alpha")
		// then
		if assert.NoError(t, err) {
			v, _, err := s.DictionaryString("alpha")
			if assert.NoError(t, err) {
				assert.Equal(t, "", v)
			}
		}
	})
	t.Run("should return false when key doesn't exist", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		v, err := s.DictionaryExists("alpha")
		// then
		if assert.NoError(t, err) {
			assert.False(t, v)
		}
	})
	t.Run("should return true when key exists", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		if err := s.DictionarySetString("alpha", "abc"); err != nil {
			t.Fatal(err)
		}
		// when
		v, err := s.DictionaryExists("alpha")
		// then
		if assert.NoError(t, err) {
			assert.True(t, v)
		}
	})
}
