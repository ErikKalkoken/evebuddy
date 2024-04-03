package model_test

import (
	"example/esiapp/internal/factory"
	"example/esiapp/internal/model"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTokenCanSaveNew(t *testing.T) {
	// given
	model.TruncateTables()
	c := factory.CreateCharacter()
	o := model.Token{AccessToken: "access", Character: c, CharacterID: c.ID, ExpiresAt: time.Now(), RefreshToken: "refresh", TokenType: "xxx"}
	// when
	err := o.Save()
	// then
	assert.NoError(t, err)
	r, err := model.FetchToken(c.ID)
	if assert.NoError(t, err) {
		assert.Equal(t, o.AccessToken, r.AccessToken)
		assert.Equal(t, o.CharacterID, r.CharacterID)
		assert.Equal(t, o.Character.ID, r.Character.ID)
	}
}

func TestTokenSaveReturnErrorWhenNoCharacter(t *testing.T) {
	// given
	model.TruncateTables()
	o := model.Token{AccessToken: "access", ExpiresAt: time.Now(), RefreshToken: "refresh", TokenType: "xxx"}
	// when
	err := o.Save()
	// then
	assert.Error(t, err)

}

func TestTokenCanUpdate(t *testing.T) {
	// given
	model.TruncateTables()
	factory.CreateCharacter()
	o := factory.CreateToken()
	assert.NoError(t, o.Save())
	o.AccessToken = "new-access"
	// when
	err := o.Save()
	assert.NoError(t, err)
	r, err := model.FetchToken(o.CharacterID)
	// then
	if assert.NoError(t, err) {
		assert.Equal(t, o.AccessToken, r.AccessToken)
		assert.Equal(t, o.CharacterID, r.CharacterID)
		assert.Equal(t, o.Character.ID, r.Character.ID)
	}
}

func TestTokenCanFetchByID(t *testing.T) {
	// given
	model.TruncateTables()
	factory.CreateToken()
	o := factory.CreateToken()
	// when
	r, err := model.FetchToken(o.CharacterID)
	// then
	if assert.NoError(t, err) {
		assert.Equal(t, o.AccessToken, r.AccessToken)
		assert.Equal(t, o.CharacterID, r.CharacterID)
		assert.Equal(t, o.Character.ID, r.Character.ID)
	}
}
