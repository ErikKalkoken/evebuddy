package storage_test

import (
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
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()

	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCharacter()
		eveType := f.CreateEveType()
		arg := storage.UpdateOrCreateCharacterSkillParams{
			ActiveSkillLevel:   3,
			TypeID:             eveType.ID,
			CharacterID:        c.ID,
			SkillPointsInSkill: 99,
			TrainedSkillLevel:  5,
		}
		// when
		err := st.UpdateOrCreateCharacterSkill(t.Context(), arg)
		// then
		require.NoError(t, err)
		x, err := st.GetCharacterSkill(t.Context(), c.ID, arg.TypeID)
		require.NoError(t, err)
		xassert.Equal(t, 3, x.ActiveSkillLevel)
		xassert.Equal(t, eveType, x.Type)
		xassert.Equal(t, 99, x.SkillPointsInSkill)
		xassert.Equal(t, 5, x.TrainedSkillLevel)
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCharacter()
		o1 := f.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
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
		err := st.UpdateOrCreateCharacterSkill(t.Context(), arg)
		// then
		require.NoError(t, err)
		o2, err := st.GetCharacterSkill(t.Context(), c.ID, o1.Type.ID)
		require.NoError(t, err)
		xassert.Equal(t, 4, o2.ActiveSkillLevel)
		xassert.Equal(t, 4, o2.TrainedSkillLevel)
		xassert.Equal(t, 99, o2.SkillPointsInSkill)
	})
	t.Run("can delete excluded skills", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCharacter()
		x1 := f.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{CharacterID: c.ID})
		x2 := f.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{CharacterID: c.ID})
		// when
		err := st.DeleteCharacterSkills(t.Context(), c.ID, set.Of(x2.Type.ID))
		// then
		require.NoError(t, err)
		ids, err := st.ListCharacterSkillIDs(t.Context(), c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(x1.Type.ID), ids)
	})
}

func TestCharacterSkillLists(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()

	t.Run("can list skill IDs of a character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCharacter()
		o1 := f.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID: c.ID,
		})
		o2 := f.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID: c.ID,
		})

		// when
		ids, err := st.ListCharacterSkillIDs(t.Context(), c.ID)

		// then
		require.NoError(t, err)
		xassert.Equal(t, set.Of(o1.Type.ID, o2.Type.ID), ids)
	})

	t.Run("can list skills of a character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCharacter()
		o1 := f.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID: c.ID,
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

	t.Run("can list all skills", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		o1 := f.CreateCharacterSkill()
		o2 := f.CreateCharacterSkill()

		// when
		oo, err := st.ListAllCharacterSkills(t.Context())

		// then
		require.NoError(t, err)
		want := set.Of(o1.Type.ID, o2.Type.ID)
		got := set.Collect(xiter.MapSlice(oo, func(x *app.CharacterSkill) int64 {
			return x.Type.ID
		}))
		xassert.Equal(t, want, got)
	})
}

func TestListCharactersActiveSkillLevels(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()

	t.Run("returns skill level", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := f.CreateCharacter()
		c2 := f.CreateCharacter()
		c3 := f.CreateCharacter()
		skill1 := f.CreateEveType()
		skill2 := f.CreateEveType()
		f.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:       c1.ID,
			TypeID:            skill1.ID,
			ActiveSkillLevel:  3,
			TrainedSkillLevel: 5,
		})
		f.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:       c2.ID,
			TypeID:            skill1.ID,
			ActiveSkillLevel:  4,
			TrainedSkillLevel: 5,
		})
		f.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:       c1.ID,
			TypeID:            skill2.ID,
			ActiveSkillLevel:  5,
			TrainedSkillLevel: 5,
		})
		// when
		got, err := st.ListAllCharactersActiveSkillLevels(t.Context(), skill1.ID)
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
