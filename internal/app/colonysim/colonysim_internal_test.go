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
)

func TestLogAborted(t *testing.T) {
	captureLog := func(t *testing.T) *bytes.Buffer {
		var buf bytes.Buffer
		old := slog.Default()
		slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
		t.Cleanup(func() { slog.SetDefault(old) })
		return &buf
	}
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
