package storage_test

import (
	"example/esiapp/internal/storage"
	"testing"
)

func TestMailLabelSaveNew(t *testing.T) {
	// given
	storage.TruncateTables()
	// char := createCharacter(1, "Erik")
	// from := createEveEntity(EveEntityArgs{})
	// m := storage.MailLabel{
	// 	Character: char,
	// 	From:      from,
	// 	MailID:    7,
	// 	Subject:   "subject",
	// 	Timestamp: time.Now(),
	// }
	// // when
	// err := m.Save()
	// // then
	// assert.NoError(t, err)
	// r, err := storage.FetchMail(m.ID)
	// assert.NoError(t, err)
	// assert.Equal(t, m.MailID, r.MailID)
}
