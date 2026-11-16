package characterservice_test

import (
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestSkillSet(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})
	ctx := t.Context()
	t.Run("can create and list", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		group := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: app.EveCategorySkill})
		skill := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID})
		// when
		id, err := s.CreateSkillSet(ctx, app.SkillSet{
			Name:         "Travel",
			Requirements: []app.SkillSetRequirement{{TypeID: skill.ID, Level: 4}},
		})
		// then
		require.NoError(t, err)
		sets, err := s.ListSkillSets(ctx)
		require.NoError(t, err)
		xassert.Equal(t, []app.SkillSet{{
			ID:           id,
			Name:         "Travel",
			Requirements: []app.SkillSetRequirement{{TypeID: skill.ID, Name: skill.Name, Level: 4}},
		}}, sets)
	})
	t.Run("should add missing skills before creating", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		group := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: app.EveCategorySkill})
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/universe/types/3449",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"description": "",
				"group_id":    group.ID,
				"name":        "Navigation",
				"published":   true,
				"type_id":     3449,
			}),
		)
		// when
		id, err := s.CreateSkillSet(ctx, app.SkillSet{
			Name:         "Travel",
			Requirements: []app.SkillSetRequirement{{TypeID: 3449, Level: 4}},
		})
		// then
		require.NoError(t, err)
		sets, err := s.ListSkillSets(ctx)
		require.NoError(t, err)
		xassert.Equal(t, []app.SkillSet{{
			ID:           id,
			Name:         "Travel",
			Requirements: []app.SkillSetRequirement{{TypeID: 3449, Name: "Navigation", Level: 4}},
		}}, sets)
	})
	t.Run("can update", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		group := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: app.EveCategorySkill})
		skill1 := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID})
		skill2 := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID})
		id, err := s.CreateSkillSet(ctx, app.SkillSet{
			Name:         "Travel",
			Requirements: []app.SkillSetRequirement{{TypeID: skill1.ID, Level: 4}},
		})
		require.NoError(t, err)
		// when
		err = s.UpdateSkillSet(ctx, app.SkillSet{
			ID:           id,
			Name:         "Fast travel",
			Requirements: []app.SkillSetRequirement{{TypeID: skill2.ID, Level: 3}},
		})
		// then
		require.NoError(t, err)
		sets, err := s.ListSkillSets(ctx)
		require.NoError(t, err)
		xassert.Equal(t, []app.SkillSet{{
			ID:           id,
			Name:         "Fast travel",
			Requirements: []app.SkillSetRequirement{{TypeID: skill2.ID, Name: skill2.Name, Level: 3}},
		}}, sets)
	})
	t.Run("can delete", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		group := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: app.EveCategorySkill})
		skill := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID})
		id, err := s.CreateSkillSet(ctx, app.SkillSet{
			Name:         "Travel",
			Requirements: []app.SkillSetRequirement{{TypeID: skill.ID, Level: 4}},
		})
		require.NoError(t, err)
		// when
		err = s.DeleteSkillSet(ctx, id)
		// then
		require.NoError(t, err)
		sets, err := s.ListSkillSets(ctx)
		require.NoError(t, err)
		assert.Empty(t, sets)
	})
}
