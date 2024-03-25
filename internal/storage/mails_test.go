package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMailCanSave(t *testing.T) {
	// given
	char := createCharacter(1, "Erik")
	from := createEveEntity(EveEntity{})
	m := Mail{
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

}
