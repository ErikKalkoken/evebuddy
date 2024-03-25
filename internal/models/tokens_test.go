package models_test

import (
	"example/esiapp/internal/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTokenCanSaveNew(t *testing.T) {
	// given
	models.TruncateTables()
	c := createCharacter(1, "Erik")
	o := models.Token{AccessToken: "access", Character: c, CharacterID: c.ID, ExpiresAt: time.Now(), RefreshToken: "refresh", TokenType: "xxx"}
	// when
	err := o.Save()
	assert.NoError(t, err)
	r, err := models.FetchToken(1)
	// then
	if assert.NoError(t, err) {
		assert.Equal(t, o.AccessToken, r.AccessToken)
		assert.Equal(t, o.CharacterID, r.CharacterID)
		assert.Equal(t, o.Character.ID, r.Character.ID)
	}
}

func TestTokenCanUpdate(t *testing.T) {
	// given
	models.TruncateTables()
	createCharacter(2, "Naoko")
	c := createCharacter(1, "Erik")
	o := models.Token{AccessToken: "access", Character: c, CharacterID: c.ID, ExpiresAt: time.Now(), RefreshToken: "refresh", TokenType: "xxx"}
	assert.NoError(t, o.Save())
	o.AccessToken = "new-access"
	// when
	err := o.Save()
	assert.NoError(t, err)
	r, err := models.FetchToken(1)
	// then
	if assert.NoError(t, err) {
		assert.Equal(t, o.AccessToken, r.AccessToken)
		assert.Equal(t, o.CharacterID, r.CharacterID)
		assert.Equal(t, o.Character.ID, r.Character.ID)
	}
}

func TestTokenCanFetchByID(t *testing.T) {
	// given
	models.TruncateTables()
	c1 := createCharacter(1, "Erik")
	o1 := models.Token{AccessToken: "one", Character: c1, CharacterID: c1.ID, ExpiresAt: time.Now(), RefreshToken: "refresh", TokenType: "xxx"}
	assert.NoError(t, o1.Save())
	c2 := createCharacter(2, "Naoko")
	o2 := models.Token{AccessToken: "two", Character: c2, CharacterID: c2.ID, ExpiresAt: time.Now(), RefreshToken: "refresh", TokenType: "xxx"}
	assert.NoError(t, o2.Save())
	// when
	r, err := models.FetchToken(2)
	// then
	if assert.NoError(t, err) {
		assert.Equal(t, o2.AccessToken, r.AccessToken)
		assert.Equal(t, o2.CharacterID, r.CharacterID)
		assert.Equal(t, o2.Character.ID, r.Character.ID)
	}
}
