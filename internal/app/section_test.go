package app_test

import (
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/stretchr/testify/assert"
)

func TestCharacterUpdateStatusIsExpired(t *testing.T) {
	now := time.Now()
	cases := []struct {
		completedAt time.Time
		want        bool
	}{
		{now.Add(-3 * time.Hour), true},
		{now, false},
		{time.Time{}, true},
	}
	for _, tc := range cases {
		t.Run("Can report when section update is expired", func(t *testing.T) {
			// given
			o := app.CharacterSectionStatus{
				Section:     app.SectionSkillqueue,
				CompletedAt: tc.completedAt,
			}
			// when/then
			assert.Equal(t, tc.want, o.IsExpired())
		})
	}
}

func TestCharacterUpdateStatusIsOK(t *testing.T) {
	cases := []struct {
		errorMessage string
		want         bool
	}{
		{"", true},
		{"error", false},
	}
	for _, tc := range cases {
		t.Run("Can report when update is ok", func(t *testing.T) {
			// given
			o := app.CharacterSectionStatus{
				Section:      app.SectionSkillqueue,
				ErrorMessage: tc.errorMessage,
			}
			// when/then
			assert.Equal(t, tc.want, o.IsOK())
		})
	}
}
