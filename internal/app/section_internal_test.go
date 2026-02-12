package app

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestMakeSectionDisplayName(t *testing.T) {
	cases := []struct {
		section section
		want    string
	}{
		{SectionCharacterAssets, "Assets"},
		{SectionCorporationIndustryJobs, "Industry Jobs"},
		{SectionEveCharacters, "Characters"},
	}
	for _, tc := range cases {
		t.Run("can make display name for section", func(t *testing.T) {
			// when/then
		xassert.Equal(t, tc.want, tc.section.DisplayName())
		})
	}
}
