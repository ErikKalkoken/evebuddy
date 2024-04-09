package model_test

import (
	"example/evebuddy/internal/model"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetting(t *testing.T) {
	t.Run("can create new", func(t *testing.T) {
		// given
		model.TruncateTables()
		// when
		err := model.SetSetting("alpha", "john")
		// then
		if assert.NoError(t, err) {
			v, err := model.GetSetting("alpha")
			if assert.NoError(t, err) {
				assert.Equal(t, "john", v)
			}
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		model.TruncateTables()
		err := model.SetSetting("alpha", "john")
		if err != nil {
			panic(err)
		}
		// when
		err = model.SetSetting("alpha", "peter")
		// then
		if assert.NoError(t, err) {
			v, err := model.GetSetting("alpha")
			if assert.NoError(t, err) {
				assert.Equal(t, "peter", v)
			}
		}
	})
	t.Run("should return empty string if not found", func(t *testing.T) {
		// given
		model.TruncateTables()
		// when
		v, err := model.GetSetting("alpha")
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, "", v)
		}
	})
}
