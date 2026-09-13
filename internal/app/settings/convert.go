package settings

import (
	"encoding/json"
	"log/slog"
	"strconv"
)

// get returns the raw string value for key, and whether it was present.
func (s *Settings) get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.values[key]
	return v, ok
}

// set stores value for key in the in-memory cache and persists it to storage,
// logging (but not returning) any persistence error.
func (s *Settings) set(key, value string) {
	s.mu.Lock()
	s.values[key] = value
	s.mu.Unlock()
	if err := s.st.SetSetting(s.ctx, key, value); err != nil {
		slog.Error("settings: failed to persist value", "key", key, "error", err)
	}
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
