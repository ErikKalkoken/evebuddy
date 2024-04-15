package service_test

import (
	"example/evebuddy/internal/repository"
	"example/evebuddy/internal/service"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetting(t *testing.T) {
	db, q, _ := setUpDB()
	defer db.Close()
	s := service.NewService(q)
	t.Run("can create new string", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		// when
		err := s.SetSettingString("alpha", "john")
		// then
		if assert.NoError(t, err) {
			v, err := s.GetSettingString("alpha")
			if assert.NoError(t, err) {
				assert.Equal(t, "john", v)
			}
		}
	})
	t.Run("can create new int", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		// when
		err := s.SetSettingInt32("alpha", 42)
		// then
		if assert.NoError(t, err) {
			v, err := s.GetSettingInt32("alpha")
			if assert.NoError(t, err) {
				assert.Equal(t, int32(42), v)
			}
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		err := s.SetSettingString("alpha", "john")
		if err != nil {
			panic(err)
		}
		// when
		err = s.SetSettingString("alpha", "peter")
		// then
		if assert.NoError(t, err) {
			v, err := s.GetSettingString("alpha")
			if assert.NoError(t, err) {
				assert.Equal(t, "peter", v)
			}
		}
	})
	t.Run("should return empty string if not found", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		// when
		v, err := s.GetSettingString("alpha")
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, "", v)
		}
	})
	t.Run("should return 0 if not found", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		// when
		v, err := s.GetSettingInt32("alpha")
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(0), v)
		}
	})
	t.Run("can delete existing key", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		err := s.SetSettingString("alpha", "abc")
		if err != nil {
			panic(err)
		}
		// when
		err = s.DeleteSetting("alpha")
		// then
		if assert.NoError(t, err) {
			v, err := s.GetSettingString("alpha")
			if assert.NoError(t, err) {
				assert.Equal(t, "", v)
			}
		}
	})
	t.Run("can delete not existing key", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		// when
		err := s.DeleteSetting("alpha")
		// then
		if assert.NoError(t, err) {
			v, err := s.GetSettingString("alpha")
			if assert.NoError(t, err) {
				assert.Equal(t, "", v)
			}
		}
	})
}
