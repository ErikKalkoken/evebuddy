package models_test

import (
	"example/esiapp/internal/models"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMailLabelSaveNew(t *testing.T) {
	// given
	models.TruncateTables()
	c := createCharacter()
	l := models.MailLabel{
		Character:   c,
		Color:       "xyz",
		LabelID:     1,
		Name:        "Dummy",
		UnreadCount: 42,
	}
	// when
	err := l.Save()
	// then
	if assert.NoError(t, err) {
		l2, err := models.FetchMailLabel(c.ID, l.LabelID)
		if assert.NoError(t, err) {
			assert.Equal(t, l.Name, l2.Name)
		}
	}
}
