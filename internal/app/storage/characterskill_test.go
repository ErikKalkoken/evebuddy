package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

func TestCharacterSkill(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		eveType := factory.CreateEveType()
		arg := storage.UpdateOrCreateCharacterSkillParams{
			ActiveSkillLevel:   3,
			TypeID:             eveType.ID,
			CharacterID:        c.ID,
			SkillPointsInSkill: 99,
			TrainedSkillLevel:  5,
		}
		// when
		err := st.UpdateOrCreateCharacterSkill(ctx, arg)
		// then
		require.NoError(t, err)
		x, err := st.GetCharacterSkill(ctx, c.ID, arg.TypeID)
		require.NoError(t, err)
		xassert.Equal(t, 3, x.ActiveSkillLevel)
		xassert.Equal(t, eveType, x.Type)
		xassert.Equal(t, 99, x.SkillPointsInSkill)
		xassert.Equal(t, 5, x.TrainedSkillLevel)
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		o1 := factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:        c.ID,
			ActiveSkillLevel:   3,
			TrainedSkillLevel:  5,
			SkillPointsInSkill: 42,
		})
		arg := storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:        c.ID,
			TypeID:             o1.Type.ID,
			ActiveSkillLevel:   4,
			TrainedSkillLevel:  4,
			SkillPointsInSkill: 99,
		}
		// when
		err := st.UpdateOrCreateCharacterSkill(ctx, arg)
		// then
		require.NoError(t, err)
		o2, err := st.GetCharacterSkill(ctx, c.ID, o1.Type.ID)
		require.NoError(t, err)
		xassert.Equal(t, 4, o2.ActiveSkillLevel)
		xassert.Equal(t, 4, o2.TrainedSkillLevel)
		xassert.Equal(t, 99, o2.SkillPointsInSkill)
	})
	t.Run("can delete excluded skills", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		x1 := factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{CharacterID: c.ID})
		x2 := factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{CharacterID: c.ID})
		// when
		err := st.DeleteCharacterSkills(ctx, c.ID, set.Of(x2.Type.ID))
		// then
		require.NoError(t, err)
		ids, err := st.ListCharacterSkillIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(x1.Type.ID), ids)
	})
}

func TestCharacterSkillLists(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can list skill IDs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
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
		require.NoError(t, err)
		xassert.Equal(t, set.Of(o1.Type.ID, o2.Type.ID), ids)
	})
	t.Run("can list skills", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		category := factory.CreateEveCategory(storage.CreateEveCategoryParams{ID: app.EveCategorySkill})
		group := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: category.ID, IsPublished: true})
		skill1 := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID, IsPublished: true})
		o1 := factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID: c.ID,
			TypeID:      skill1.ID,
		})
		// when
		oo, err := st.ListCharacterSkills(t.Context(), c.ID)
		// then
		require.NoError(t, err)
		want := set.Of(o1.Type.ID)
		got := set.Collect(xiter.MapSlice(oo, func(x *app.CharacterSkill) int64 {
			return x.Type.ID
		}))
		xassert.Equal(t, want, got)
	})
}

func TestListCharactersActiveSkillLevels(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("returns skill level", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateCharacterFull()
		c2 := factory.CreateCharacterFull()
		c3 := factory.CreateCharacterFull()
		skill1 := factory.CreateEveType()
		skill2 := factory.CreateEveType()
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:       c1.ID,
			TypeID:            skill1.ID,
			ActiveSkillLevel:  3,
			TrainedSkillLevel: 5,
		})
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:       c2.ID,
			TypeID:            skill1.ID,
			ActiveSkillLevel:  4,
			TrainedSkillLevel: 5,
		})
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:       c1.ID,
			TypeID:            skill2.ID,
			ActiveSkillLevel:  5,
			TrainedSkillLevel: 5,
		})
		// when
		got, err := st.ListAllCharactersActiveSkillLevels(ctx, skill1.ID)
		// then
		require.NoError(t, err)
		want := []app.CharacterActiveSkillLevel{{
			CharacterID: c1.ID,
			TypeID:      skill1.ID,
			Level:       3,
		}, {
			CharacterID: c2.ID,
			TypeID:      skill1.ID,
			Level:       4,
		}, {
			CharacterID: c3.ID,
			TypeID:      skill1.ID,
			Level:       0,
		}}
		assert.ElementsMatch(t, want, got)
	})
}
