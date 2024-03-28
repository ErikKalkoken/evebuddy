package model_test

import (
	"example/esiapp/internal/model"
	"testing"

	"github.com/stretchr/testify/assert"
)

// createMailLabel is a test factory for MailList objects
func createMailList(args ...model.MailList) model.MailList {
	var l model.MailList
	if len(args) > 0 {
		l = args[0]
	}
	if l.Character.ID == 0 {
		l.Character = createCharacter()
	}
	if l.EveEntity.ID == 0 {
		l.EveEntity = createEveEntity(model.EveEntity{Category: model.EveEntityMailList})
	}
	if err := l.CreateIfNew(); err != nil {
		panic(err)
	}
	return l
}

func TestMailList(t *testing.T) {
	t.Run("can create and fetch", func(t *testing.T) {
		// given
		model.TruncateTables()
		c := createCharacter()
		e := createEveEntity(model.EveEntity{Category: model.EveEntityMailList})
		l := model.MailList{
			Character: c,
			EveEntity: e,
		}
		// when
		err := l.CreateIfNew()
		// then
		if assert.NoError(t, err) {
			_, err := model.FetchMailList(c.ID, e.ID)
			assert.NoError(t, err)
		}
	})
	t.Run("can fetch all mail lists", func(t *testing.T) {
		// given
		model.TruncateTables()
		c := createCharacter()
		e1 := createEveEntity(model.EveEntity{Category: model.EveEntityMailList, Name: "alpha"})
		l1 := model.MailList{Character: c, EveEntity: e1}
		assert.NoError(t, l1.CreateIfNew())
		e2 := createEveEntity(model.EveEntity{Category: model.EveEntityMailList, Name: "bravo"})
		l2 := model.MailList{Character: c, EveEntity: e2}
		assert.NoError(t, l2.CreateIfNew())
		// when
		ll, err := model.FetchAllMailLists(c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, ll, 2)
			o := ll[0]
			assert.Equal(t, o.EveEntity.Name, "alpha")
		}
	})
}
