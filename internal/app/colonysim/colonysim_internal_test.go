package colonysim

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

// captureLog captures the default logger as JSON during the test.
func captureLog(t *testing.T) *bytes.Buffer {
	var buf bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(old) })
	return &buf
}

func TestLogAborted(t *testing.T) {
	parse := func(t *testing.T, buf *bytes.Buffer) map[string]any {
		var m map[string]any
		require.NoError(t, json.Unmarshal(buf.Bytes(), &m))
		return m
	}
	t.Run("should log warning with colony details", func(t *testing.T) {
		buf := captureLog(t)
		cp := &app.CharacterPlanet{CharacterID: 42, EvePlanet: &app.EvePlanet{ID: 7}}
		logAborted(cp, t0.Add(time.Hour))
		m := parse(t, buf)
		assert.Equal(t, "WARN", m["level"])
		assert.Equal(t, "Colony simulation aborted after exceeding event limit", m["msg"])
		assert.EqualValues(t, 42, m["characterID"])
		assert.EqualValues(t, 7, m["planetID"])
		assert.Equal(t, t0.Add(time.Hour).Format(time.RFC3339), m["simTime"])
		assert.EqualValues(t, maxEvents, m["maxEvents"])
	})
	t.Run("should log zero planet ID when planet is unknown", func(t *testing.T) {
		buf := captureLog(t)
		logAborted(&app.CharacterPlanet{CharacterID: 42}, t0)
		assert.EqualValues(t, 0, parse(t, buf)["planetID"])
	})
}

func TestForecast_Aborted(t *testing.T) {
	setMaxEvents := func(t *testing.T, n int) {
		old := maxEvents
		maxEvents = n
		t.Cleanup(func() { maxEvents = old })
	}
	// extractor runs every 30 minutes for 20 days
	cp := &app.CharacterPlanet{
		LastUpdate: t0,
		Pins: []*app.PlanetPin{
			newExtractor(1, aqueousLiquids, 1081, 30*time.Minute, t0, t0.Add(20*24*time.Hour)),
			newStorage(2, app.EveGroupStorageFacilities, 12_000),
		},
		Routes: []*app.PlanetRoute{newRoute(1, 1, 2, aqueousLiquids, 10_000)},
	}
	t.Run("should never reuse forecast when aborted before now", func(t *testing.T) {
		setMaxEvents(t, 10)
		buf := captureLog(t)
		now := t0.Add(24 * time.Hour)
		f := Forecast(cp, now)
		assert.Equal(t, optional.New(now), f.ValidUntil)
		assert.True(t, f.WorkEndsAt.IsEmpty())
		assert.False(t, f.WorksBeyondHorizon)
		assert.Contains(t, buf.String(), "Colony simulation aborted")
	})
	t.Run("should report no work end when aborted after now", func(t *testing.T) {
		setMaxEvents(t, 10)
		buf := captureLog(t)
		now := t0.Add(time.Hour)
		f := Forecast(cp, now)
		assert.Equal(t, optional.New(t0.Add(90*time.Minute)), f.ValidUntil, "next extractor cycle")
		assert.True(t, f.WorkEndsAt.IsEmpty())
		assert.False(t, f.WorksBeyondHorizon)
		assert.Contains(t, buf.String(), "Colony simulation aborted")
	})
	t.Run("should report no work end from WorkEndsAt when aborted", func(t *testing.T) {
		setMaxEvents(t, 10)
		buf := captureLog(t)
		assert.True(t, WorkEndsAt(cp, t0.Add(30*24*time.Hour)).IsEmpty())
		assert.Contains(t, buf.String(), "Colony simulation aborted")
	})
	t.Run("should report work end when not aborted", func(t *testing.T) {
		assert.Equal(t, optional.New(t0.Add(20*24*time.Hour)), WorkEndsAt(cp, t0.Add(30*24*time.Hour)))
	})
}
