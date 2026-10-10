package fynetools_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"fyne.io/fyne/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/fynetools"
)

func TestThemedPNG(t *testing.T) {
	red := color.NRGBA{R: 255, A: 255}

	t.Run("should recolor pixels and keep the name", func(t *testing.T) {
		in := makePNGResource(t, "icon.png", 2, 1, func(x, y int) color.NRGBA {
			return color.NRGBA{R: 10, G: 20, B: 30, A: 255}
		})
		got, err := fynetools.ThemedPNG(in, red)
		require.NoError(t, err)
		assert.Equal(t, "icon.png", got.Name())
		img := decodePNG(t, got)
		assert.Equal(t, image.Rect(0, 0, 2, 1), img.Bounds())
		for x := range 2 {
			assert.Equal(t, red, color.NRGBAModel.Convert(img.At(x, 0)))
		}
	})

	t.Run("should preserve source alpha", func(t *testing.T) {
		alphas := []uint8{0, 128, 255}
		in := makePNGResource(t, "icon.png", len(alphas), 1, func(x, y int) color.NRGBA {
			return color.NRGBA{G: 255, A: alphas[x]}
		})
		got, err := fynetools.ThemedPNG(in, red)
		require.NoError(t, err)
		img := decodePNG(t, got)
		for x, want := range alphas {
			c := color.NRGBAModel.Convert(img.At(x, 0)).(color.NRGBA)
			assert.Equal(t, want, c.A, "x=%d", x)
			if want > 0 {
				assert.Equal(t, red.R, c.R, "x=%d", x)
				assert.Equal(t, red.G, c.G, "x=%d", x)
				assert.Equal(t, red.B, c.B, "x=%d", x)
			}
		}
	})

	t.Run("should combine source alpha with target alpha", func(t *testing.T) {
		in := makePNGResource(t, "icon.png", 2, 1, func(x, y int) color.NRGBA {
			if x == 0 {
				return color.NRGBA{A: 255}
			}
			return color.NRGBA{A: 128}
		})
		got, err := fynetools.ThemedPNG(in, color.NRGBA{B: 255, A: 128})
		require.NoError(t, err)
		img := decodePNG(t, got)
		assert.Equal(t, uint8(128), color.NRGBAModel.Convert(img.At(0, 0)).(color.NRGBA).A)
		assert.Equal(t, uint8(64), color.NRGBAModel.Convert(img.At(1, 0)).(color.NRGBA).A)
	})

	t.Run("should accept premultiplied target colors", func(t *testing.T) {
		in := makePNGResource(t, "icon.png", 1, 1, func(x, y int) color.NRGBA {
			return color.NRGBA{A: 255}
		})
		got, err := fynetools.ThemedPNG(in, color.RGBA{R: 100, A: 200})
		require.NoError(t, err)
		img := decodePNG(t, got)
		c := color.NRGBAModel.Convert(img.At(0, 0)).(color.NRGBA)
		assert.Equal(t, uint8(200), c.A)
		assert.InDelta(t, 127, int(c.R), 1)
	})

	t.Run("should return error when resource is not a PNG", func(t *testing.T) {
		in := fyne.NewStaticResource("bad.png", []byte("not a png"))
		_, err := fynetools.ThemedPNG(in, red)
		assert.Error(t, err)
	})
}

func makePNGResource(t *testing.T, name string, w, h int, f func(x, y int) color.NRGBA) fyne.Resource {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.SetNRGBA(x, y, f(x, y))
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return fyne.NewStaticResource(name, buf.Bytes())
}

func decodePNG(t *testing.T, r fyne.Resource) image.Image {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(r.Content()))
	require.NoError(t, err)
	return img
}
