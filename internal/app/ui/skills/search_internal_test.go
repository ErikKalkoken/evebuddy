package skills

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestSkillNameCondensed(t *testing.T) {
	tests := []struct {
		name string
		row  searchRow
		want string
	}{
		{
			name: "active level zero and trained level greater than zero",
			row: searchRow{
				typeName:          "Gunnery",
				activeLevel:       0,
				activeLevelRoman:  "0",
				trainedLevel:      3,
				trainedLevelRoman: "III",
			},
			want: "Gunnery [III]",
		},
		{
			name: "active level equal to trained level",
			row: searchRow{
				typeName:          "Gunnery",
				activeLevel:       5,
				activeLevelRoman:  "V",
				trainedLevel:      5,
				trainedLevelRoman: "V",
			},
			want: "Gunnery V",
		},
		{
			name: "active level different from trained level",
			row: searchRow{
				typeName:          "Gunnery",
				activeLevel:       3,
				activeLevelRoman:  "III",
				trainedLevel:      5,
				trainedLevelRoman: "V",
			},
			want: "Gunnery III [V]",
		},
		{
			name: "both levels zero",
			row: searchRow{
				typeName:          "Gunnery",
				activeLevel:       0,
				activeLevelRoman:  "-",
				trainedLevel:      0,
				trainedLevelRoman: "-",
			},
			want: "Gunnery -",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.row.skillNameCondensed()
			xassert.Equal(t, tt.want, got)
		})
	}
}
