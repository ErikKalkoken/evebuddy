package repository_test

import (
	"context"
	"example/evebuddy/internal/repository"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetting(t *testing.T) {
	db, _, _ := setUpDB()
	defer db.Close()
	r := repository.New(db)
	ctx := context.Background()
	t.Run("can create new string", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		// when
		err := r.SetSettingString(ctx, "alpha", "john")
		// then
		if assert.NoError(t, err) {
			v, err := r.GetSettingString(ctx, "alpha")
			if assert.NoError(t, err) {
				assert.Equal(t, "john", v)
			}
		}
	})
	t.Run("can create new int", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		// when
		err := r.SetSettingInt32(ctx, "alpha", 42)
		// then
		if assert.NoError(t, err) {
			v, err := r.GetSettingInt32(ctx, "alpha")
			if assert.NoError(t, err) {
				assert.Equal(t, int32(42), v)
			}
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		err := r.SetSettingString(ctx, "alpha", "john")
		if err != nil {
			panic(err)
		}
		// when
		err = r.SetSettingString(ctx, "alpha", "peter")
		// then
		if assert.NoError(t, err) {
			v, err := r.GetSettingString(ctx, "alpha")
			if assert.NoError(t, err) {
				assert.Equal(t, "peter", v)
			}
		}
	})
	t.Run("should return empty string if not found", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		// when
		v, err := r.GetSettingString(ctx, "alpha")
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, "", v)
		}
	})
	t.Run("should return 0 if not found", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		// when
		v, err := r.GetSettingInt32(ctx, "alpha")
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(0), v)
		}
	})
	t.Run("can delete existing key", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		err := r.SetSettingString(ctx, "alpha", "abc")
		if err != nil {
			panic(err)
		}
		// when
		err = r.DeleteSetting(ctx, "alpha")
		// then
		if assert.NoError(t, err) {
			v, err := r.GetSettingString(ctx, "alpha")
			if assert.NoError(t, err) {
				assert.Equal(t, "", v)
			}
		}
	})
	t.Run("can delete not existing key", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		// when
		err := r.DeleteSetting(ctx, "alpha")
		// then
		if assert.NoError(t, err) {
			v, err := r.GetSettingString(ctx, "alpha")
			if assert.NoError(t, err) {
				assert.Equal(t, "", v)
			}
		}
	})
}
