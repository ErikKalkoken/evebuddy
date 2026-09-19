package settings

import (
	"encoding/json"
	"log/slog"
	"strconv"
	"time"
)

// get returns the raw string value for key, and whether it was present.
func (s *Settings) get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.values[key]
	return v, ok
}

// settingWrite is a queued write for persistLoop to apply to storage.
// A write with done set is a flush barrier: persistLoop closes it without
// persisting anything, once every write queued ahead of it has been applied.
type settingWrite struct {
	key, value string
	done       chan struct{}
}

// set stores value for key in the in-memory cache immediately and queues it to
// be persisted to storage asynchronously by persistLoop; any persistence error
// is logged (but not returned) there.
func (s *Settings) set(key, value string) {
	s.mu.Lock()
	s.values[key] = value
	s.mu.Unlock()
	s.writeQueue.Put(settingWrite{key: key, value: value})
}

// flush blocks until every write enqueued before this call has been persisted.
// For tests only.
func (s *Settings) flush() {
	done := make(chan struct{})
	s.writeQueue.Put(settingWrite{done: done})
	<-done
}

func (s *Settings) getString(key string, fallback string) string {
	v, ok := s.get(key)
	if !ok {
		return fallback
	}
	return v
}

func (s *Settings) setString(key string, v string) {
	s.set(key, v)
}

func (s *Settings) getBool(key string, fallback bool) bool {
	v, ok := s.get(key)
	if !ok {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func (s *Settings) setBool(key string, v bool) {
	s.set(key, strconv.FormatBool(v))
}

func (s *Settings) getInt(key string, fallback int) int {
	v, ok := s.get(key)
	if !ok {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func (s *Settings) setInt(key string, v int) {
	s.set(key, strconv.Itoa(v))
}

// getInt64 and setInt64 are used instead of getInt/setInt for values that can
// exceed the platform's native int range on 32-bit platforms (e.g. Android),
// such as EVE Online IDs.
func (s *Settings) getInt64(key string, fallback int64) int64 {
	v, ok := s.get(key)
	if !ok {
		return fallback
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fallback
	}
	return n
}

func (s *Settings) setInt64(key string, v int64) {
	s.set(key, strconv.FormatInt(v, 10))
}

func (s *Settings) getFloat(key string, fallback float64) float64 {
	v, ok := s.get(key)
	if !ok {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}

func (s *Settings) setFloat(key string, v float64) {
	s.set(key, strconv.FormatFloat(v, 'g', -1, 64))
}

// getTime and setTime store a time.Time as an RFC3339 string. fallback is
// returned both when key is absent and when the stored value fails to parse.
func (s *Settings) getTime(key string, fallback time.Time) time.Time {
	v, ok := s.get(key)
	if !ok {
		return fallback
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return fallback
	}
	return t
}

func (s *Settings) setTime(key string, v time.Time) {
	s.set(key, v.Format(time.RFC3339))
}

func (s *Settings) getStringList(key string, fallback []string) []string {
	v, ok := s.get(key)
	if !ok {
		return fallback
	}
	var out []string
	if err := json.Unmarshal([]byte(v), &out); err != nil {
		return fallback
	}
	return out
}

func (s *Settings) setStringList(key string, v []string) {
	b, err := json.Marshal(v)
	if err != nil {
		slog.Error("settings: failed to encode string list", "key", key, "error", err)
		return
	}
	s.set(key, string(b))
}

func (s *Settings) getFloatList(key string, fallback []float64) []float64 {
	v, ok := s.get(key)
	if !ok {
		return fallback
	}
	var out []float64
	if err := json.Unmarshal([]byte(v), &out); err != nil {
		return fallback
	}
	return out
}

func (s *Settings) setFloatList(key string, v []float64) {
	b, err := json.Marshal(v)
	if err != nil {
		slog.Error("settings: failed to encode float list", "key", key, "error", err)
		return
	}
	s.set(key, string(b))
}
