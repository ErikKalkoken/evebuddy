package logic_test

import (
	"example/evebuddy/internal/logic"
	"example/evebuddy/internal/model"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetting(t *testing.T) {
	t.Run("can create new string", func(t *testing.T) {
		// given
		model.TruncateTables()
		// when
		err := logic.SetSetting("alpha", "john")
		// then
		if assert.NoError(t, err) {
			v, err := logic.GetSetting[string]("alpha")
			if assert.NoError(t, err) {
				assert.Equal(t, "john", v)
			}
		}
	})
	t.Run("can create new int", func(t *testing.T) {
		// given
		model.TruncateTables()
		// when
		err := logic.SetSetting("alpha", 42)
		// then
		if assert.NoError(t, err) {
			v, err := logic.GetSetting[int]("alpha")
			if assert.NoError(t, err) {
				assert.Equal(t, 42, v)
			}
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		model.TruncateTables()
		err := logic.SetSetting("alpha", "john")
		if err != nil {
			panic(err)
		}
		// when
		err = logic.SetSetting("alpha", "peter")
		// then
		if assert.NoError(t, err) {
			v, err := logic.GetSetting[string]("alpha")
			if assert.NoError(t, err) {
				assert.Equal(t, "peter", v)
			}
		}
	})
	t.Run("should return empty string if not found", func(t *testing.T) {
		// given
		model.TruncateTables()
		// when
		v, err := logic.GetSetting[string]("alpha")
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, "", v)
		}
	})
	t.Run("should return 0 if not found", func(t *testing.T) {
		// given
		model.TruncateTables()
		// when
		v, err := logic.GetSetting[int]("alpha")
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, 0, v)
		}
	})
	t.Run("can delete existing key", func(t *testing.T) {
		// given
		model.TruncateTables()
		err := logic.SetSetting("alpha", "abc")
		if err != nil {
			panic(err)
		}
		// when
		err = logic.DeleteSetting("alpha")
		// then
		if assert.NoError(t, err) {
			v, err := logic.GetSetting[string]("alpha")
			if assert.NoError(t, err) {
				assert.Equal(t, "", v)
			}
		}
	})
	t.Run("can delete not existing key", func(t *testing.T) {
		// given
		model.TruncateTables()
		// when
		err := logic.DeleteSetting("alpha")
		// then
		if assert.NoError(t, err) {
			v, err := logic.GetSetting[string]("alpha")
			if assert.NoError(t, err) {
				assert.Equal(t, "", v)
			}
		}
	})
}
