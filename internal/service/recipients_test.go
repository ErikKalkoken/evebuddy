package service

import (
	// "example/evebuddy/internal/factory"

	"testing"

	// "github.com/antihax/goesi/esi"
	"github.com/stretchr/testify/assert"

	"example/evebuddy/internal/model"
)

// TODO: Reimplement tests

func TestRecipient(t *testing.T) {
	t.Run("can create from EveEntity", func(t *testing.T) {
		// given
		e := model.EveEntity{ID: 7, Name: "Dummy", Category: model.EveEntityCharacter}
		// when
		r := newRecipientFromEntity(e)
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
			r := newRecipientFromText(tt.in)
			assert.Equal(t, tt.out, r)
		})
	}
}

func TestNewRecipientsFromText(t *testing.T) {
	s := NewService(nil)
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
				r := s.NewRecipientsFromText(tt.in)
				s := r.String()
				assert.Equal(t, tt.out, s)
			})
		}
	})
	t.Run("can create from empty text", func(t *testing.T) {
		r := s.NewRecipientsFromText("")
		assert.Equal(t, r.Size(), 0)
	})
}

// func TestResolveLocally(t *testing.T) {
// 	s := NewService(nil)
// 	t.Run("should resolve to existing entities", func(t *testing.T) {
// 		repository.TruncateTables()
// 		e := factory.CreateEveEntity()
// 		r := s.NewRecipientsFromEntities([]EveEntity{eveEntityFromDBModel(e)})
// 		mm, names, err := r.buildMailRecipients()
// 		if assert.NoError(t, err) {
// 			assert.Len(t, mm, 1)
// 			assert.Len(t, names, 0)
// 			assert.Equal(t, e.ID, mm[0].RecipientId)
// 		}
// 	})
// 	t.Run("should resolve all to existing entities", func(t *testing.T) {
// 		repository.TruncateTables()
// 		e1 := factory.CreateEveEntity()
// 		e2 := factory.CreateEveEntity()
// 		r := s.NewRecipientsFromEntities([]EveEntity{eveEntityFromDBModel(e1), eveEntityFromDBModel(e2)})
// 		mm, names, err := r.buildMailRecipients()
// 		if assert.NoError(t, err) {
// 			assert.Len(t, mm, 2)
// 			assert.Len(t, names, 0)
// 		}
// 	})
// 	t.Run("should resolve to existing entities and names", func(t *testing.T) {
// 		repository.TruncateTables()
// 		e1 := factory.CreateEveEntity()
// 		e2 := factory.CreateEveEntity()
// 		r := s.NewRecipientsFromEntities([]EveEntity{eveEntityFromDBModel(e1), eveEntityFromDBModel(e2)})
// 		r.AddFromText("Other")
// 		mm, names, err := r.buildMailRecipients()
// 		if assert.NoError(t, err) {
// 			assert.Len(t, mm, 2)
// 			assert.Len(t, names, 1)
// 		}
// 	})
// 	t.Run("should resolve to names", func(t *testing.T) {
// 		repository.TruncateTables()
// 		r := s.NewRecipientsFromText("Other")
// 		mm, names, err := r.buildMailRecipients()
// 		if assert.NoError(t, err) {
// 			assert.Len(t, mm, 0)
// 			assert.Len(t, names, 1)
// 		}
// 	})
// }

// func TestBuildMailRecipientsCategories(t *testing.T) {
// 	s := NewService(nil)
// 	var cases = []struct {
// 		in  repository.EveEntityCategory
// 		out string
// 	}{
// 		{repository.EveEntityAlliance, "alliance"},
// 		{repository.EveEntityCharacter, "character"},
// 		{repository.EveEntityCorporation, "corporation"},
// 		{repository.EveEntityMailList, "mailing_list"},
// 	}
// 	for _, tc := range cases {
// 		repository.TruncateTables()
// 		t.Run(fmt.Sprintf("category %s", tc.in), func(t *testing.T) {
// 			e := factory.CreateEveEntity(repository.EveEntity{Category: tc.in})
// 			r := s.NewRecipientsFromEntities([]EveEntity{eveEntityFromDBModel(e)})
// 			mm, names, err := r.buildMailRecipients()
// 			if assert.NoError(t, err) {
// 				assert.Len(t, mm, 1)
// 				assert.Len(t, names, 0)
// 				assert.Equal(t, e.ID, mm[0].RecipientId)
// 				assert.Equal(t, tc.out, mm[0].RecipientType)
// 			}
// 		})
// 	}
// }

// func TestBuildMailRecipientsFromNames(t *testing.T) {
// 	s := NewService(nil)
// 	var cases = []struct {
// 		in  repository.EveEntity
// 		out esi.PostCharactersCharacterIdMailRecipient
// 	}{
// 		{
// 			repository.EveEntity{ID: 42, Name: "alpha", Category: repository.EveEntityCharacter},
// 			esi.PostCharactersCharacterIdMailRecipient{RecipientId: 42, RecipientType: "character"},
// 		},
// 		{
// 			repository.EveEntity{ID: 42, Name: "alpha", Category: repository.EveEntityCorporation},
// 			esi.PostCharactersCharacterIdMailRecipient{RecipientId: 42, RecipientType: "corporation"},
// 		},
// 		{
// 			repository.EveEntity{ID: 42, Name: "alpha", Category: repository.EveEntityAlliance},
// 			esi.PostCharactersCharacterIdMailRecipient{RecipientId: 42, RecipientType: "alliance"},
// 		},
// 		{
// 			repository.EveEntity{ID: 42, Name: "alpha", Category: repository.EveEntityMailList},
// 			esi.PostCharactersCharacterIdMailRecipient{RecipientId: 42, RecipientType: "mailing_list"},
// 		},
// 	}
// 	for _, tc := range cases {
// 		t.Run("should return mail recipient when match is found", func(t *testing.T) {
// 			repository.TruncateTables()
// 			factory.CreateEveEntity(tc.in)
// 			rr, err := s.buildMailRecipientsFromNames([]string{tc.in.Name})
// 			if assert.NoError(t, err) {
// 				assert.Len(t, rr, 1)
// 				assert.Equal(t, rr[0], tc.out)
// 			}
// 		})
// 	}
// 	t.Run("should abort with specific error when a name does not match", func(t *testing.T) {
// 		repository.TruncateTables()
// 		r, err := s.buildMailRecipientsFromNames([]string{"dummy"})
// 		assert.ErrorIs(t, err, ErrNameNoMatch)
// 		assert.Nil(t, r)
// 	})
// 	t.Run("should abort with specific error when a name matches more then once", func(t *testing.T) {
// 		repository.TruncateTables()
// 		factory.CreateEveEntity(repository.EveEntity{Name: "alpha", Category: repository.EveEntityCharacter})
// 		factory.CreateEveEntity(repository.EveEntity{Name: "alpha", Category: repository.EveEntityAlliance})
// 		r, err := s.buildMailRecipientsFromNames([]string{"alpha"})
// 		assert.ErrorIs(t, err, ErrNameMultipleMatches)
// 		assert.Nil(t, r)
// 	})
// }
