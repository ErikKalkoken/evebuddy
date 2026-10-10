package fynetools_test

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"fyne.io/fyne/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/fynetools"
)

func TestImageToGrayscale(t *testing.T) {
	colors := []color.NRGBA{
		{R: 255, A: 255},
		{G: 255, A: 255},
		{B: 255, A: 255},
		{R: 255, G: 255, B: 255, A: 255},
		{A: 255},
	}
	makeImage := func() *image.NRGBA {
		img := image.NewNRGBA(image.Rect(0, 0, len(colors), 1))
		for x, c := range colors {
			img.SetNRGBA(x, 0, c)
		}
		return img
	}

	t.Run("should convert PNG to grayscale", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, png.Encode(&buf, makeImage()))
		in := fyne.NewStaticResource("icon.png", buf.Bytes())

		got, err := fynetools.ImageToGrayscale(in)
		require.NoError(t, err)
		assert.Equal(t, "icon.png", got.Name())

		img, format, err := image.Decode(bytes.NewReader(got.Content()))
		require.NoError(t, err)
		assert.Equal(t, "png", format)
		assert.Equal(t, image.Rect(0, 0, len(colors), 1), img.Bounds())
		var lums []uint8
		for x := range colors {
			c := color.NRGBAModel.Convert(img.At(x, 0)).(color.NRGBA)
			assert.Equal(t, c.R, c.G, "x=%d", x)
			assert.Equal(t, c.R, c.B, "x=%d", x)
			assert.Equal(t, uint8(255), c.A, "x=%d", x)
			lums = append(lums, c.R)
		}
		assert.Equal(t, uint8(255), lums[3], "white stays white")
		assert.Equal(t, uint8(0), lums[4], "black stays black")
		assert.Greater(t, lums[1], lums[0], "green is brighter than red")
		assert.Greater(t, lums[0], lums[2], "red is brighter than blue")
	})

	t.Run("should preserve transparency of PNG", func(t *testing.T) {
		img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
		img.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 0})
		img.SetNRGBA(1, 0, color.NRGBA{R: 255, G: 255, B: 255, A: 128})
		var buf bytes.Buffer
		require.NoError(t, png.Encode(&buf, img))
		in := fyne.NewStaticResource("icon.png", buf.Bytes())

		got, err := fynetools.ImageToGrayscale(in)
		require.NoError(t, err)

		out, err := png.Decode(bytes.NewReader(got.Content()))
		require.NoError(t, err)
		assert.Equal(t, uint8(0), color.NRGBAModel.Convert(out.At(0, 0)).(color.NRGBA).A)
		assert.Equal(t, uint8(128), color.NRGBAModel.Convert(out.At(1, 0)).(color.NRGBA).A)
	})

	t.Run("should convert JPEG to grayscale", func(t *testing.T) {
		img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
		for y := range 16 {
			for x := range 16 {
				img.SetNRGBA(x, y, color.NRGBA{R: 200, G: 50, B: 100, A: 255})
			}
		}
		var buf bytes.Buffer
		require.NoError(t, jpeg.Encode(&buf, img, nil))
		in := fyne.NewStaticResource("photo.jpg", buf.Bytes())

		got, err := fynetools.ImageToGrayscale(in)
		require.NoError(t, err)
		assert.Equal(t, "photo.jpg", got.Name())

		out, format, err := image.Decode(bytes.NewReader(got.Content()))
		require.NoError(t, err)
		assert.Equal(t, "jpeg", format)
		assert.Equal(t, image.Rect(0, 0, 16, 16), out.Bounds())
		c := color.NRGBAModel.Convert(out.At(8, 8)).(color.NRGBA)
		// JPEG is lossy, so allow small deviations between channels
		assert.InDelta(t, int(c.R), int(c.G), 3)
		assert.InDelta(t, int(c.R), int(c.B), 3)
	})

	t.Run("should return error when resource is not an image", func(t *testing.T) {
		in := fyne.NewStaticResource("bad.png", []byte("not an image"))
		_, err := fynetools.ImageToGrayscale(in)
		assert.Error(t, err)
	})
}
