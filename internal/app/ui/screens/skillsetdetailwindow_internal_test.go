package screens

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"github.com/maniartech/signals"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestSkillSetSkillItem_Set(t *testing.T) {
	test.NewTempApp(t)
	success := theme.NewSuccessThemedResource(theme.ConfirmIcon())
	failure := theme.NewErrorThemedResource(theme.CancelIcon())
	for _, tc := range []struct {
		name        string
		activeLevel int64
		skillsKnown bool
		wantIcon    fyne.Resource
		wantCurrent string
	}{
		{"met", 4, true, success, "IV"},
		{"below required level", 2, true, failure, "II"},
		{"not injected", 0, true, failure, "–"},
		{"unknown", 0, false, theme.QuestionIcon(), "?"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// given
			w := newSkillSetSkillItem()
			x := skillSetSkill{activeLevel: tc.activeLevel, name: "Navigation", requiredLevel: 3}
			// when
			w.set(x, tc.skillsKnown)
			// then
			xassert.Equal(t, "Navigation III", w.nameLabel.Text)
			xassert.Equal(t, tc.wantIcon.Name(), w.statusIcon.Resource.Name())
			xassert.Equal(t, tc.wantCurrent, w.currentLabel.Text)
		})
	}
}

func TestSkillSetDetail(t *testing.T) {
	_, st, _ := testutil.NewDBOnDisk(t)
	set := app.SkillSet{ID: 1, Name: "Travel", Requirements: []app.SkillSetRequirement{
		{TypeID: 10, Name: "Navigation", Level: 3},
		{TypeID: 11, Name: "Afterburner", Level: 2},
	}}
	character := skillSetCharacter{id: 5, name: "Bruce", levels: map[int64]int64{10: 3, 11: 1}}
	newDetail := func(t *testing.T) (*skillSetDetail, *signals.SyncSignal[skillSetsData]) {
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		dataChanged := signals.NewSync[skillSetsData]()
		return newSkillSetDetail(u, set.ID, newSkillSetRow(character, set), dataChanged), dataChanged
	}
	t.Run("should show missing skills when opened for a character with missing skills", func(t *testing.T) {
		// when
		a, _ := newDetail(t)
		// then
		xassert.Equal(t, skillSetsStatusMissing, a.statusLabel.Text)
		xassert.Equal(t, "1 / 2", a.skillsLabel.Text)
		xassert.Equal(t, 1, a.filterButton.Selected)
		xassert.Equal(t, []int64{11}, skillSetDetailTypeIDs(a))
	})
	t.Run("should update when skill levels changed", func(t *testing.T) {
		// given
		a, dataChanged := newDetail(t)
		c2 := skillSetCharacter{id: 5, name: "Bruce", levels: map[int64]int64{10: 3, 11: 2}}
		// when
		dataChanged.Emit(t.Context(), skillSetsData{characters: []skillSetCharacter{c2}, sets: []app.SkillSet{set}})
		// then
		xassert.Equal(t, skillSetsStatusQualified, a.statusLabel.Text)
		xassert.Equal(t, "2 / 2", a.skillsLabel.Text)
		xassert.Equal(t, []int64{}, skillSetDetailTypeIDs(a))
		xassert.Equal(t, true, a.hintLabel.Visible())
	})
	t.Run("should update when skill set changed and keep the selected filter", func(t *testing.T) {
		// given
		a, dataChanged := newDetail(t)
		a.filterButton.SetSelected(0)
		set2 := app.SkillSet{ID: 1, Name: "Travel", Requirements: []app.SkillSetRequirement{
			{TypeID: 10, Name: "Navigation", Level: 3},
		}}
		// when
		dataChanged.Emit(t.Context(), skillSetsData{characters: []skillSetCharacter{character}, sets: []app.SkillSet{set2}})
		// then
		xassert.Equal(t, 0, a.filterButton.Selected)
		xassert.Equal(t, []int64{10}, skillSetDetailTypeIDs(a))
		xassert.Equal(t, skillSetsStatusQualified, a.statusLabel.Text)
	})
	t.Run("should show issue when skill set was deleted", func(t *testing.T) {
		// given
		a, dataChanged := newDetail(t)
		// when
		dataChanged.Emit(t.Context(), skillSetsData{characters: []skillSetCharacter{character}})
		// then
		xassert.Equal(t, true, a.issueLabel.Visible())
		xassert.Equal(t, "Skill set has been deleted", a.issueLabel.Text)
		xassert.Equal(t, []int64{}, skillSetDetailTypeIDs(a))
		xassert.Equal(t, false, a.filterBox.Visible())
		xassert.Equal(t, false, a.hintLabel.Visible())
	})
	t.Run("should show issue when character was removed", func(t *testing.T) {
		// given
		a, dataChanged := newDetail(t)
		// when
		dataChanged.Emit(t.Context(), skillSetsData{sets: []app.SkillSet{set}})
		// then
		xassert.Equal(t, true, a.issueLabel.Visible())
		xassert.Equal(t, "Character has been removed", a.issueLabel.Text)
		xassert.Equal(t, []int64{}, skillSetDetailTypeIDs(a))
	})
	t.Run("should stop listening when stopped", func(t *testing.T) {
		// given
		a, dataChanged := newDetail(t)
		// when
		a.stop()
		// then
		xassert.Equal(t, 0, dataChanged.Len())
	})
}

func TestShowSkillSetDetailWindow(t *testing.T) {
	_, st, _ := testutil.NewDBOnDisk(t)
	t.Run("should stop listening when window is closed", func(t *testing.T) {
		// given
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		set := app.SkillSet{ID: 1, Name: "Travel", Requirements: []app.SkillSetRequirement{
			{TypeID: 10, Name: "Navigation", Level: 3},
		}}
		r := newSkillSetRow(skillSetCharacter{id: 5, name: "Bruce"}, set)
		dataChanged := signals.NewSync[skillSetsData]()
		showSkillSetDetailWindow(u, set, r, dataChanged)
		w, created, _ := u.GetOrCreateWindowWithOnClosed("skillset-1-5")
		require.False(t, created)
		require.Equal(t, 1, dataChanged.Len())
		// when
		w.Close()
		// then
		xassert.Equal(t, 0, dataChanged.Len())
	})
}

// skillSetDetailTypeIDs returns the type IDs of the shown skills.
func skillSetDetailTypeIDs(a *skillSetDetail) []int64 {
	ids := make([]int64, 0)
	for _, x := range a.skillsFiltered {
		ids = append(ids, x.typeID)
	}
	return ids
}
