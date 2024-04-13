package service_test

import (
	"example/evebuddy/internal/model"
	"example/evebuddy/internal/service"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetting(t *testing.T) {
	t.Run("can create new string", func(t *testing.T) {
		// given
		model.TruncateTables()
		// when
		err := service.SetSetting("alpha", "john")
		// then
		if assert.NoError(t, err) {
			v, err := service.GetSetting[string]("alpha")
			if assert.NoError(t, err) {
				assert.Equal(t, "john", v)
			}
		}
	})
	t.Run("can create new int", func(t *testing.T) {
		// given
		model.TruncateTables()
		// when
		err := service.SetSetting("alpha", 42)
		// then
		if assert.NoError(t, err) {
			v, err := service.GetSetting[int]("alpha")
			if assert.NoError(t, err) {
				assert.Equal(t, 42, v)
			}
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		model.TruncateTables()
		err := service.SetSetting("alpha", "john")
		if err != nil {
			panic(err)
		}
		// when
		err = service.SetSetting("alpha", "peter")
		// then
		if assert.NoError(t, err) {
			v, err := service.GetSetting[string]("alpha")
			if assert.NoError(t, err) {
				assert.Equal(t, "peter", v)
			}
		}
	})
	t.Run("should return empty string if not found", func(t *testing.T) {
		// given
		model.TruncateTables()
		// when
		v, err := service.GetSetting[string]("alpha")
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, "", v)
		}
	})
	t.Run("should return 0 if not found", func(t *testing.T) {
		// given
		model.TruncateTables()
		// when
		v, err := service.GetSetting[int]("alpha")
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, 0, v)
		}
	})
	t.Run("can delete existing key", func(t *testing.T) {
		// given
		model.TruncateTables()
		err := service.SetSetting("alpha", "abc")
		if err != nil {
			panic(err)
		}
		// when
		err = service.DeleteSetting("alpha")
		// then
		if assert.NoError(t, err) {
			v, err := service.GetSetting[string]("alpha")
			if assert.NoError(t, err) {
				assert.Equal(t, "", v)
			}
		}
	})
	t.Run("can delete not existing key", func(t *testing.T) {
		// given
		model.TruncateTables()
		// when
		err := service.DeleteSetting("alpha")
		// then
		if assert.NoError(t, err) {
			v, err := service.GetSetting[string]("alpha")
			if assert.NoError(t, err) {
				assert.Equal(t, "", v)
			}
		}
	})
}
