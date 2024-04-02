package ui

import (
	"example/esiapp/internal/model"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecipient(t *testing.T) {
	t.Run("can create from EveEntity", func(t *testing.T) {
		// given
		e := model.EveEntity{ID: 7, Name: "Dummy", Category: model.EveEntityCharacter}
		// when
		r := NewRecipientFromEntity(e)
		// then
		assert.Equal(t, "Dummy", r.name)
		assert.Equal(t, recipientCategoryCharacter, r.category)
	})
	t.Run("can create string from full", func(t *testing.T) {
		r := recipient{name: "Erik Kalkoken", category: recipientCategoryCharacter}
		assert.Equal(t, "Erik Kalkoken [Character]", r.String())
	})
	t.Run("can create string from partial", func(t *testing.T) {
		r := recipient{name: "Erik Kalkoken", category: recipientCategoryUnknown}
		assert.Equal(t, "Erik Kalkoken", r.String())
	})

}

func TestNewRecipientFromText(t *testing.T) {
	var cases = []struct {
		name string
		in   string
		out  recipient
	}{
		{
			"can create from name 1",
			"Erik Kalkoken",
			recipient{name: "Erik Kalkoken", category: recipientCategoryUnknown},
		},
		{
			"can create from name 2",
			"Erik",
			recipient{name: "Erik", category: recipientCategoryUnknown},
		},
		{
			"can create from name w category 1",
			"Erik Kalkoken [Character]",
			recipient{name: "Erik Kalkoken", category: recipientCategoryCharacter},
		},
		{
			"can create from name w category 2",
			"Erik [Alliance]",
			recipient{name: "Erik", category: recipientCategoryAlliance},
		},
		{
			"should ignore invalid text",
			"ErikKalkoken[Character]",
			recipient{name: "", category: recipientCategoryUnknown},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRecipientFromText(tt.in)
			assert.Equal(t, tt.out, r)
		})
	}
}

func TestNewRecipientsFromText(t *testing.T) {
	var cases = []struct {
		name string
		in   string
		out  string
	}{
		{"can create from text", "Erik Kalkoken [Character]", "Erik Kalkoken [Character]"},
		{"can create from text", "Erik Kalkoken", "Erik Kalkoken"},
		{"can create from text", "", ""},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRecipientsFromText(tt.in)
			s := r.String()
			assert.Equal(t, tt.out, s)
		})
	}
}

func TestRecipients(t *testing.T) {
	r := NewRecipientsFromText("")
	assert.Equal(t, r.Size(), 0)
}
