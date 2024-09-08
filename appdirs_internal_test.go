package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeleteAll(t *testing.T) {
	// given
	ap := appDirs{
		cache:    t.TempDir(),
		data:     t.TempDir(),
		log:      t.TempDir(),
		settings: t.TempDir(),
	}
	paths := []string{ap.cache, ap.data, ap.log, ap.settings}
	for _, p := range paths {
		x := filepath.Join(p, "dummy.txt")
		if err := os.WriteFile(x, []byte("dummy"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range paths {
		assert.True(t, fileExists(p))
	}
	// when
	ap.deleteAll()
	// then
	for _, p := range paths {
		assert.False(t, fileExists(p))
	}
}

func fileExists(name string) bool {
	_, err := os.Stat(name)
	if err == nil {
		return true
	}
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	panic(err)
}
