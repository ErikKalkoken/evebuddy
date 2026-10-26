package storage_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
)

func TestSkillPlanStorage(t *testing.T) {
	db, ro, err := storage.InitDB("file:" + filepath.Join(t.TempDir(), "plans.sqlite"))
	require.NoError(t, err)
	defer db.Close()
	defer ro.Close()
	st := storage.New(db, ro)
	f := testutil.NewFactory(st, ro)
	ctx := t.Context()
	group := f.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: app.EveCategorySkill})
	skill := f.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID, Name: "Navigation", IsPublished: true})
	other := f.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID, Name: "Afterburner", IsPublished: true})
	p := app.SkillPlan{
		Name: " Travel ",
		Requirements: []app.SkillPlanRequirement{
			{TypeID: skill.ID, Level: 4},
		},
	}
	id, err := st.SaveSkillPlan(ctx, p)
	require.NoError(t, err)
	require.NotZero(t, id)

	// Read through another storage instance to ensure plans are persisted.
	reader := storage.New(db, db)
	plans, err := reader.ListSkillPlans(ctx)
	require.NoError(t, err)
	require.Len(t, plans, 1)
	assert.Equal(t, "Travel", plans[0].Name)
	assert.Equal(t, []app.SkillPlanRequirement{{TypeID: skill.ID, Name: skill.Name, Level: 4}}, plans[0].Requirements)
	p = plans[0]
	p.Name = "travel"
	_, err = st.SaveSkillPlan(ctx, app.SkillPlan{Name: p.Name, Requirements: p.Requirements})
	assert.ErrorIs(t, err, app.ErrAlreadyExists)

	p.Name = "Fast travel"
	p.Requirements = []app.SkillPlanRequirement{{TypeID: other.ID, Level: 3}}
	updatedID, err := st.SaveSkillPlan(ctx, p)
	require.NoError(t, err)
	assert.Equal(t, id, updatedID)
	plans, err = st.ListSkillPlans(ctx)
	require.NoError(t, err)
	assert.Equal(t, "Fast travel", plans[0].Name)
	assert.Equal(t, []app.SkillPlanRequirement{{TypeID: other.ID, Name: other.Name, Level: 3}}, plans[0].Requirements)

	// A failure after updating the name and clearing requirements must roll back both.
	p.Name = "Broken edit"
	p.Requirements = []app.SkillPlanRequirement{{TypeID: 999999999, Level: 2}}
	_, err = st.SaveSkillPlan(ctx, p)
	require.Error(t, err)
	unchanged, err := st.ListSkillPlans(ctx)
	require.NoError(t, err)
	assert.Equal(t, plans, unchanged)

	nonSkill := f.CreateEveType()
	p.Requirements = []app.SkillPlanRequirement{{TypeID: nonSkill.ID, Level: 2}}
	_, err = st.SaveSkillPlan(ctx, p)
	assert.ErrorIs(t, err, app.ErrInvalid)
	p.Requirements = []app.SkillPlanRequirement{{TypeID: skill.ID, Level: 6}}
	_, err = st.SaveSkillPlan(ctx, p)
	assert.ErrorIs(t, err, app.ErrInvalid)
	p.Requirements = []app.SkillPlanRequirement{{TypeID: skill.ID, Level: 1}, {TypeID: skill.ID, Level: 2}}
	_, err = st.SaveSkillPlan(ctx, p)
	assert.ErrorIs(t, err, app.ErrInvalid)
	p.Requirements = nil
	_, err = st.SaveSkillPlan(ctx, p)
	assert.ErrorIs(t, err, app.ErrInvalid)
	p.ID = id + 999
	p.Requirements = []app.SkillPlanRequirement{{TypeID: skill.ID, Level: 1}}
	_, err = st.SaveSkillPlan(ctx, p)
	assert.ErrorIs(t, err, app.ErrNotFound)

	require.NoError(t, st.DeleteSkillPlan(ctx, id))
	plans, err = st.ListSkillPlans(ctx)
	require.NoError(t, err)
	assert.Empty(t, plans)
	var count int
	require.NoError(t, db.QueryRow("SELECT count(*) FROM skill_plan_requirements").Scan(&count))
	assert.Zero(t, count)
	assert.ErrorIs(t, st.DeleteSkillPlan(ctx, id), app.ErrNotFound)
}
