package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func TestCharacterSkill(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterFull()
		eveType := factory.CreateEveType()
		arg := storage.UpdateOrCreateCharacterSkillParams{
			ActiveSkillLevel:   3,
			EveTypeID:          eveType.ID,
			CharacterID:        c.ID,
			SkillPointsInSkill: 99,
			TrainedSkillLevel:  5,
		}
		// when
		err := st.UpdateOrCreateCharacterSkill(ctx, arg)
		// then
		if assert.NoError(t, err) {
			x, err := st.GetCharacterSkill(ctx, c.ID, arg.EveTypeID)
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
		c := factory.CreateCharacterFull()
		o1 := factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:        c.ID,
			ActiveSkillLevel:   3,
			TrainedSkillLevel:  5,
			SkillPointsInSkill: 42,
		})
		arg := storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:        c.ID,
			EveTypeID:          o1.EveType.ID,
			ActiveSkillLevel:   4,
			TrainedSkillLevel:  4,
			SkillPointsInSkill: 99,
		}
		// when
		err := st.UpdateOrCreateCharacterSkill(ctx, arg)
		// then
		if assert.NoError(t, err) {
			o2, err := st.GetCharacterSkill(ctx, c.ID, o1.EveType.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, 4, o2.ActiveSkillLevel)
				assert.Equal(t, 4, o2.TrainedSkillLevel)
				assert.Equal(t, 99, o2.SkillPointsInSkill)
			}
		}
	})
	t.Run("can list skill IDs", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterFull()
		o1 := factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID: c.ID,
		})
		o2 := factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID: c.ID,
		})
		// when
		ids, err := st.ListCharacterSkillIDs(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.ElementsMatch(t, []int32{o1.EveType.ID, o2.EveType.ID}, ids.Slice())
		}
	})
	t.Run("can delete excluded skills", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterFull()
		x1 := factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{CharacterID: c.ID})
		x2 := factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{CharacterID: c.ID})
		// when
		err := st.DeleteCharacterSkills(ctx, c.ID, []int32{x2.EveType.ID})
		// then
		if assert.NoError(t, err) {
			ids, err := st.ListCharacterSkillIDs(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.ElementsMatch(t, []int32{x1.EveType.ID}, ids.Slice())
			}
		}
	})
}

func TestCharacterSkillLists(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("should return list of skill groups with progress", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterFull()
		category := factory.CreateEveCategory(storage.CreateEveCategoryParams{ID: app.EveCategorySkill})
		group := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: category.ID, IsPublished: true})
		myType := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID, IsPublished: true})
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:       c.ID,
			EveTypeID:         myType.ID,
			TrainedSkillLevel: 5,
		})
		// when
		xx, err := st.ListCharacterSkillGroupsProgress(ctx, c.ID)
		if assert.NoError(t, err) {
			assert.Len(t, xx, 1)
		}
	})
	t.Run("should return list of skill groups with progress", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterFull()
		category := factory.CreateEveCategory(storage.CreateEveCategoryParams{ID: app.EveCategorySkill})
		group := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: category.ID, IsPublished: true})
		myType := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID, IsPublished: true})
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:       c.ID,
			EveTypeID:         myType.ID,
			TrainedSkillLevel: 5,
		})
		// when
		xx, err := st.ListCharacterSkillProgress(ctx, c.ID, group.ID)
		if assert.NoError(t, err) {
			assert.Len(t, xx, 1)
		}
	})
}

func TestListCharactersActiveSkillLevels(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("returns skill level", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCharacterFull()
		c2 := factory.CreateCharacterFull()
		c3 := factory.CreateCharacterFull()
		skill1 := factory.CreateEveType()
		skill2 := factory.CreateEveType()
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:       c1.ID,
			EveTypeID:         skill1.ID,
			ActiveSkillLevel:  3,
			TrainedSkillLevel: 5,
		})
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:       c2.ID,
			EveTypeID:         skill1.ID,
			ActiveSkillLevel:  4,
			TrainedSkillLevel: 5,
		})
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:       c1.ID,
			EveTypeID:         skill2.ID,
			ActiveSkillLevel:  5,
			TrainedSkillLevel: 5,
		})
		// when
		got, err := st.ListAllCharactersActiveSkillLevels(ctx, skill1.ID)
		if assert.NoError(t, err) {
			want := []app.CharacterActiveSkillLevel{
				{
					CharacterID: c1.ID,
					TypeID:      skill1.ID,
					Level:       3,
				},
				{
					CharacterID: c2.ID,
					TypeID:      skill1.ID,
					Level:       4,
				},
				{
					CharacterID: c3.ID,
					TypeID:      skill1.ID,
					Level:       0,
				},
			}
			assert.ElementsMatch(t, want, got)
		}
	})
}
