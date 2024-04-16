package repository_test

import (
	"context"
	"example/evebuddy/internal/repository"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCharacter(t *testing.T) {
	db, r, factory := setUpDB()
	defer db.Close()
	ctx := context.Background()
	// t.Run("can create new", func(t *testing.T) {
	// 	// given
	// 	repository.TruncateTables(db)
	// 	corp := factory.CreateEveEntityCorporation()
	// 	c := repository.Character{ID: 1, Name: "Erik", Corporation: corp}
	// 	// when
	// 	err := c.Save()
	// 	// then
	// 	if assert.NoError(t, err) {
	// 		r, err := model.GetCharacter(c.ID)
	// 		if assert.NoError(t, err) {
	// 			assert.Equal(t, c, r)
	// 		}
	// 	}
	// })
	t.Run("can list characters", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		factory.CreateCharacter()
		factory.CreateCharacter()
		// when
		cc, err := r.ListCharacters(ctx)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, cc, 2)
		}
	})

}
