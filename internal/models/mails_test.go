package models_test

import (
	"example/esiapp/internal/models"
	"example/esiapp/internal/set"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func createMail(m models.Mail) models.Mail {
	// if m.Character.ID == 0 {
	// 	m.Character = createCharacter()
	// }
	if err := m.Save(); err != nil {
		panic(err)
	}
	return m
}

func TestMailCanSaveNew(t *testing.T) {
	// given
	models.TruncateTables()
	char := createCharacter()
	from := createEveEntity(models.EveEntity{})
	m := models.Mail{
		Body:      "body",
		Character: char,
		From:      from,
		MailID:    7,
		Subject:   "subject",
		Timestamp: time.Now(),
	}
	// when
	err := m.Save()
	// then
	assert.NoError(t, err)
	r, err := models.FetchMail(m.ID)
	assert.NoError(t, err)
	assert.Equal(t, m.MailID, r.MailID)
}

func TestMailCanUpdateExisting(t *testing.T) {
	// given
	models.TruncateTables()
	char := createCharacter()
	from := createEveEntity(models.EveEntity{})
	m := models.Mail{
		Body:      "body",
		Character: char,
		From:      from,
		MailID:    7,
		Subject:   "subject",
		Timestamp: time.Now(),
	}
	assert.NoError(t, m.Save())
	m.Subject = "other"
	// when
	err := m.Save()
	// then
	assert.NoError(t, err)
	r, err := models.FetchMail(m.ID)
	assert.NoError(t, err)
	assert.Equal(t, m.MailID, r.MailID)
	assert.Equal(t, m.Subject, r.Subject)
}

func TestMailCanFetchMailIDs(t *testing.T) {
	// given
	models.TruncateTables()
	char := createCharacter()
	from := createEveEntity(models.EveEntity{})
	for i := range 3 {
		m := models.Mail{
			Body:      "body",
			Character: char,
			From:      from,
			MailID:    int32(10 + i),
			Subject:   "subject",
			Timestamp: time.Now(),
		}
		err := m.Save()
		assert.NoError(t, err)
	}
	// when
	ids, err := models.FetchMailIDs(char.ID)
	assert.NoError(t, err)
	got := set.NewFromSlice(ids)
	want := set.NewFromSlice([]int32{10, 11, 12})
	assert.Equal(t, want, got)
}
