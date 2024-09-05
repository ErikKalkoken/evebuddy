package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUninstall(t *testing.T) {
	// given
	paths := make([]string, 0)
	for range 3 {
		paths = append(paths, t.TempDir())
	}
	for _, p := range paths {
		x := filepath.Join(p, "dummy.txt")
		if err := os.WriteFile(x, []byte("dummy"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range paths {
		assert.True(t, Exists(p))
	}
	// when
	uninstall(paths[0], paths[1], paths[2])
	// then
	for _, p := range paths {
		assert.False(t, Exists(p))
	}
}

func Exists(name string) bool {
	_, err := os.Stat(name)
	if err == nil {
		return true
	}
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	panic(err)
}
