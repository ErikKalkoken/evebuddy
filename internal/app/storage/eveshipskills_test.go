package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func TestEveShipSkills(t *testing.T) {
	db, r, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		category := factory.CreateEveCategory(storage.CreateEveCategoryParams{ID: app.EveCategoryShip})
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
		err := r.CreateEveShipSkill(ctx, arg)
		// then
		if assert.NoError(t, err) {
			x, err := r.GetEveShipSkill(ctx, ship.ID, 2)
			if assert.NoError(t, err) {
				assert.Equal(t, skill.ID, x.SkillTypeID)
				assert.Equal(t, uint(3), x.SkillLevel)
			}
		}
	})
	t.Run("can replace and create complete skill ship table", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		category := factory.CreateEveCategory(storage.CreateEveCategoryParams{ID: app.EveCategoryShip})
		group := factory.CreateEveGroup(storage.CreateEveGroupParams{
			CategoryID: category.ID,
		})
		ship_1 := factory.CreateEveType(storage.CreateEveTypeParams{
			GroupID:     group.ID,
			IsPublished: true,
		})
		skill_11 := factory.CreateEveType()
		primarySkillID := factory.CreateEveDogmaAttribute(storage.CreateEveDogmaAttributeParams{
			ID: app.EveDogmaAttributePrimarySkillID,
		})
		primarySkillLevel := factory.CreateEveDogmaAttribute(storage.CreateEveDogmaAttributeParams{
			ID: app.EveDogmaAttributePrimarySkillLevel,
		})
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        ship_1.ID,
			DogmaAttributeID: primarySkillID.ID,
			Value:            float32(skill_11.ID),
		})
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        ship_1.ID,
			DogmaAttributeID: primarySkillLevel.ID,
			Value:            float32(1),
		})
		ship_2 := factory.CreateEveType(storage.CreateEveTypeParams{
			GroupID:     group.ID,
			IsPublished: true,
		})
		skill_21 := factory.CreateEveType()
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        ship_2.ID,
			DogmaAttributeID: primarySkillID.ID,
			Value:            float32(skill_21.ID),
		})
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        ship_2.ID,
			DogmaAttributeID: primarySkillLevel.ID,
			Value:            float32(1),
		})
		skill_22 := factory.CreateEveType()
		secondarySkillID := factory.CreateEveDogmaAttribute(storage.CreateEveDogmaAttributeParams{
			ID: app.EveDogmaAttributeSecondarySkillID,
		})
		secondarySkillLevel := factory.CreateEveDogmaAttribute(storage.CreateEveDogmaAttributeParams{
			ID: app.EveDogmaAttributeSecondarySkillLevel,
		})
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        ship_2.ID,
			DogmaAttributeID: secondarySkillID.ID,
			Value:            float32(skill_22.ID),
		})
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        ship_2.ID,
			DogmaAttributeID: secondarySkillLevel.ID,
			Value:            float32(2),
		})
		skill_23 := factory.CreateEveType()
		tertiarySkillID := factory.CreateEveDogmaAttribute(storage.CreateEveDogmaAttributeParams{
			ID: app.EveDogmaAttributeTertiarySkillID,
		})
		tertiarySkillLevel := factory.CreateEveDogmaAttribute(storage.CreateEveDogmaAttributeParams{
			ID: app.EveDogmaAttributeTertiarySkillLevel,
		})
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        ship_2.ID,
			DogmaAttributeID: tertiarySkillID.ID,
			Value:            float32(skill_23.ID),
		})
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        ship_2.ID,
			DogmaAttributeID: tertiarySkillLevel.ID,
			Value:            float32(3),
		})
		skill_24 := factory.CreateEveType()
		quaternarySkillID := factory.CreateEveDogmaAttribute(storage.CreateEveDogmaAttributeParams{
			ID: app.EveDogmaAttributeQuaternarySkillID,
		})
		quaternarySkillLevel := factory.CreateEveDogmaAttribute(storage.CreateEveDogmaAttributeParams{
			ID: app.EveDogmaAttributeQuaternarySkillLevel,
		})
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        ship_2.ID,
			DogmaAttributeID: quaternarySkillID.ID,
			Value:            float32(skill_24.ID),
		})
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        ship_2.ID,
			DogmaAttributeID: quaternarySkillLevel.ID,
			Value:            float32(4),
		})
		skill_25 := factory.CreateEveType()
		quinarySkillID := factory.CreateEveDogmaAttribute(storage.CreateEveDogmaAttributeParams{
			ID: app.EveDogmaAttributeQuinarySkillID,
		})
		quinarySkillLevel := factory.CreateEveDogmaAttribute(storage.CreateEveDogmaAttributeParams{
			ID: app.EveDogmaAttributeQuinarySkillLevel,
		})
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        ship_2.ID,
			DogmaAttributeID: quinarySkillID.ID,
			Value:            float32(skill_25.ID),
		})
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        ship_2.ID,
			DogmaAttributeID: quinarySkillLevel.ID,
			Value:            float32(5),
		})
		skill_26 := factory.CreateEveType()
		senarySkillID := factory.CreateEveDogmaAttribute(storage.CreateEveDogmaAttributeParams{
			ID: app.EveDogmaAttributeSenarySkillID,
		})
		senarySkillLevel := factory.CreateEveDogmaAttribute(storage.CreateEveDogmaAttributeParams{
			ID: app.EveDogmaAttributeSenarySkillLevel,
		})
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        ship_2.ID,
			DogmaAttributeID: senarySkillID.ID,
			Value:            float32(skill_26.ID),
		})
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        ship_2.ID,
			DogmaAttributeID: senarySkillLevel.ID,
			Value:            float32(3),
		})
		// when
		err := r.UpdateEveShipSkills(ctx)
		// then
		if assert.NoError(t, err) {
			xx, err := r.ListEveShipSkills(ctx, ship_1.ID)
			if assert.NoError(t, err) {
				if assert.Len(t, xx, 1) {
					x := xx[0]
					assert.Equal(t, skill_11.ID, x.SkillTypeID)
					assert.Equal(t, uint(1), x.Rank)
					assert.Equal(t, uint(1), x.SkillLevel)
				}
			}
			xx, err = r.ListEveShipSkills(ctx, ship_2.ID)
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
