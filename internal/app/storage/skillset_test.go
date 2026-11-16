package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestSkillSet(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		group := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: app.EveCategorySkill})
		skill := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID})
		// when
		id, err := st.CreateSkillSet(ctx, app.SkillSet{
			Name:         "Travel",
			Requirements: []app.SkillSetRequirement{{TypeID: skill.ID, Level: 4}},
		})
		// then
		require.NoError(t, err)
		sets, err := st.ListSkillSets(ctx)
		require.NoError(t, err)
		xassert.Equal(t, []app.SkillSet{{
			ID:           id,
			Name:         "Travel",
			Requirements: []app.SkillSetRequirement{{TypeID: skill.ID, Name: skill.Name, Level: 4}},
		}}, sets)
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		group := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: app.EveCategorySkill})
		skill1 := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID})
		skill2 := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID})
		id, err := st.CreateSkillSet(ctx, app.SkillSet{
			Name:         "Travel",
			Requirements: []app.SkillSetRequirement{{TypeID: skill1.ID, Level: 4}},
		})
		require.NoError(t, err)
		// when
		err = st.UpdateSkillSet(ctx, app.SkillSet{
			ID:           id,
			Name:         "Fast travel",
			Requirements: []app.SkillSetRequirement{{TypeID: skill2.ID, Level: 3}},
		})
		// then
		require.NoError(t, err)
		sets, err := st.ListSkillSets(ctx)
		require.NoError(t, err)
		xassert.Equal(t, []app.SkillSet{{
			ID:           id,
			Name:         "Fast travel",
			Requirements: []app.SkillSetRequirement{{TypeID: skill2.ID, Name: skill2.Name, Level: 3}},
		}}, sets)
	})
	t.Run("can list ordered by name with requirements ordered by skill name", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		group := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: app.EveCategorySkill})
		skill1 := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID, Name: "Navigation"})
		skill2 := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID, Name: "Afterburner"})
		idBravo, err := st.CreateSkillSet(ctx, app.SkillSet{
			Name:         "bravo",
			Requirements: []app.SkillSetRequirement{{TypeID: skill1.ID, Level: 1}, {TypeID: skill2.ID, Level: 2}},
		})
		require.NoError(t, err)
		idAlpha, err := st.CreateSkillSet(ctx, app.SkillSet{
			Name:         "Alpha",
			Requirements: []app.SkillSetRequirement{{TypeID: skill1.ID, Level: 3}},
		})
		require.NoError(t, err)
		// when
		sets, err := st.ListSkillSets(ctx)
		// then
		require.NoError(t, err)
		xassert.Equal(t, []app.SkillSet{
			{
				ID:           idAlpha,
				Name:         "Alpha",
				Requirements: []app.SkillSetRequirement{{TypeID: skill1.ID, Name: "Navigation", Level: 3}},
			},
			{
				ID:   idBravo,
				Name: "bravo",
				Requirements: []app.SkillSetRequirement{
					{TypeID: skill2.ID, Name: "Afterburner", Level: 2},
					{TypeID: skill1.ID, Name: "Navigation", Level: 1},
				},
			},
		}, sets)
	})
	t.Run("should return specific error when name already exists ignoring case", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		group := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: app.EveCategorySkill})
		skill := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID})
		_, err := st.CreateSkillSet(ctx, app.SkillSet{
			Name:         "Travel",
			Requirements: []app.SkillSetRequirement{{TypeID: skill.ID, Level: 4}},
		})
		require.NoError(t, err)
		// when
		_, err = st.CreateSkillSet(ctx, app.SkillSet{
			Name:         "travel",
			Requirements: []app.SkillSetRequirement{{TypeID: skill.ID, Level: 4}},
		})
		// then
		assert.ErrorIs(t, err, app.ErrAlreadyExists)
	})
	t.Run("should return not found when updating unknown skill set", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		group := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: app.EveCategorySkill})
		skill := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID})
		// when
		err := st.UpdateSkillSet(ctx, app.SkillSet{
			ID:           42,
			Name:         "Travel",
			Requirements: []app.SkillSetRequirement{{TypeID: skill.ID, Level: 4}},
		})
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
	t.Run("should return invalid when a skill is listed twice", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		group := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: app.EveCategorySkill})
		skill := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID})
		// when
		_, err := st.CreateSkillSet(ctx, app.SkillSet{
			Name:         "Travel",
			Requirements: []app.SkillSetRequirement{{TypeID: skill.ID, Level: 1}, {TypeID: skill.ID, Level: 2}},
		})
		// then
		assert.ErrorIs(t, err, app.ErrInvalid)
	})
	t.Run("should return invalid when name is empty", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		group := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: app.EveCategorySkill})
		skill := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID})
		// when
		_, err := st.CreateSkillSet(ctx, app.SkillSet{
			Name:         "",
			Requirements: []app.SkillSetRequirement{{TypeID: skill.ID, Level: 4}},
		})
		// then
		assert.ErrorIs(t, err, app.ErrInvalid)
	})
	t.Run("should return invalid when level is zero", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		group := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: app.EveCategorySkill})
		skill := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID})
		// when
		_, err := st.CreateSkillSet(ctx, app.SkillSet{
			Name:         "Travel",
			Requirements: []app.SkillSetRequirement{{TypeID: skill.ID, Level: 0}},
		})
		// then
		assert.ErrorIs(t, err, app.ErrInvalid)
	})
	t.Run("can delete with requirements", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		group := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: app.EveCategorySkill})
		skill := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID})
		id, err := st.CreateSkillSet(ctx, app.SkillSet{
			Name:         "Travel",
			Requirements: []app.SkillSetRequirement{{TypeID: skill.ID, Level: 4}},
		})
		require.NoError(t, err)
		// when
		err = st.DeleteSkillSet(ctx, id)
		// then
		require.NoError(t, err)
		sets, err := st.ListSkillSets(ctx)
		require.NoError(t, err)
		assert.Empty(t, sets)
		var count int
		require.NoError(t, db.QueryRow("SELECT count(*) FROM skill_set_requirements").Scan(&count))
		xassert.Equal(t, 0, count)
	})
	t.Run("should return not found when deleting unknown skill set", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		err := st.DeleteSkillSet(ctx, 42)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
}
