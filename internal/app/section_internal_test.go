package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMakeSectionDisplayName(t *testing.T) {
	cases := []struct {
		section Section
		want    string
	}{
		{SectionCharacterAssets, "Assets"},
		{SectionCorporationIndustryJobs, "Industry Jobs"},
		{SectionEveCharacters, "Characters"},
	}
	for _, tc := range cases {
		t.Run("can make display name for section", func(t *testing.T) {
			// when/then
			assert.Equal(t, tc.want, tc.section.DisplayName())
		})
	}
}
