package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app/pcache"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/stretchr/testify/assert"
)

func TestObfuscate(t *testing.T) {
	cases := []struct {
		name string
		s    string
		n    int
		want string
	}{
		{"normal", "123456789", 4, "XXXXX6789"},
		{"s too short", "123", 4, "XXX"},
		{"n is zero", "123456789", 0, "XXXXXXXXX"},
		{"n is negative", "123456789", -5, "XXXXXXXXX"},
		{"s is empty", "", 4, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := obfuscate(tc.s, tc.n, "X")
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCacheAdapter(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()
	pc := pcache.New(st, 0)
	ca := newCacheAdapter(pc, "prefix", 0)
	t.Run("get existing key", func(t *testing.T) {
		pc.Clear()
		ca.Set("a", []byte("alpha"))
		got, ok := ca.Get("a")
		if assert.True(t, ok) {
			assert.Equal(t, []byte("alpha"), got)
		}
	})
	t.Run("get non existing key", func(t *testing.T) {
		pc.Clear()
		_, ok := ca.Get("a")
		assert.False(t, ok)
	})
	t.Run("delete existing key", func(t *testing.T) {
		pc.Clear()
		ca.Set("a", []byte("alpha"))
		ca.Delete("a")
		_, ok := ca.Get("a")
		assert.False(t, ok)
	})
}

func TestSetupCrashFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), crashFileName)
	err := setupCrashFile(p)
	if assert.NoError(t, err) {
		_, err := os.Stat(p)
		assert.NoError(t, err)
	}
}
