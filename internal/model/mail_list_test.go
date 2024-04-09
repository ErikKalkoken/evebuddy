package model_test

import (
	"example/evebuddy/internal/factory"
	"example/evebuddy/internal/model"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMailList(t *testing.T) {
	t.Run("can create and fetch", func(t *testing.T) {
		// given
		model.TruncateTables()
		c := factory.CreateCharacter()
		e := factory.CreateEveEntity(model.EveEntity{Category: model.EveEntityMailList})
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
		c := factory.CreateCharacter()
		e1 := factory.CreateEveEntity(model.EveEntity{Category: model.EveEntityMailList, Name: "alpha"})
		l1 := model.MailList{Character: c, EveEntity: e1}
		assert.NoError(t, l1.CreateIfNew())
		e2 := factory.CreateEveEntity(model.EveEntity{Category: model.EveEntityMailList, Name: "bravo"})
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
