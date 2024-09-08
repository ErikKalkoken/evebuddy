package appdirs_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/appdirs"
	"github.com/stretchr/testify/assert"
)

func TestAppDirsDeleteAll(t *testing.T) {
	// given
	ap := appdirs.AppDirs{
		Cache:    t.TempDir(),
		Data:     t.TempDir(),
		Log:      t.TempDir(),
		Settings: t.TempDir(),
	}
	paths := []string{ap.Cache, ap.Data, ap.Log, ap.Settings}
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
	ap.DeleteAll()
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
