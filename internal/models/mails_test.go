package models_test

import (
	"example/esiapp/internal/models"
	"example/esiapp/internal/set"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func createMail(args ...models.Mail) models.Mail {
	var m models.Mail
	if len(args) > 0 {
		m = args[0]
	}
	if m.Character.ID == 0 {
		m.Character = createCharacter()
	}
	if m.From.ID == 0 {
		m.From = createEveEntity(models.EveEntity{Category: models.EveEntityCharacter})
	}
	if m.MailID == 0 {
		ids, err := models.FetchMailIDs(m.Character.ID)
		if err != nil {
			panic(err)
		}
		if len(ids) > 0 {
			m.MailID = slices.Max(ids) + 1
		} else {
			m.MailID = 1
		}
	}
	if m.Body == "" {
		m.Body = fmt.Sprintf("Generated body #%d", m.MailID)
	}
	if m.Subject == "" {
		m.Body = fmt.Sprintf("Generated subject #%d", m.MailID)
	}
	if m.Timestamp.IsZero() {
		m.Timestamp = time.Now()
	}
	if err := m.Save(); err != nil {
		panic(err)
	}
	return m
}

func TestMailCanSaveNew(t *testing.T) {
	// given
	models.TruncateTables()
	char := createCharacter()
	from := createEveEntity()
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

func TestMailSaveShouldReturnErrorWhenCharacterIDMissing(t *testing.T) {
	// given
	models.TruncateTables()
	from := createEveEntity()
	m := models.Mail{
		Body:      "body",
		From:      from,
		MailID:    7,
		Subject:   "subject",
		Timestamp: time.Now(),
	}
	// when
	err := m.Save()
	// then
	assert.Error(t, err)
}

func TestMailSaveShouldReturnErrorWhenFromIDMissing(t *testing.T) {
	// given
	models.TruncateTables()
	c := createCharacter()
	m := models.Mail{
		Body:      "body",
		Character: c,
		MailID:    7,
		Subject:   "subject",
		Timestamp: time.Now(),
	}
	// when
	err := m.Save()
	// then
	assert.Error(t, err)
}
func TestMailCanUpdateExisting(t *testing.T) {
	// given
	models.TruncateTables()
	m := createMail()
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
	for i := range 3 {
		createMail(models.Mail{
			Character: char,
			MailID:    int32(10 + i),
		})
	}
	// when
	ids, err := models.FetchMailIDs(char.ID)
	assert.NoError(t, err)
	got := set.NewFromSlice(ids)
	want := set.NewFromSlice([]int32{10, 11, 12})
	assert.Equal(t, want, got)
}
