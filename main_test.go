package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupCrashFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), crashFileName)
	err := setupCrashFile(p)
	require.NoError(t, err)
	_, err = os.Stat(p)
	assert.NoError(t, err)
}
