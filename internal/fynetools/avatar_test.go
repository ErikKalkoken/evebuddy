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

func TestMakeAvatar(t *testing.T) {
	red := color.NRGBA{R: 255, A: 255}
	makeImage := func(w, h int) *image.NRGBA {
		img := image.NewNRGBA(image.Rect(0, 0, w, h))
		for y := range h {
			for x := range w {
				img.SetNRGBA(x, y, red)
			}
		}
		return img
	}
	alphaAt := func(img image.Image, x, y int) uint8 {
		return color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA).A
	}

	t.Run("should cut a circle from a square PNG", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, png.Encode(&buf, makeImage(10, 10)))
		in := fyne.NewStaticResource("icon.png", buf.Bytes())

		got, err := fynetools.MakeAvatar(in)
		require.NoError(t, err)
		assert.Equal(t, "icon_avatar.png", got.Name())

		img, format, err := image.Decode(bytes.NewReader(got.Content()))
		require.NoError(t, err)
		assert.Equal(t, "png", format)
		assert.Equal(t, image.Rect(0, 0, 10, 10), img.Bounds())
		assert.Equal(t, red, color.NRGBAModel.Convert(img.At(5, 5)), "center")
		assert.Equal(t, uint8(255), alphaAt(img, 0, 5), "left edge middle")
		assert.Equal(t, uint8(255), alphaAt(img, 5, 0), "top edge middle")
		for _, p := range []image.Point{{0, 0}, {9, 0}, {0, 9}, {9, 9}} {
			assert.Equal(t, uint8(0), alphaAt(img, p.X, p.Y), "corner %v", p)
		}
	})

	t.Run("should crop a non-square image to a square around its center", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, png.Encode(&buf, makeImage(20, 10)))
		in := fyne.NewStaticResource("icon.png", buf.Bytes())

		got, err := fynetools.MakeAvatar(in)
		require.NoError(t, err)

		img, err := png.Decode(bytes.NewReader(got.Content()))
		require.NoError(t, err)
		assert.Equal(t, image.Rect(0, 0, 10, 10), img.Bounds())
	})

	t.Run("should keep all pixels of an odd-sized image and center the circle", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, png.Encode(&buf, makeImage(5, 5)))
		in := fyne.NewStaticResource("icon.png", buf.Bytes())

		got, err := fynetools.MakeAvatar(in)
		require.NoError(t, err)

		img, err := png.Decode(bytes.NewReader(got.Content()))
		require.NoError(t, err)
		assert.Equal(t, image.Rect(0, 0, 5, 5), img.Bounds())
		assert.Equal(t, red, color.NRGBAModel.Convert(img.At(2, 2)), "center")
		assert.Equal(t, uint8(0), alphaAt(img, 0, 0), "corner")
		for y := range 5 {
			for x := range 5 {
				a := alphaAt(img, x, y)
				assert.Equal(t, a, alphaAt(img, 4-x, y), "mirror x at %d,%d", x, y)
				assert.Equal(t, a, alphaAt(img, x, 4-y), "mirror y at %d,%d", x, y)
			}
		}
	})

	t.Run("should handle a 1x1 image", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, png.Encode(&buf, makeImage(1, 1)))
		in := fyne.NewStaticResource("icon.png", buf.Bytes())

		got, err := fynetools.MakeAvatar(in)
		require.NoError(t, err)

		img, err := png.Decode(bytes.NewReader(got.Content()))
		require.NoError(t, err)
		assert.Equal(t, image.Rect(0, 0, 1, 1), img.Bounds())
		assert.Equal(t, red, color.NRGBAModel.Convert(img.At(0, 0)))
	})

	t.Run("should crop an image with odd size difference", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, png.Encode(&buf, makeImage(21, 10)))
		in := fyne.NewStaticResource("icon.png", buf.Bytes())

		got, err := fynetools.MakeAvatar(in)
		require.NoError(t, err)

		img, err := png.Decode(bytes.NewReader(got.Content()))
		require.NoError(t, err)
		assert.Equal(t, image.Rect(0, 0, 10, 10), img.Bounds())
	})

	t.Run("should accept JPEG and always return PNG", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, jpeg.Encode(&buf, makeImage(10, 10), nil))
		in := fyne.NewStaticResource("photo.jpg", buf.Bytes())

		got, err := fynetools.MakeAvatar(in)
		require.NoError(t, err)
		assert.Equal(t, "photo_avatar.png", got.Name())

		_, format, err := image.Decode(bytes.NewReader(got.Content()))
		require.NoError(t, err)
		assert.Equal(t, "png", format)
	})

	t.Run("should append suffix when name has no extension", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, png.Encode(&buf, makeImage(4, 4)))
		in := fyne.NewStaticResource("icon", buf.Bytes())

		got, err := fynetools.MakeAvatar(in)
		require.NoError(t, err)
		assert.Equal(t, "icon_avatar.png", got.Name())
	})

	t.Run("should return error when resource is not an image", func(t *testing.T) {
		in := fyne.NewStaticResource("bad.png", []byte("not an image"))
		_, err := fynetools.MakeAvatar(in)
		assert.Error(t, err)
	})
}
