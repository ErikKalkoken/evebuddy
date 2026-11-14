package screens

import (
	"context"
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestSkillSetsFilter_Match(t *testing.T) {
	qualified := skillSetRow{
		skills:      []skillSetSkill{{activeLevel: 5, requiredLevel: 4}},
		skillsKnown: true,
		tags:        set.Of("alpha"),
	}
	for _, tc := range []struct {
		name   string
		filter skillSetsFilter
		want   bool
	}{
		{"no filter", skillSetsFilter{}, true},
		{"status matches", skillSetsFilter{status: skillSetsStatusQualified}, true},
		{"status differs", skillSetsFilter{status: skillSetsStatusMissing}, false},
		{"tag matches", skillSetsFilter{tag: "alpha"}, true},
		{"tag missing", skillSetsFilter{tag: "bravo"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got := tc.filter.match(qualified)
			// then
			xassert.Equal(t, tc.want, got)
		})
	}
}

func TestNewSkillSetRow(t *testing.T) {
	s := app.SkillSet{Requirements: []app.SkillSetRequirement{
		{TypeID: 1, Name: "Navigation", Level: 4},
		{TypeID: 2, Name: "Afterburner", Level: 3},
	}}
	for _, tc := range []struct {
		name       string
		levels     map[int64]int64
		wantStatus string
		wantSkills string
	}{
		{"all requirements met", map[int64]int64{1: 4, 2: 5}, skillSetsStatusQualified, "2 / 2"},
		{"some requirements missing", map[int64]int64{1: 3, 2: 3}, skillSetsStatusMissing, "1 / 2"},
		{"absent skills count as missing", map[int64]int64{99: 5}, skillSetsStatusMissing, "0 / 2"},
		{"character without skills is unknown", nil, skillSetsStatusUnknown, "?"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// when
			r := newSkillSetRow(skillSetCharacter{id: 42, name: "Bruce", levels: tc.levels}, s)
			// then
			xassert.Equal(t, tc.wantStatus, r.status())
			xassert.Equal(t, tc.wantSkills, r.skillsDisplay())
		})
	}
}

func TestSkillSetCharacterItem_Set(t *testing.T) {
	test.NewTempApp(t)
	s := app.SkillSet{Requirements: []app.SkillSetRequirement{
		{TypeID: 1, Name: "Navigation", Level: 4},
		{TypeID: 2, Name: "Afterburner", Level: 3},
	}}
	for _, tc := range []struct {
		name           string
		levels         map[int64]int64
		wantStatus     string
		wantImportance widget.Importance
		wantSkills     string
	}{
		{"qualified", map[int64]int64{1: 4, 2: 3}, skillSetsStatusQualified, widget.SuccessImportance, "2 / 2 skills"},
		{"missing skills", map[int64]int64{1: 4}, skillSetsStatusMissing, widget.WarningImportance, "1 / 2 skills"},
		{"unknown", nil, skillSetsStatusUnknown, widget.MediumImportance, "?"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// given
			w := newSkillSetCharacterItem()
			r := newSkillSetRow(skillSetCharacter{id: 42, name: "Bruce", levels: tc.levels}, s)
			// when
			w.set(r)
			// then
			xassert.Equal(t, "Bruce", w.name.Text)
			xassert.Equal(t, tc.wantStatus, w.status.Text)
			xassert.Equal(t, tc.wantImportance, w.status.Importance)
			xassert.Equal(t, tc.wantSkills, w.skills.Text)
		})
	}
}

