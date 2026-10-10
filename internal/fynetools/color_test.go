package fynetools_test

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/fynetools"
)

func TestToNRGBA(t *testing.T) {
	cases := []struct {
		name       string
		c          color.Color
		r, g, b, a int
	}{
		{"NRGBA", color.NRGBA{R: 10, G: 20, B: 30, A: 128}, 10, 20, 30, 128},
		{"*NRGBA", &color.NRGBA{R: 10, G: 20, B: 30, A: 128}, 10, 20, 30, 128},
		{"NRGBA64", color.NRGBA64{R: 0x0aff, G: 0x14ff, B: 0x1eff, A: 0x80ff}, 10, 20, 30, 128},
		{"*NRGBA64", &color.NRGBA64{R: 0x0aff, G: 0x14ff, B: 0x1eff, A: 0x80ff}, 10, 20, 30, 128},
		{"Gray", color.Gray{Y: 42}, 42, 42, 42, 255},
		{"*Gray", &color.Gray{Y: 42}, 42, 42, 42, 255},
		{"Gray16", color.Gray16{Y: 0x2aff}, 42, 42, 42, 255},
		{"*Gray16", &color.Gray16{Y: 0x2aff}, 42, 42, 42, 255},
		{"Alpha", color.Alpha{A: 99}, 255, 255, 255, 99},
		{"*Alpha", &color.Alpha{A: 99}, 255, 255, 255, 99},
		{"Alpha16", color.Alpha16{A: 0x63ff}, 255, 255, 255, 99},
		{"*Alpha16", &color.Alpha16{A: 0x63ff}, 255, 255, 255, 99},
		{"RGBA opaque", color.RGBA{R: 10, G: 20, B: 30, A: 255}, 10, 20, 30, 255},
		{"RGBA premultiplied", color.RGBA{R: 64, G: 32, B: 0, A: 128}, 127, 63, 0, 128},
		{"RGBA transparent", color.RGBA{}, 0, 0, 0, 0},
		{"RGBA64 premultiplied", color.RGBA64{R: 0x4040, A: 0x8080}, 127, 0, 0, 128},
		{"unknown type", color.CMYK{C: 255}, 0, 255, 255, 255},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, g, b, a := fynetools.ToNRGBA(tc.c)
			assert.Equal(t, []int{tc.r, tc.g, tc.b, tc.a}, []int{r, g, b, a})
		})
	}
}
