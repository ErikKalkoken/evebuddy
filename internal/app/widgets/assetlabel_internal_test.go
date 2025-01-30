package widgets

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitLines(t *testing.T) {
	const maxLine = 10
	cases := []struct {
		name  string
		in    string
		want1 string
		want2 string
	}{
		{"single line 1", "alpha", "alpha", ""},
		{"single line 2", "alpha boy", "alpha boy", ""},
		{"two lines single word", "verySophisticated", "verySophis", "ticated"},
		{"two lines long word", "verySophisticatedIndeed", "verySophis", "ticatedInd"},
		{"two lines", "first second", "first", "second"},
		{"two lines with truncation", "first second third", "first", "second thi"},
		{"one long word", "firstSecondThirdForth", "firstSecon", "dThirdFort"},
		{"special 1", "Erik Kalkoken's Cald", "Erik", "Kalkoken's"},
		// {"two lines two words", "Contaminated Nanite", "Contaminat", "ed Nanite"}, FIXME!
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got1, got2 := splitLines(tc.in, maxLine)
			assert.Equal(t, tc.want1, got1)
			assert.Equal(t, tc.want2, got2)
		})
	}
}
