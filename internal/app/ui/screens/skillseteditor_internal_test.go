package screens

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestParseSkillSetRequirements(t *testing.T) {
	skills := map[string]*app.EveType{
		"navigation":  {ID: 1, Name: "Navigation"},
		"afterburner": {ID: 2, Name: "Afterburner"},
	}
	t.Run("can parse and merge repeated skills at highest level", func(t *testing.T) {
		// when
		got, err := parseSkillSetRequirements("\r\n navigation II\r\nNavigation IV\r\nNavigation 1\r\nAfterburner 3\n", skills)
		// then
		require.NoError(t, err)
		xassert.Equal(t, []app.SkillSetRequirement{
			{TypeID: 2, Name: "Afterburner", Level: 3},
			{TypeID: 1, Name: "Navigation", Level: 4},
		}, got)
	})
	for _, level := range []string{"I", "II", "III", "IV", "V", "1", "2", "3", "4", "5"} {
		t.Run("can parse level "+level, func(t *testing.T) {
			// when
			_, err := parseSkillSetRequirements("Navigation "+level, skills)
			// then
			require.NoError(t, err)
		})
	}
	for _, tc := range []struct {
		name string
		text string
	}{
		{"no skills", ""},
		{"missing level", "Navigation"},
		{"level too low", "Navigation 0"},
		{"level too high", "Navigation 6"},
		{"roman level too high", "Navigation VI"},
		{"invalid level", "Navigation xyz"},
		{"unknown skill", "Unknown 3"},
		{"invalid second line", "Navigation IV\nbad"},
	} {
		t.Run("should return error for "+tc.name, func(t *testing.T) {
			// when
			_, err := parseSkillSetRequirements(tc.text, skills)
			// then
			assert.Error(t, err)
		})
	}
}

func TestNewSkillSetNameValidator(t *testing.T) {
	sets := []app.SkillSet{{ID: 1, Name: "Travel"}, {ID: 2, Name: "Mining"}}
	for _, tc := range []struct {
		name     string
		editedID int64
		text     string
		wantErr  bool
	}{
		{"new name", 0, "Combat", false},
		{"empty name", 0, "  ", true},
		{"existing name ignoring case", 0, "travel", true},
		{"own name when editing", 1, "TRAVEL", false},
		{"other name when editing", 1, "Mining", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// given
			v := newSkillSetNameValidator(sets, tc.editedID)
			// when
			err := v(tc.text)
			// then
			xassert.Equal(t, tc.wantErr, err != nil)
		})
	}
}

func TestSkillSetEditor_SaveButton(t *testing.T) {
	test.NewTempApp(t)
	sets := []app.SkillSet{{ID: 1, Name: "Travel"}}
	skills := []*app.EveType{{ID: 1, Name: "Navigation"}}
	travel := app.SkillSet{ID: 1, Name: "Travel", Requirements: []app.SkillSetRequirement{
		{TypeID: 1, Name: "Navigation", Level: 4},
	}}
	for _, tc := range []struct {
		name         string
		set          app.SkillSet
		nameText     string
		requirements string
		wantEnabled  bool
	}{
		{"new set without input", app.SkillSet{}, "", "", false},
		{"empty name", app.SkillSet{}, "  ", "Navigation IV", false},
		{"duplicate name", app.SkillSet{}, "travel", "Navigation IV", false},
		{"invalid line", app.SkillSet{}, "Combat", "Navigation", false},
		{"valid input", app.SkillSet{}, "Combat", "Navigation IV", true},
		{"existing set unchanged", travel, "Travel", "Navigation IV", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// given
			w := newSkillSetEditor(test.NewTempWindow(t, nil), tc.set, sets, skills, func(app.SkillSet) {}, func() {}, func() {})
			// when
			w.nameEntry.SetText(tc.nameText)
			w.skillsEntry.SetText(tc.requirements)
			// then
			xassert.Equal(t, tc.wantEnabled, !w.saveButton.Disabled())
		})
	}
}