func TestSkillSets_Update(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ctx := context.Background()
	newSkillSets := func(t *testing.T) *SkillSets {
		return NewSkillSets(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
	}
	statuses := func(a *SkillSets) map[int64]string {
		m := make(map[int64]string)
		for _, r := range a.rowsFiltered {
			m[r.characterID] = r.status()
		}
		return m
	}
	t.Run("shows characters evaluated against the first skill set", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		group := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: app.EveCategorySkill})
		skill := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID})
		qualified := factory.CreateCharacter()
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID: qualified.ID, TypeID: skill.ID, ActiveSkillLevel: 5, TrainedSkillLevel: 5,
		})
		missing := factory.CreateCharacter()
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID: missing.ID, TypeID: skill.ID, ActiveSkillLevel: 4, TrainedSkillLevel: 5,
		})
		unknown := factory.CreateCharacter()
		_, err := st.CreateSkillSet(ctx, app.SkillSet{
			Name:         "Travel",
			Requirements: []app.SkillSetRequirement{{TypeID: skill.ID, Level: 4}},
		})
		require.NoError(t, err)
		_, err = st.CreateSkillSet(ctx, app.SkillSet{
			Name:         "Expert",
			Requirements: []app.SkillSetRequirement{{TypeID: skill.ID, Level: 5}},
		})
		require.NoError(t, err)
		a := newSkillSets(t)
		// when
		a.update(ctx)
		// then
		xassert.Equal(t, []string{"Expert", "Travel"}, a.setSelect.Options)
		xassert.Equal(t, "Expert", a.setSelect.Selected)
		xassert.Equal(t, map[int64]string{
			qualified.ID: skillSetsStatusQualified,
			missing.ID:   skillSetsStatusMissing,
			unknown.ID:   skillSetsStatusUnknown,
		}, statuses(a))
		assert.False(t, a.setSelect.Disabled())
		assert.False(t, a.editButton.Disabled())
	})
	t.Run("can switch to another skill set", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		group := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: app.EveCategorySkill})
		skill := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID})
		c := factory.CreateCharacter()
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID: c.ID, TypeID: skill.ID, ActiveSkillLevel: 4, TrainedSkillLevel: 4,
		})
		_, err := st.CreateSkillSet(ctx, app.SkillSet{
			Name:         "Expert",
			Requirements: []app.SkillSetRequirement{{TypeID: skill.ID, Level: 5}},
		})
		require.NoError(t, err)
		_, err = st.CreateSkillSet(ctx, app.SkillSet{
			Name:         "Travel",
			Requirements: []app.SkillSetRequirement{{TypeID: skill.ID, Level: 4}},
		})
		require.NoError(t, err)
		a := newSkillSets(t)
		a.update(ctx)
		// when
		a.setSelect.SetSelected("Travel")
		// then
		xassert.Equal(t, "Travel", a.selectedSet.Name)
		xassert.Equal(t, map[int64]string{c.ID: skillSetsStatusQualified}, statuses(a))
	})
	t.Run("keeps selected skill set after reload", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		group := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: app.EveCategorySkill})
		skill := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID})
		_, err := st.CreateSkillSet(ctx, app.SkillSet{
			Name:         "Expert",
			Requirements: []app.SkillSetRequirement{{TypeID: skill.ID, Level: 5}},
		})
		require.NoError(t, err)
		_, err = st.CreateSkillSet(ctx, app.SkillSet{
			Name:         "Travel",
			Requirements: []app.SkillSetRequirement{{TypeID: skill.ID, Level: 4}},
		})
		require.NoError(t, err)
		a := newSkillSets(t)
		a.update(ctx)
		a.setSelect.SetSelected("Travel")
		// when
		a.update(ctx)
		// then
		xassert.Equal(t, "Travel", a.setSelect.Selected)
		xassert.Equal(t, "Travel", a.selectedSet.Name)
	})
	t.Run("selects created skill set", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		group := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: app.EveCategorySkill})
		skill := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID})
		_, err := st.CreateSkillSet(ctx, app.SkillSet{
			Name:         "Expert",
			Requirements: []app.SkillSetRequirement{{TypeID: skill.ID, Level: 5}},
		})
		require.NoError(t, err)
		a := newSkillSets(t)
		a.update(ctx)
		// when
		a.saveSkillSet(ctx, app.SkillSet{
			Name:         "Travel",
			Requirements: []app.SkillSetRequirement{{TypeID: skill.ID, Level: 4}},
		})
		// then
		xassert.Equal(t, "Travel", a.setSelect.Selected)
		xassert.Equal(t, []app.SkillSetRequirement{
			{TypeID: skill.ID, Name: skill.Name, Level: 4},
		}, a.selectedSet.Requirements)
	})
	t.Run("shows updated skill set", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		group := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: app.EveCategorySkill})
		skill := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID})
		id, err := st.CreateSkillSet(ctx, app.SkillSet{
			Name:         "Travel",
			Requirements: []app.SkillSetRequirement{{TypeID: skill.ID, Level: 4}},
		})
		require.NoError(t, err)
		a := newSkillSets(t)
		a.update(ctx)
		// when
		a.saveSkillSet(ctx, app.SkillSet{
			ID:           id,
			Name:         "Fast travel",
			Requirements: []app.SkillSetRequirement{{TypeID: skill.ID, Level: 5}},
		})
		// then
		xassert.Equal(t, []string{"Fast travel"}, a.setSelect.Options)
		xassert.Equal(t, "Fast travel", a.setSelect.Selected)
		xassert.Equal(t, 5, a.selectedSet.Requirements[0].Level)
	})
	t.Run("shows hint and disables select and edit when there are no skill sets", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateCharacter()
		a := newSkillSets(t)
		// when
		a.update(ctx)
		// then
		assert.Empty(t, a.rowsFiltered)
		assert.Contains(t, a.footerLabel.Text, "No skill sets")
		assert.True(t, a.setSelect.Disabled())
		assert.True(t, a.editButton.Disabled())
	})
	t.Run("reloads when skills of a character changed", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		group := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: app.EveCategorySkill})
		skill := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID})
		c := factory.CreateCharacter()
		_, err := st.CreateSkillSet(ctx, app.SkillSet{
			Name:         "Travel",
			Requirements: []app.SkillSetRequirement{{TypeID: skill.ID, Level: 4}},
		})
		require.NoError(t, err)
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		a := NewSkillSets(u)
		a.update(ctx)
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID: c.ID, TypeID: skill.ID, ActiveSkillLevel: 4, TrainedSkillLevel: 4,
		})
		// when
		u.Signals().CharacterSectionChanged.Emit(ctx, app.CharacterSectionUpdated{
			CharacterID: c.ID,
			Section:     app.SectionCharacterSkills,
		})
		// then
		xassert.Equal(t, map[int64]string{c.ID: skillSetsStatusQualified}, statuses(a))
	})
}

