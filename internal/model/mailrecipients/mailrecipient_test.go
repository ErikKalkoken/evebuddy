package mailrecipients

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/model"
)

func TestRecipient(t *testing.T) {
	t.Run("can create from model.EveEntity", func(t *testing.T) {
		// given
		e := model.EveEntity{ID: 7, Name: "Dummy", Category: model.EveEntityCharacter}
		// when
		r := newRecipientFromEntity(&e)
		// then
		assert.Equal(t, "Dummy", r.name)
		assert.Equal(t, mailRecipientCategoryCharacter, r.category)
	})
	t.Run("can create string from full", func(t *testing.T) {
		r := recipient{name: "Erik Kalkoken", category: mailRecipientCategoryCharacter}
		assert.Equal(t, "Erik Kalkoken [Character]", r.String())
	})
	t.Run("can create string from partial", func(t *testing.T) {
		r := recipient{name: "Erik Kalkoken", category: mailRecipientCategoryUnknown}
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
			recipient{name: "Erik Kalkoken", category: mailRecipientCategoryUnknown},
		},
		{
			"can create from name 2",
			"Erik",
			recipient{name: "Erik", category: mailRecipientCategoryUnknown},
		},
		{
			"can create from name w category 1",
			"Erik Kalkoken [Character]",
			recipient{name: "Erik Kalkoken", category: mailRecipientCategoryCharacter},
		},
		{
			"can create from name w category 2",
			"Erik [Alliance]",
			recipient{name: "Erik", category: mailRecipientCategoryAlliance},
		},
		{
			"should ignore invalid text",
			"ErikKalkoken[Character]",
			recipient{name: "", category: mailRecipientCategoryUnknown},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			r := newRecipientFromText(tt.in)
			assert.Equal(t, tt.out, r)
		})
	}
}

func TestNewRecipientsFromText(t *testing.T) {
	t.Run("can create from names", func(t *testing.T) {
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
				r := NewFromText(tt.in)
				s := r.String()
				assert.Equal(t, tt.out, s)
			})
		}
	})
	t.Run("can create from empty text", func(t *testing.T) {
		r := NewFromText("")
		assert.Equal(t, r.Size(), 0)
	})
}

// func TestResolveMailRecipients(t *testing.T) {
// 	db, r, factory := testutil.New()
// 	defer db.Close()
// 	s := NewService(r)
// 	// ctx := context.Background()
// 	t.Run("should resolve to existing entities", func(t *testing.T) {
// 		testutil.TruncateTables(db)
// 		e := factory.CreateEveEntity()
// 		rr := NewMailRecipientsFromEntities([]model.EveEntity{e})
// 		mm, err := s.MailRecipientsToEveEntities(*rr)
// 		if assert.NoError(t, err) {
// 			assert.Len(t, mm, 1)
// 			assert.Equal(t, e, mm[0])
// 		}
// 	})
// t.Run("should abort with specific error when a name does not match", func(t *testing.T) {
// 	testutil.TruncateTables(db)
// 	r, err := s.buildMailRecipientsFromNames([]string{"dummy"})
// 	assert.ErrorIs(t, err, ErrNameNoMatch)
// 	assert.Nil(t, r)
// })
// t.Run("should abort with specific error when a name matches more then once", func(t *testing.T) {
// 	testutil.TruncateTables(db)
// 	factory.CreateEveEntity(model.EveEntity{Name: "alpha", Category: model.EveEntityCharacter})
// 	factory.CreateEveEntity(model.EveEntity{Name: "alpha", Category: model.EveEntityAlliance})
// 	r, err := s.buildMailRecipientsFromNames([]string{"alpha"})
// 	assert.ErrorIs(t, err, ErrNameMultipleMatches)
// 	assert.Nil(t, r)
// })
// }
