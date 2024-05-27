package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func TestShipSkills(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		category := factory.CreateEveCategory(storage.CreateEveCategoryParams{ID: model.EveCategoryIDShip})
		group := factory.CreateEveGroup(storage.CreateEveGroupParams{
			CategoryID: category.ID,
		})
		ship := factory.CreateEveType(storage.CreateEveTypeParams{
			GroupID:     group.ID,
			IsPublished: true,
		})
		skill := factory.CreateEveType()
		arg := storage.CreateShipSkillParams{
			Rank:        2,
			ShipTypeID:  ship.ID,
			SkillTypeID: skill.ID,
			SkillLevel:  3,
		}
		// when
		err := r.CreateShipSkill(ctx, arg)
		// then
		if assert.NoError(t, err) {
			x, err := r.GetShipSkill(ctx, ship.ID, 2)
			if assert.NoError(t, err) {
				assert.Equal(t, skill.ID, x.SkillTypeID)
				assert.Equal(t, uint(3), x.SkillLevel)
			}
		}
	})
	t.Run("can replace and create complete skill ship table", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		category := factory.CreateEveCategory(storage.CreateEveCategoryParams{ID: model.EveCategoryIDShip})
		group := factory.CreateEveGroup(storage.CreateEveGroupParams{
			CategoryID: category.ID,
		})
		ship := factory.CreateEveType(storage.CreateEveTypeParams{
			GroupID:     group.ID,
			IsPublished: true,
		})
		skill := factory.CreateEveType()
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        ship.ID,
			DogmaAttributeID: 182,
			Value:            float32(skill.ID),
		})
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        ship.ID,
			DogmaAttributeID: 277,
			Value:            float32(3),
		})
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        ship.ID,
			DogmaAttributeID: 183,
			Value:            float32(skill.ID),
		})
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        ship.ID,
			DogmaAttributeID: 278,
			Value:            float32(1),
		})
		// when
		err := r.UpdateShipSkills(ctx)
		// then
		if assert.NoError(t, err) {
			xx, err := r.ListShipSkills(ctx, ship.ID)
			if assert.NoError(t, err) {
				assert.Len(t, xx, 2)
				x := xx[0]
				assert.Equal(t, skill.ID, x.SkillTypeID)
				assert.Equal(t, uint(1), x.Rank)
				assert.Equal(t, uint(3), x.SkillLevel)
				x = xx[1]
				assert.Equal(t, skill.ID, x.SkillTypeID)
				assert.Equal(t, uint(2), x.Rank)
				assert.Equal(t, uint(1), x.SkillLevel)
			}
		}
	})
}
