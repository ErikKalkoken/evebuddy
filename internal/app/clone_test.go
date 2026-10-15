package app_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func TestCalcNextCloneJump(t *testing.T) {
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name       string
		lastJumpAt optional.Optional[time.Time]
		skillLevel int64
		want       optional.Optional[time.Time]
	}{
		{"last jump unknown", optional.Optional[time.Time]{}, 0, optional.Optional[time.Time]{}},
		{"cooldown over", optional.New(now.Add(-25 * time.Hour)), 0, optional.New(time.Time{})},
		{"cooldown running without skill", optional.New(now.Add(-time.Hour)), 0, optional.New(now.Add(23 * time.Hour))},
		{"cooldown running with skill", optional.New(now.Add(-time.Hour)), 5, optional.New(now.Add(18 * time.Hour))},
		{"skill ends cooldown", optional.New(now.Add(-20 * time.Hour)), 5, optional.New(time.Time{})},
		{"cooldown ends exactly now", optional.New(now.Add(-24 * time.Hour)), 0, optional.New(now)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := app.CalcNextCloneJump(tc.lastJumpAt, tc.skillLevel, now)
			assert.Equal(t, tc.want, got)
		})
	}
}
