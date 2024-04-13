package model_test

import (
	"example/evebuddy/internal/factory"
	"example/evebuddy/internal/model"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTokenSave(t *testing.T) {
	t.Run("can create new", func(t *testing.T) {
		// given
		model.TruncateTables()
		c := factory.CreateCharacter()
		o := model.Token{AccessToken: "access", CharacterID: c.ID, ExpiresAt: time.Now(), RefreshToken: "refresh", TokenType: "xxx"}
		// when
		err := o.Save()
		// then
		assert.NoError(t, err)
		r, err := model.GetToken(c.ID)
		if assert.NoError(t, err) {
			assert.Equal(t, o.AccessToken, r.AccessToken)
			assert.Equal(t, o.CharacterID, r.CharacterID)
		}
	})
	t.Run("should return error when obj has no character", func(t *testing.T) {
		// given
		model.TruncateTables()
		o := model.Token{AccessToken: "access", ExpiresAt: time.Now(), RefreshToken: "refresh", TokenType: "xxx"}
		// when
		err := o.Save()
		// then
		assert.Error(t, err)
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		model.TruncateTables()
		factory.CreateCharacter()
		o := factory.CreateToken()
		assert.NoError(t, o.Save())
		o.AccessToken = "new-access"
		// when
		err := o.Save()
		assert.NoError(t, err)
		r, err := model.GetToken(o.CharacterID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, o.AccessToken, r.AccessToken)
			assert.Equal(t, o.CharacterID, r.CharacterID)
		}

	})
}

func TestFetchToken(t *testing.T) {
	t.Run("can fetch existing by ID", func(t *testing.T) {
		// given
		model.TruncateTables()
		factory.CreateToken()
		o := factory.CreateToken()
		// when
		r, err := model.GetToken(o.CharacterID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, o.AccessToken, r.AccessToken)
			assert.Equal(t, o.CharacterID, r.CharacterID)
		}
	})
	t.Run("should return error when not exists", func(t *testing.T) {
		// given
		model.TruncateTables()
		// when
		_, err := model.GetToken(42)
		// then
		assert.Error(t, err)
	})
}
