package model_test

import (
	"example/esiapp/internal/model"
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func createToken(args ...model.Token) model.Token {
	var t model.Token
	if len(args) > 0 {
		t = args[0]
	}
	if t.Character.ID == 0 || t.CharacterID == 0 {
		t.Character = createCharacter()
	}
	if t.AccessToken == "" {
		t.AccessToken = fmt.Sprintf("GeneratedAccessToken#%d", rand.IntN(1000000))
	}
	if t.RefreshToken == "" {
		t.RefreshToken = fmt.Sprintf("GeneratedRefreshToken#%d", rand.IntN(1000000))
	}
	if t.ExpiresAt.IsZero() {
		t.ExpiresAt = time.Now().Add(time.Minute * 20)
	}
	if t.TokenType == "" {
		t.TokenType = "Bearer"
	}
	err := t.Save()
	if err != nil {
		panic(err)
	}
	return t
}

func TestTokenCanSaveNew(t *testing.T) {
	// given
	model.TruncateTables()
	c := createCharacter()
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
	createCharacter()
	o := createToken()
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
	createToken()
	o := createToken()
	// when
	r, err := model.FetchToken(o.CharacterID)
	// then
	if assert.NoError(t, err) {
		assert.Equal(t, o.AccessToken, r.AccessToken)
		assert.Equal(t, o.CharacterID, r.CharacterID)
		assert.Equal(t, o.Character.ID, r.Character.ID)
	}
}