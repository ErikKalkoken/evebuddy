package ui_test

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/stretchr/testify/assert"
)

func TestSettingsNoDuplicates(t *testing.T) {
	m := make(map[string]int)
	for _, s := range ui.SettingKeys() {
		m[s]++
	}
	for k, v := range m {
		assert.Equalf(t, 1, v, "duplicate setting key %s", k)
	}
}
