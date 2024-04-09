package images_test

import (
	"example/evebuddy/internal/api/images"
	"fmt"

	"testing"

	"fyne.io/fyne/v2/storage"
	"github.com/stretchr/testify/assert"
)

var testCases = []struct {
	id    int32
	size  int
	valid bool
}{
	{42, -1, false},
	{42, 0, false},
	{42, 16, false},
	{42, 32, true},
	{42, 64, true},
	{42, 128, true},
	{42, 256, true},
	{42, 512, true},
	{42, 1024, true},
	{42, 2048, false},
}

func TestCharacterPortraitURL(t *testing.T) {
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("characterID:%d sizeID:%d", tc.id, tc.size), func(t *testing.T) {
			got, err := images.CharacterPortraitURL(tc.id, tc.size)
			if tc.valid && assert.NoError(t, err) {
				s := fmt.Sprintf("https://images.evetech.net/characters/%d/portrait?size=%d", tc.id, tc.size)
				want, _ := storage.ParseURI(s)
				assert.Equal(t, want, got)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestCorporationLogoURL(t *testing.T) {
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("characterID:%d sizeID:%d", tc.id, tc.size), func(t *testing.T) {
			got, err := images.CorporationLogoURL(tc.id, tc.size)
			if tc.valid && assert.NoError(t, err) {
				s := fmt.Sprintf("https://images.evetech.net/corporations/%d/logo?size=%d", tc.id, tc.size)
				want, _ := storage.ParseURI(s)
				assert.Equal(t, want, got)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
