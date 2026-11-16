package screens

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
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

func TestParseSkillSetLine(t *testing.T) {
	skills := map[string]*app.EveType{
		"navigation":          {ID: 1, Name: "Navigation"},
		"evasive maneuvering": {ID: 2, Name: "Evasive Maneuvering"},
	}
	for _, tc := range []struct {
		name      string
		line      string
		wantID    int64
		wantLevel int
	}{
		{"roman level", "Navigation IV", 1, 4},
		{"arabic level", "Navigation 2", 1, 2},
		{"lower case level", "Navigation iii", 1, 3},
		{"name in other case", "NAVIGATION V", 1, 5},
		{"name with spaces", "Evasive Maneuvering I", 2, 1},
		{"extra whitespace", "  Evasive   Maneuvering\tII\r", 2, 2},
	} {
		t.Run("can parse "+tc.name, func(t *testing.T) {
			// when
			et, level, err := parseSkillSetLine(tc.line, skills)
			// then
			require.NoError(t, err)
			xassert.Equal(t, tc.wantID, et.ID)
			xassert.Equal(t, tc.wantLevel, level)
		})
	}
	for _, tc := range []struct {
		name string
		line string
	}{
		{"empty line", ""},
		{"missing level", "Navigation"},
		{"missing name", "IV"},
		{"level too low", "Navigation 0"},
		{"level too high", "Navigation 6"},
		{"roman level too high", "Navigation VI"},
		{"invalid level", "Navigation xyz"},
		{"unknown skill", "Unknown 3"},
	} {
		t.Run("should return error for "+tc.name, func(t *testing.T) {
			// when
			_, _, err := parseSkillSetLine(tc.line, skills)
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
		assert.Subset(t, buttonTexts(win.Canvas().Overlays().Top()), []string{"I", "II", "III", "IV", "V"})
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
		test.Tap(findButton(t, win.Canvas().Overlays().Top(), "III"))
		// then
		xassert.Equal(t, "Afterburner I\nNavigation III", w.skillsEntry.Text)
		xassert.Equal(t, "", w.searchEntry.Text)
		assert.Nil(t, win.Canvas().Overlays().Top())
	})
	for _, tc := range []struct {
		name   string
		skills string
		want   string
	}{
		{"replaces the existing line of that skill", "Navigation IV\nAfterburner III", "Navigation II\nAfterburner III"},
		{"removes repeated lines of that skill", "Navigation IV\nAfterburner III\nnavigation 5", "Navigation II\nAfterburner III"},
		{"keeps lines it can not parse", "foo\nNavigation IV", "foo\nNavigation II"},
	} {
		t.Run("selecting a level "+tc.name, func(t *testing.T) {
			// given
			win := test.NewTempWindow(t, nil)
			w := newSkillSetEditor(win, app.SkillSet{}, nil, skills, func(app.SkillSet) {}, func() {}, func() {})
			win.SetContent(w)
			w.skillsEntry.SetText(tc.skills)
			w.searchEntry.OnPicked("Navigation")
			// when
			test.Tap(findButton(t, win.Canvas().Overlays().Top(), "II"))
			// then
			xassert.Equal(t, tc.want, w.skillsEntry.Text)
		})
	}
	t.Run("cancelling the level dialog clears the search entry", func(t *testing.T) {
		// given
		win := test.NewTempWindow(t, nil)
		w := newSkillSetEditor(win, app.SkillSet{}, nil, skills, func(app.SkillSet) {}, func() {}, func() {})
		win.SetContent(w)
		w.searchEntry.SetText("Navigation")
		w.searchEntry.OnPicked("Navigation")
		// when
		test.Tap(findButton(t, win.Canvas().Overlays().Top(), "Cancel"))
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
		test.Tap(findButton(t, w, "Cancel"))
		// then
		assert.True(t, called)
	})
	t.Run("delete calls its callback when editing", func(t *testing.T) {
		// given
		var called bool
		w := newSkillSetEditor(test.NewTempWindow(t, nil), travel, nil, skills, func(app.SkillSet) {}, func() {}, func() { called = true })
		// when
		test.Tap(findButton(t, w, "Delete"))
		// then
		assert.True(t, called)
	})
	t.Run("has no delete button for a new set", func(t *testing.T) {
		// when
		w := newSkillSetEditor(test.NewTempWindow(t, nil), app.SkillSet{}, nil, skills, func(app.SkillSet) {}, func() {}, func() {})
		// then
		assert.NotContains(t, buttonTexts(w), "Delete")
	})
}

// findButton returns the button with text under o or fails the test.
func findButton(t *testing.T, o fyne.CanvasObject, text string) *widget.Button {
	t.Helper()
	for _, x := range test.LaidOutObjects(o) {
		if b, ok := x.(*widget.Button); ok && b.Text == text {
			return b
		}
	}
	require.Failf(t, "button not found", "%q", text)
	return nil
}

// buttonTexts returns the texts of all buttons under o.
func buttonTexts(o fyne.CanvasObject) []string {
	var s []string
	for _, x := range test.LaidOutObjects(o) {
		if b, ok := x.(*widget.Button); ok {
			s = append(s, b.Text)
		}
	}
	return s
}

func TestSkillSetEditor_Saving(t *testing.T) {
	test.NewTempApp(t)
	skills := []*app.EveType{{ID: 1, Name: "Navigation"}}
	travel := app.SkillSet{ID: 7, Name: "Travel", Requirements: []app.SkillSetRequirement{
		{TypeID: 1, Name: "Navigation", Level: 4},
	}}
	t.Run("saving disables inputs and buttons", func(t *testing.T) {
		// given
		w := newSkillSetEditor(test.NewTempWindow(t, nil), travel, nil, skills, func(app.SkillSet) {}, func() {}, func() {})
		// when
		w.setSaving(true)
		// then
		assert.True(t, w.nameEntry.Disabled())
		assert.True(t, w.skillsEntry.Disabled())
		assert.True(t, w.searchEntry.Disabled())
		assert.True(t, w.saveButton.Disabled())
		assert.True(t, w.cancelButton.Disabled())
		assert.True(t, w.deleteButton.Disabled())
	})
	t.Run("ending saving enables inputs and buttons", func(t *testing.T) {
		// given
		w := newSkillSetEditor(test.NewTempWindow(t, nil), travel, nil, skills, func(app.SkillSet) {}, func() {}, func() {})
		w.setSaving(true)
		// when
		w.setSaving(false)
		// then
		assert.False(t, w.nameEntry.Disabled())
		assert.False(t, w.skillsEntry.Disabled())
		assert.False(t, w.searchEntry.Disabled())
		assert.False(t, w.saveButton.Disabled())
		assert.False(t, w.cancelButton.Disabled())
		assert.False(t, w.deleteButton.Disabled())
	})
	t.Run("ending saving keeps save disabled for invalid input", func(t *testing.T) {
		// given
		w := newSkillSetEditor(test.NewTempWindow(t, nil), travel, nil, skills, func(app.SkillSet) {}, func() {}, func() {})
		w.skillsEntry.SetText("bad")
		w.setSaving(true)
		// when
		w.setSaving(false)
		// then
		assert.True(t, w.saveButton.Disabled())
	})
	t.Run("shows error", func(t *testing.T) {
		// given
		w := newSkillSetEditor(test.NewTempWindow(t, nil), travel, nil, skills, func(app.SkillSet) {}, func() {}, func() {})
		// when
		w.setError("Failed to save")
		// then
		assert.True(t, w.errorLabel.Visible())
		xassert.Equal(t, "Failed to save", w.errorLabel.Text)
	})
	t.Run("saving again hides the error", func(t *testing.T) {
		// given
		w := newSkillSetEditor(test.NewTempWindow(t, nil), travel, nil, skills, func(app.SkillSet) {}, func() {}, func() {})
		w.setError("Failed to save")
		// when
		w.setSaving(true)
		// then
		assert.False(t, w.errorLabel.Visible())
	})
}
