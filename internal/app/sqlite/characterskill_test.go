package sqlite_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/sqlite"
	"github.com/ErikKalkoken/evebuddy/internal/app/sqlite/testutil"
)

func TestCharacterSkill(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		eveType := factory.CreateEveType()
		arg := sqlite.UpdateOrCreateCharacterSkillParams{
			ActiveSkillLevel:   3,
			EveTypeID:          eveType.ID,
			CharacterID:        c.ID,
			SkillPointsInSkill: 99,
			TrainedSkillLevel:  5,
		}
		// when
		err := r.UpdateOrCreateCharacterSkill(ctx, arg)
		// then
		if assert.NoError(t, err) {
			x, err := r.GetCharacterSkill(ctx, c.ID, arg.EveTypeID)
			if assert.NoError(t, err) {
				assert.Equal(t, 3, x.ActiveSkillLevel)
				assert.Equal(t, eveType, x.EveType)
				assert.Equal(t, 99, x.SkillPointsInSkill)
				assert.Equal(t, 5, x.TrainedSkillLevel)
			}
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		o1 := factory.CreateCharacterSkill(sqlite.UpdateOrCreateCharacterSkillParams{
			CharacterID:        c.ID,
			ActiveSkillLevel:   3,
			TrainedSkillLevel:  5,
			SkillPointsInSkill: 42,
		})
		arg := sqlite.UpdateOrCreateCharacterSkillParams{
			CharacterID:        c.ID,
			EveTypeID:          o1.EveType.ID,
			ActiveSkillLevel:   4,
			TrainedSkillLevel:  4,
			SkillPointsInSkill: 99,
		}
		// when
		err := r.UpdateOrCreateCharacterSkill(ctx, arg)
		// then
		if assert.NoError(t, err) {
			o2, err := r.GetCharacterSkill(ctx, c.ID, o1.EveType.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, 4, o2.ActiveSkillLevel)
				assert.Equal(t, 4, o2.TrainedSkillLevel)
				assert.Equal(t, 99, o2.SkillPointsInSkill)
			}
		}
	})
	t.Run("can delete excluded skills", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		o1 := factory.CreateCharacterSkill(sqlite.UpdateOrCreateCharacterSkillParams{CharacterID: c.ID})
		o2 := factory.CreateCharacterSkill(sqlite.UpdateOrCreateCharacterSkillParams{CharacterID: c.ID})
		// when
		err := r.DeleteExcludedCharacterSkills(ctx, c.ID, []int32{o2.EveType.ID})
		// then
		if assert.NoError(t, err) {
			_, err := r.GetCharacterSkill(ctx, c.ID, o1.EveType.ID)
			assert.Error(t, err, sqlite.ErrNotFound)
			_, err = r.GetCharacterSkill(ctx, c.ID, o2.EveType.ID)
			assert.NoError(t, err)
		}
	})
}

func TestCharacterSkillList(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("should return list of skill groups with progress", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		category := factory.CreateEveCategory(sqlite.CreateEveCategoryParams{ID: app.EveCategorySkill})
		group := factory.CreateEveGroup(sqlite.CreateEveGroupParams{CategoryID: category.ID, IsPublished: true})
		myType := factory.CreateEveType(sqlite.CreateEveTypeParams{GroupID: group.ID, IsPublished: true})
		factory.CreateCharacterSkill(sqlite.UpdateOrCreateCharacterSkillParams{
			CharacterID:       c.ID,
			EveTypeID:         myType.ID,
			TrainedSkillLevel: 5,
		})
		// when
		xx, err := r.ListCharacterSkillGroupsProgress(ctx, c.ID)
		if assert.NoError(t, err) {
			assert.Len(t, xx, 1)
		}
	})
}
