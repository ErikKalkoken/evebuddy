package evenotification

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

func TestMakeInfoLink2(t *testing.T) {
	cases := []struct {
		name     string
		label    string
		linkData []any
		want     string
	}{
		{"2 elements", "Jita IV", []any{"showinfo", uint64(60003760)}, "[Jita IV](showinfo:60003760)"},
		{"3 elements", "Jita IV", []any{"showinfo", uint64(3802), uint64(60003760)}, "[Jita IV](showinfo:3802//60003760)"},
		{"wrong length 1", "Jita IV", []any{"showinfo"}, "Jita IV"},
		{"wrong length 4", "Jita IV", []any{"showinfo", 1, 2, 3}, "Jita IV"},
		{"not showinfo", "Jita IV", []any{"other", uint64(60003760)}, "Jita IV"},
		{"empty", "Jita IV", []any{}, "Jita IV"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := makeInfoLink2(tc.label, tc.linkData)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestMakeInfoLink(t *testing.T) {
	t.Run("returns ? when entity is nil", func(t *testing.T) {
		var e *app.EveEntity
		got := makeInfoLink(e)
		assert.Equal(t, "?", got)
	})
	t.Run("returns name when link can not be constructed and name is set", func(t *testing.T) {
		e := &app.EveEntity{ID: 42, Name: "Unknown Thing", Category: app.EveEntityUndefined}
		got := makeInfoLink(e)
		assert.Equal(t, "Unknown Thing", got)
	})
	t.Run("returns ? when link can not be constructed and name is empty", func(t *testing.T) {
		e := &app.EveEntity{ID: 42, Category: app.EveEntityUndefined}
		got := makeInfoLink(e)
		assert.Equal(t, "?", got)
	})
	t.Run("returns markdown link for a resolvable entity", func(t *testing.T) {
		e := &app.EveEntity{ID: 42, Name: "Bob", Category: app.EveEntityCharacter}
		got := makeInfoLink(e)
		assert.Equal(t, "[Bob](showinfo:1373//42)", got)
	})
}

func TestMakeMarkDownLink(t *testing.T) {
	t.Run("returns label when url is empty", func(t *testing.T) {
		got := makeMarkDownLink("Bob", "")
		assert.Equal(t, "Bob", got)
	})
	t.Run("returns markdown link when url is set", func(t *testing.T) {
		got := makeMarkDownLink("Bob", "showinfo:1377//42")
		assert.Equal(t, "[Bob](showinfo:1377//42)", got)
	})
}

func TestLinkDataUint64(t *testing.T) {
	t.Run("returns value for valid index and type", func(t *testing.T) {
		got, err := linkDataUint64("ctx", []any{"x", uint64(123)}, 1)
		if assert.NoError(t, err) {
			assert.Equal(t, uint64(123), got)
		}
	})
	t.Run("returns error when index out of range", func(t *testing.T) {
		_, err := linkDataUint64("ctx", []any{"x"}, 5)
		assert.Error(t, err)
	})
	t.Run("returns error when index negative", func(t *testing.T) {
		_, err := linkDataUint64("ctx", []any{"x"}, -1)
		assert.Error(t, err)
	})
	t.Run("returns error when wrong type", func(t *testing.T) {
		_, err := linkDataUint64("ctx", []any{"x", "not a uint64"}, 1)
		assert.Error(t, err)
	})
}

func TestLinkDataString(t *testing.T) {
	t.Run("returns value for valid index and type", func(t *testing.T) {
		got, err := linkDataString("ctx", []any{"x", "hello"}, 1)
		if assert.NoError(t, err) {
			assert.Equal(t, "hello", got)
		}
	})
	t.Run("returns error when index out of range", func(t *testing.T) {
		_, err := linkDataString("ctx", []any{"x"}, 5)
		assert.Error(t, err)
	})
	t.Run("returns error when index negative", func(t *testing.T) {
		_, err := linkDataString("ctx", []any{"x"}, -1)
		assert.Error(t, err)
	})
	t.Run("returns error when wrong type", func(t *testing.T) {
		_, err := linkDataString("ctx", []any{"x", uint64(123)}, 1)
		assert.Error(t, err)
	})
}
