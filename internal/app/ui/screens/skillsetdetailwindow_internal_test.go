package screens

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"

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
			xassert.Equal(t, "Navigation III", w.name.Text)
			xassert.Equal(t, tc.wantIcon.Name(), w.icon.Resource.Name())
			xassert.Equal(t, tc.wantCurrent, w.current.Text)
		})
	}
}