func TestSkillSets_Filter(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	rows := []skillSetRow{
		{characterID: 1, skills: []skillSetSkill{{activeLevel: 5, requiredLevel: 4}}, skillsKnown: true},
		{characterID: 2, skills: []skillSetSkill{{activeLevel: 1, requiredLevel: 4}}, skillsKnown: true},
	}
	newSkillSets := func(t *testing.T, isMobile bool) *SkillSets {
		a := NewSkillSets(testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			IsMobile: isMobile,
			Storage:  st,
		}))
		a.rows = rows
		a.filterRowsAsync("")
		require.Len(t, a.rowsFiltered, 2)
		return a
	}
	characterIDs := func(a *SkillSets) []int64 {
		var ids []int64
		for _, r := range a.rowsFiltered {
			ids = append(ids, r.characterID)
		}
		return ids
	}
	t.Run("can filter on desktop", func(t *testing.T) {
		// given
		a := newSkillSets(t, false)
		// when
		a.selectStatus.SetSelected(skillSetsStatusQualified)
		// then
		xassert.Equal(t, []int64{1}, characterIDs(a))
	})
	t.Run("can filter on mobile", func(t *testing.T) {
		// given
		a := newSkillSets(t, true)
		// when
		a.filterChip.SetSelected(map[string]string{skillSetsFilterStatus: skillSetsStatusMissing})
		// then
		xassert.Equal(t, []int64{2}, characterIDs(a))
	})
}

func TestSkillSets_ErrorDisplay(t *testing.T) {
	t.Run("shows error and clears skill sets when update fails", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		ctx := context.Background()
		group := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: app.EveCategorySkill})
		skill := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID})
		_, err := st.CreateSkillSet(ctx, app.SkillSet{
			Name:         "Travel",
			Requirements: []app.SkillSetRequirement{{TypeID: skill.ID, Level: 4}},
		})
		require.NoError(t, err)
		a := NewSkillSets(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		a.update(ctx)
		require.Equal(t, "Travel", a.setSelect.Selected)
		_, err = db.Exec("DROP TABLE skill_set_requirements") // let the next update fail
		require.NoError(t, err)
		// when
		a.update(ctx)
		// then
		assert.True(t, a.errorLabel.Visible())
		assert.Contains(t, a.errorLabel.Text, "ERROR:")
		assert.Empty(t, a.setSelect.Options)
		assert.Empty(t, a.setSelect.Selected)
		assert.Empty(t, a.rowsFiltered)
		assert.True(t, a.setSelect.Disabled())
		assert.True(t, a.editButton.Disabled())
	})
	t.Run("hides error after successful update", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewSkillSets(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		a.errorLabel.SetText("ERROR: failed")
		a.errorLabel.Show()
		// when
		a.update(context.Background())
		// then
		assert.False(t, a.errorLabel.Visible())
	})
}
