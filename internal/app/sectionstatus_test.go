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
				SectionStatus: app.SectionStatus{
					CompletedAt: tc.completedAt,
					Section:     app.SectionCharacterSkillqueue,
				},
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
		{"", false},
		{"error", true},
	}
	for _, tc := range cases {
		t.Run("Can report when update is ok", func(t *testing.T) {
			// given
			o := app.CharacterSectionStatus{
				SectionStatus: app.SectionStatus{
					ErrorMessage: tc.errorMessage,
					Section:      app.SectionCharacterSkillqueue,
				},
			}
			// when/then
			assert.Equal(t, tc.want, o.HasError())
		})
	}
}

func TestCharacterUpdateStatusIsMissing(t *testing.T) {
	cases := []struct {
		completedAt time.Time
		want        bool
	}{
		{time.Now(), false},
		{time.Time{}, true},
	}
	for _, tc := range cases {
		t.Run("can report when status is missing", func(t *testing.T) {
			// given
			o := app.CharacterSectionStatus{
				SectionStatus: app.SectionStatus{
					CompletedAt: tc.completedAt,
					Section:     app.SectionCharacterSkillqueue,
				},
			}
			// when/then
			assert.Equal(t, tc.want, o.IsMissing())
		})
	}
}
