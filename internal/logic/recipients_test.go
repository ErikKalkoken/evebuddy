package logic

import (
	"example/esiapp/internal/api/esi"
	"example/esiapp/internal/factory"
	"example/esiapp/internal/model"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	// Initialize the test database for this test package
	db, err := model.InitDB(":memory:")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	os.Exit(m.Run())
}

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

func TestNewRecipientsFromText2(t *testing.T) {
	r := NewRecipientsFromText("")
	assert.Equal(t, r.Size(), 0)
}

func TestResolveLocally(t *testing.T) {
	t.Run("should resolve to existing entities", func(t *testing.T) {
		model.TruncateTables()
		e := factory.CreateEveEntity()
		r := NewRecipientsFromEntities([]model.EveEntity{e})
		mm, names, err := r.buildMailRecipients()
		if assert.NoError(t, err) {
			assert.Len(t, mm, 1)
			assert.Len(t, names, 0)
			assert.Equal(t, e.ID, mm[0].ID)
		}
	})
	t.Run("should resolve all to existing entities", func(t *testing.T) {
		model.TruncateTables()
		e1 := factory.CreateEveEntity()
		e2 := factory.CreateEveEntity()
		r := NewRecipientsFromEntities([]model.EveEntity{e1, e2})
		mm, names, err := r.buildMailRecipients()
		if assert.NoError(t, err) {
			assert.Len(t, mm, 2)
			assert.Len(t, names, 0)
		}
	})
	t.Run("should resolve to existing entities and names", func(t *testing.T) {
		model.TruncateTables()
		e1 := factory.CreateEveEntity()
		e2 := factory.CreateEveEntity()
		r := NewRecipientsFromEntities([]model.EveEntity{e1, e2})
		r.AddFromText("Other")
		mm, names, err := r.buildMailRecipients()
		if assert.NoError(t, err) {
			assert.Len(t, mm, 2)
			assert.Len(t, names, 1)
		}
	})
	t.Run("should resolve to names", func(t *testing.T) {
		model.TruncateTables()
		r := NewRecipientsFromText("Other")
		mm, names, err := r.buildMailRecipients()
		if assert.NoError(t, err) {
			assert.Len(t, mm, 0)
			assert.Len(t, names, 1)
		}
	})
}

func TestBuildMailRecipientsCategories(t *testing.T) {
	var cases = []struct {
		in  model.EveEntityCategory
		out esi.MailRecipientType
	}{
		{model.EveEntityAlliance, esi.MailRecipientTypeAlliance},
		{model.EveEntityCharacter, esi.MailRecipientTypeCharacter},
		{model.EveEntityCorporation, esi.MailRecipientTypeCorporation},
		{model.EveEntityMailList, esi.MailRecipientTypeMailingList},
	}
	for _, tc := range cases {
		model.TruncateTables()
		t.Run(fmt.Sprintf("category %s", tc.in), func(t *testing.T) {
			e := factory.CreateEveEntity(model.EveEntity{Category: tc.in})
			r := NewRecipientsFromEntities([]model.EveEntity{e})
			mm, names, err := r.buildMailRecipients()
			if assert.NoError(t, err) {
				assert.Len(t, mm, 1)
				assert.Len(t, names, 0)
				assert.Equal(t, e.ID, mm[0].ID)
				assert.Equal(t, tc.out, mm[0].Type)
			}
		})
	}
}

func TestBuildMailRecipientsFromNames(t *testing.T) {
	var cases = []struct {
		in  model.EveEntity
		out esi.MailRecipient
	}{
		{
			model.EveEntity{ID: 42, Name: "alpha", Category: model.EveEntityCharacter},
			esi.MailRecipient{ID: 42, Type: esi.MailRecipientTypeCharacter},
		},
		{
			model.EveEntity{ID: 42, Name: "alpha", Category: model.EveEntityCorporation},
			esi.MailRecipient{ID: 42, Type: esi.MailRecipientTypeCorporation},
		},
		{
			model.EveEntity{ID: 42, Name: "alpha", Category: model.EveEntityAlliance},
			esi.MailRecipient{ID: 42, Type: esi.MailRecipientTypeAlliance},
		},
		{
			model.EveEntity{ID: 42, Name: "alpha", Category: model.EveEntityMailList},
			esi.MailRecipient{ID: 42, Type: esi.MailRecipientTypeMailingList},
		},
	}
	for _, tc := range cases {
		t.Run("should return mail recipient when match is found", func(t *testing.T) {
			model.TruncateTables()
			factory.CreateEveEntity(tc.in)
			rr, err := buildMailRecipientsFromNames([]string{tc.in.Name})
			if assert.NoError(t, err) {
				assert.Len(t, rr, 1)
				assert.Equal(t, rr[0], tc.out)
			}
		})
	}
	t.Run("should abort with specific error when a name does not match", func(t *testing.T) {
		model.TruncateTables()
		r, err := buildMailRecipientsFromNames([]string{"dummy"})
		assert.ErrorIs(t, err, ErrNameNoMatch)
		assert.Nil(t, r)
	})
	t.Run("should abort with specific error when a name matches more then once", func(t *testing.T) {
		model.TruncateTables()
		factory.CreateEveEntity(model.EveEntity{Name: "alpha", Category: model.EveEntityCharacter})
		factory.CreateEveEntity(model.EveEntity{Name: "alpha", Category: model.EveEntityAlliance})
		r, err := buildMailRecipientsFromNames([]string{"alpha"})
		assert.ErrorIs(t, err, ErrNameMultipleMatches)
		assert.Nil(t, r)
	})
}