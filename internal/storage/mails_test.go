package storage_test

import (
	"example/esiapp/internal/set"
	"example/esiapp/internal/storage"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMailCanSaveNew(t *testing.T) {
	// given
	storage.TruncateTables()
	char := createCharacter(1, "Erik")
	from := createEveEntity(EveEntityArgs{})
	m := storage.Mail{
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
	r, err := storage.FetchMail(m.ID)
	assert.NoError(t, err)
	assert.Equal(t, m.MailID, r.MailID)
}

func TestMailCanUpdateExisting(t *testing.T) {
	// given
	storage.TruncateTables()
	char := createCharacter(1, "Erik")
	from := createEveEntity(EveEntityArgs{})
	m := storage.Mail{
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
	r, err := storage.FetchMail(m.ID)
	assert.NoError(t, err)
	assert.Equal(t, m.MailID, r.MailID)
	assert.Equal(t, m.Subject, r.Subject)
}

func TestMailCanFetchMailIDs(t *testing.T) {
	// given
	storage.TruncateTables()
	char := createCharacter(7, "Erik")
	from := createEveEntity(EveEntityArgs{})
	for i := range 3 {
		m := storage.Mail{
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
	ids, err := storage.FetchMailIDs(7)
	assert.NoError(t, err)
	got := set.NewFromSlice(ids)
	want := set.NewFromSlice([]int32{10, 11, 12})
	assert.Equal(t, want, got)
}
