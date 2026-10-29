package characterservice_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
)

func TestSkillPlanParsing(t *testing.T) {
	skills := []*app.EveSkill{
		{Type: &app.EveType{ID: 1, Name: "Navigation"}},
		{Type: &app.EveType{ID: 2, Name: "Afterburner"}},
	}
	p, err := characterservice.ParseSkillPlan(" Travel ", "\r\n navigation II\r\nNavigation IV\r\nNavigation 1\r\nAfterburner 3\n", skills)
	require.NoError(t, err)
	assert.Equal(t, "Travel", p.Name)
	assert.Equal(t, []app.SkillPlanRequirement{
		{TypeID: 2, Name: "Afterburner", Level: 3},
		{TypeID: 1, Name: "Navigation", Level: 4},
	}, p.Requirements)
	for _, text := range []string{"", "Navigation", "Navigation 0", "Navigation 6", "Navigation VI", "Navigation xyz", "Unknown 3", "Navigation IV\nbad"} {
		t.Run(text, func(t *testing.T) {
			_, err := characterservice.ParseSkillPlan("Plan", text, skills)
			assert.ErrorIs(t, err, app.ErrInvalid)
		})
	}
	_, err = characterservice.ParseSkillPlan("  ", "Navigation I", skills)
	assert.ErrorIs(t, err, app.ErrInvalid)
	for _, level := range []string{"I", "II", "III", "IV", "V", "1", "2", "3", "4", "5"} {
		_, err := characterservice.ParseSkillPlan("Plan", "Navigation "+level, skills)
		require.NoError(t, err)
	}
}

func TestSkillPlanQualification(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()
	ctx := t.Context()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})
	group := f.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: app.EveCategorySkill})
	navigation := f.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID, Name: "Navigation", IsPublished: true})
	afterburner := f.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID, Name: "Afterburner", IsPublished: true})
	id, err := s.SaveSkillPlan(ctx, 0, "Travel", "Navigation IV\nAfterburner III")
	require.NoError(t, err)
	plans, err := s.ListSkillPlans(ctx)
	require.NoError(t, err)
	require.Len(t, plans, 1)
	assert.Equal(t, id, plans[0].ID)

	qualified := f.CreateCharacter()
	restricted := f.CreateCharacter()
	missing := f.CreateCharacter()
	unknown := f.CreateCharacter()
	empty := f.CreateCharacter()
	for _, c := range []*app.Character{qualified, restricted, missing, empty} {
		f.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterSkills,
			CompletedAt: time.Now().Add(-time.Hour),
		})
	}
	for _, c := range []*app.Character{qualified, restricted, unknown} {
		active := int64(5)
		if c.ID == restricted.ID {
			active = 2
		}
		f.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID: c.ID, TypeID: navigation.ID, ActiveSkillLevel: active, TrainedSkillLevel: 5,
		})
		f.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID: c.ID, TypeID: afterburner.ID, ActiveSkillLevel: 3, TrainedSkillLevel: 3,
		})
	}
	f.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
		CharacterID: missing.ID, TypeID: navigation.ID, ActiveSkillLevel: 4, TrainedSkillLevel: 4,
	})
	for _, name := range []string{"Account A", "PvP"} {
		tag, err := s.CreateTag(ctx, name)
		require.NoError(t, err)
		require.NoError(t, s.AddTagToCharacter(ctx, qualified.ID, tag.ID))
	}
	results, err := s.CheckSkillPlan(ctx, plans[0])
	require.NoError(t, err)
	require.Len(t, results, 5)
	byID := make(map[int64]characterservice.SkillPlanCharacter)
	for _, c := range results {
		byID[c.Character.ID] = c
	}
	assert.True(t, byID[qualified.ID].Qualified())
	assert.Equal(t, []string{"Account A", "PvP"}, byID[qualified.ID].Tags)
	assert.Equal(t, 2, byID[qualified.ID].MetCount())
	assert.False(t, byID[restricted.ID].Qualified(), "trained level must not substitute for active level")
	assert.Equal(t, 1, byID[restricted.ID].MetCount())
	assert.False(t, byID[missing.ID].Qualified())
	assert.Equal(t, 1, byID[missing.ID].MetCount())
	assert.True(t, byID[empty.ID].SkillsKnown)
	assert.False(t, byID[empty.ID].Qualified())
	assert.Zero(t, byID[empty.ID].MetCount())
	assert.False(t, byID[unknown.ID].SkillsKnown)
	assert.False(t, byID[unknown.ID].Qualified(), "unsynced characters cannot qualify")
	assert.Zero(t, byID[unknown.ID].MetCount())

	require.NoError(t, s.DeleteSkillPlan(ctx, id))
	plans, err = s.ListSkillPlans(ctx)
	require.NoError(t, err)
	assert.Empty(t, plans)
}
