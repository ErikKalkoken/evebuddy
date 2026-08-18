package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetupCrashFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), crashFileName)
	err := setupCrashFile(p)
	if assert.NoError(t, err) {
		_, err := os.Stat(p)
		assert.NoError(t, err)
	}
}