func TestSkillSetEditor_AddSkill(t *testing.T) {
	test.NewTempApp(t)
	skills := []*app.EveType{{ID: 1, Name: "Navigation"}, {ID: 2, Name: "Afterburner"}}
	t.Run("picking a skill opens the level dialog", func(t *testing.T) {
		// given
		win := test.NewTempWindow(t, nil)
		w := newSkillSetEditor(win, app.SkillSet{}, nil, skills, func(app.SkillSet) {}, func() {}, func() {})
		win.SetContent(w)
		// when
		w.searchEntry.OnPicked("Navigation")
		// then
		assert.NotNil(t, win.Canvas().Overlays().Top())
		assert.Len(t, w.levelButtons, 5)
	})
	t.Run("enter on a skill name opens the level dialog", func(t *testing.T) {
		// given
		win := test.NewTempWindow(t, nil)
		w := newSkillSetEditor(win, app.SkillSet{}, nil, skills, func(app.SkillSet) {}, func() {}, func() {})
		win.SetContent(w)
		w.searchEntry.SetText(" navigation ")
		// when
		w.searchEntry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
		// then
		assert.NotNil(t, win.Canvas().Overlays().Top())
	})
	t.Run("enter on text that is not a skill name does nothing", func(t *testing.T) {
		// given
		win := test.NewTempWindow(t, nil)
		w := newSkillSetEditor(win, app.SkillSet{}, nil, skills, func(app.SkillSet) {}, func() {}, func() {})
		win.SetContent(w)
		w.searchEntry.SetText("Navi")
		// when
		w.searchEntry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
		// then
		assert.Nil(t, win.Canvas().Overlays().Top())
		xassert.Equal(t, "Navi", w.searchEntry.Text)
	})
	t.Run("selecting a level adds the skill and closes the dialog", func(t *testing.T) {
		// given
		win := test.NewTempWindow(t, nil)
		w := newSkillSetEditor(win, app.SkillSet{}, nil, skills, func(app.SkillSet) {}, func() {}, func() {})
		win.SetContent(w)
		w.skillsEntry.SetText("Afterburner I\n")
		w.searchEntry.SetText("Navigation")
		w.searchEntry.OnPicked("Navigation")
		// when
		test.Tap(w.levelButtons[2])
		// then
		xassert.Equal(t, "Afterburner I\nNavigation III", w.skillsEntry.Text)
		xassert.Equal(t, "", w.searchEntry.Text)
		assert.Nil(t, win.Canvas().Overlays().Top())
	})
	t.Run("cancelling the level dialog clears the search entry", func(t *testing.T) {
		// given
		win := test.NewTempWindow(t, nil)
		w := newSkillSetEditor(win, app.SkillSet{}, nil, skills, func(app.SkillSet) {}, func() {}, func() {})
		win.SetContent(w)
		w.searchEntry.SetText("Navigation")
		w.searchEntry.OnPicked("Navigation")
		// when
		w.levelDialog.Hide() // same as tapping Cancel
		// then
		xassert.Equal(t, "", w.searchEntry.Text)
		xassert.Equal(t, "", w.skillsEntry.Text)
	})
}

func TestSkillSetEditor_Callbacks(t *testing.T) {
	test.NewTempApp(t)
	skills := []*app.EveType{{ID: 1, Name: "Navigation"}}
	travel := app.SkillSet{ID: 7, Name: "Travel", Requirements: []app.SkillSetRequirement{
		{TypeID: 1, Name: "Navigation", Level: 4},
	}}
	t.Run("save reports the edited skill set", func(t *testing.T) {
		// given
		var got app.SkillSet
		w := newSkillSetEditor(test.NewTempWindow(t, nil), travel, nil, skills, func(s app.SkillSet) { got = s }, func() {}, func() {})
		w.nameEntry.SetText("  Fast travel ")
		w.skillsEntry.SetText("navigation 2\nNavigation V")
		// when
		test.Tap(w.saveButton)
		// then
		xassert.Equal(t, app.SkillSet{ID: 7, Name: "Fast travel", Requirements: []app.SkillSetRequirement{
			{TypeID: 1, Name: "Navigation", Level: 5},
		}}, got)
	})
	t.Run("cancel calls its callback", func(t *testing.T) {
		// given
		var called bool
		w := newSkillSetEditor(test.NewTempWindow(t, nil), travel, nil, skills, func(app.SkillSet) {}, func() { called = true }, func() {})
		// when
		test.Tap(w.cancelButton)
		// then
		assert.True(t, called)
	})
	t.Run("delete calls its callback when editing", func(t *testing.T) {
		// given
		var called bool
		w := newSkillSetEditor(test.NewTempWindow(t, nil), travel, nil, skills, func(app.SkillSet) {}, func() {}, func() { called = true })
		// when
		require.NotNil(t, w.deleteButton)
		test.Tap(w.deleteButton)
		// then
		assert.True(t, called)
	})
	t.Run("has no delete button for a new set", func(t *testing.T) {
		// when
		w := newSkillSetEditor(test.NewTempWindow(t, nil), app.SkillSet{}, nil, skills, func(app.SkillSet) {}, func() {}, func() {})
		// then
		assert.Nil(t, w.deleteButton)
	})
}
