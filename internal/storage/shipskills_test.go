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
		ship_1 := factory.CreateEveType(storage.CreateEveTypeParams{
			GroupID:     group.ID,
			IsPublished: true,
		})
		skill_11 := factory.CreateEveType()
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        ship_1.ID,
			DogmaAttributeID: model.EveDogmaAttributeIDPrimarySkillID,
			Value:            float32(skill_11.ID),
		})
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        ship_1.ID,
			DogmaAttributeID: model.EveDogmaAttributeIDPrimarySkillLevel,
			Value:            float32(1),
		})
		ship_2 := factory.CreateEveType(storage.CreateEveTypeParams{
			GroupID:     group.ID,
			IsPublished: true,
		})
		skill_21 := factory.CreateEveType()
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        ship_2.ID,
			DogmaAttributeID: model.EveDogmaAttributeIDPrimarySkillID,
			Value:            float32(skill_21.ID),
		})
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        ship_2.ID,
			DogmaAttributeID: model.EveDogmaAttributeIDPrimarySkillLevel,
			Value:            float32(1),
		})
		skill_22 := factory.CreateEveType()
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        ship_2.ID,
			DogmaAttributeID: model.EveDogmaAttributeIDSecondarySkillID,
			Value:            float32(skill_22.ID),
		})
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        ship_2.ID,
			DogmaAttributeID: model.EveDogmaAttributeIDSecondarySkillLevel,
			Value:            float32(2),
		})
		skill_23 := factory.CreateEveType()
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        ship_2.ID,
			DogmaAttributeID: model.EveDogmaAttributeIDTertiarySkillID,
			Value:            float32(skill_23.ID),
		})
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        ship_2.ID,
			DogmaAttributeID: model.EveDogmaAttributeIDTertiarySkillLevel,
			Value:            float32(3),
		})
		skill_24 := factory.CreateEveType()
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        ship_2.ID,
			DogmaAttributeID: model.EveDogmaAttributeIDQuaternarySkillID,
			Value:            float32(skill_24.ID),
		})
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        ship_2.ID,
			DogmaAttributeID: model.EveDogmaAttributeIDQuaternarySkillLevel,
			Value:            float32(4),
		})
		skill_25 := factory.CreateEveType()
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        ship_2.ID,
			DogmaAttributeID: model.EveDogmaAttributeIDQuinarySkillID,
			Value:            float32(skill_25.ID),
		})
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        ship_2.ID,
			DogmaAttributeID: model.EveDogmaAttributeIDQuinarySkillLevel,
			Value:            float32(5),
		})
		skill_26 := factory.CreateEveType()
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        ship_2.ID,
			DogmaAttributeID: model.EveDogmaAttributeIDSenarySkillID,
			Value:            float32(skill_26.ID),
		})
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        ship_2.ID,
			DogmaAttributeID: model.EveDogmaAttributeIDSenarySkillLevel,
			Value:            float32(3),
		})
		// when
		err := r.UpdateShipSkills(ctx)
		// then
		if assert.NoError(t, err) {
			xx, err := r.ListShipSkills(ctx, ship_1.ID)
			if assert.NoError(t, err) {
				if assert.Len(t, xx, 1) {
					x := xx[0]
					assert.Equal(t, skill_11.ID, x.SkillTypeID)
					assert.Equal(t, uint(1), x.Rank)
					assert.Equal(t, uint(1), x.SkillLevel)
				}
			}
			xx, err = r.ListShipSkills(ctx, ship_2.ID)
			if assert.NoError(t, err) {
				if assert.Len(t, xx, 6) {
					x := xx[0]
					assert.Equal(t, skill_21.ID, x.SkillTypeID)
					assert.Equal(t, uint(1), x.Rank)
					assert.Equal(t, uint(1), x.SkillLevel)
					x = xx[1]
					assert.Equal(t, skill_22.ID, x.SkillTypeID)
					assert.Equal(t, uint(2), x.Rank)
					assert.Equal(t, uint(2), x.SkillLevel)
					x = xx[2]
					assert.Equal(t, skill_23.ID, x.SkillTypeID)
					assert.Equal(t, uint(3), x.Rank)
					assert.Equal(t, uint(3), x.SkillLevel)
					x = xx[3]
					assert.Equal(t, skill_24.ID, x.SkillTypeID)
					assert.Equal(t, uint(4), x.Rank)
					assert.Equal(t, uint(4), x.SkillLevel)
					x = xx[4]
					assert.Equal(t, skill_25.ID, x.SkillTypeID)
					assert.Equal(t, uint(5), x.Rank)
					assert.Equal(t, uint(5), x.SkillLevel)
					x = xx[5]
					assert.Equal(t, skill_26.ID, x.SkillTypeID)
					assert.Equal(t, uint(6), x.Rank)
					assert.Equal(t, uint(3), x.SkillLevel)
				}
			}
		}
	})
}
