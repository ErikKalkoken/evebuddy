package screens

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSkillsForClipboard(t *testing.T) {
	t.Run("formats sorted active skills as name and level", func(t *testing.T) {
		rows := []skillCatalogueRow{
			{name: "Gunnery", levelActive: 4},
			{name: "Caldari Frigate", levelActive: 5},
			{name: "Spaceship Command", levelActive: 3},
		}
		got, err := skillsForClipboard(rows)
		if assert.NoError(t, err) {
			want := "Caldari Frigate 5\nGunnery 4\nSpaceship Command 3\n"
			assert.Equal(t, want, got)
		}
	})
	t.Run("skips skills that are not active", func(t *testing.T) {
		rows := []skillCatalogueRow{
			{name: "Gunnery", levelActive: 4},
			{name: "Caldari Frigate", levelActive: 0},
		}
		got, err := skillsForClipboard(rows)
		if assert.NoError(t, err) {
			want := "Gunnery 4\n"
			assert.Equal(t, want, got)
		}
	})
	t.Run("returns empty string for no rows", func(t *testing.T) {
		got, err := skillsForClipboard(nil)
		if assert.NoError(t, err) {
			assert.Equal(t, "", got)
		}
	})
}
