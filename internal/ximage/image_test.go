package ximage_test

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

	"github.com/ErikKalkoken/evebuddy/internal/ximage"
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

		got, err := ximage.MakeAvatar(in)
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

		got, err := ximage.MakeAvatar(in)
		require.NoError(t, err)

		img, err := png.Decode(bytes.NewReader(got.Content()))
		require.NoError(t, err)
		assert.Equal(t, image.Rect(0, 0, 10, 10), img.Bounds())
	})

	t.Run("should keep all pixels of an odd-sized image and center the circle", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, png.Encode(&buf, makeImage(5, 5)))
		in := fyne.NewStaticResource("icon.png", buf.Bytes())

		got, err := ximage.MakeAvatar(in)
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

		got, err := ximage.MakeAvatar(in)
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

		got, err := ximage.MakeAvatar(in)
		require.NoError(t, err)

		img, err := png.Decode(bytes.NewReader(got.Content()))
		require.NoError(t, err)
		assert.Equal(t, image.Rect(0, 0, 10, 10), img.Bounds())
	})

	t.Run("should accept JPEG and always return PNG", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, jpeg.Encode(&buf, makeImage(10, 10), nil))
		in := fyne.NewStaticResource("photo.jpg", buf.Bytes())

		got, err := ximage.MakeAvatar(in)
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

		got, err := ximage.MakeAvatar(in)
		require.NoError(t, err)
		assert.Equal(t, "icon_avatar.png", got.Name())
	})

	t.Run("should return error when resource is not an image", func(t *testing.T) {
		in := fyne.NewStaticResource("bad.png", []byte("not an image"))
		_, err := ximage.MakeAvatar(in)
		assert.Error(t, err)
	})
}

func TestToGrayscale(t *testing.T) {
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

		got, err := ximage.ToGrayscale(in)
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

		got, err := ximage.ToGrayscale(in)
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

		got, err := ximage.ToGrayscale(in)
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
		_, err := ximage.ToGrayscale(in)
		assert.Error(t, err)
	})
}

func TestTint(t *testing.T) {
	red := color.NRGBA{R: 255, A: 255}

	t.Run("should recolor pixels and keep the name", func(t *testing.T) {
		in := makePNGResource(t, "icon.png", 2, 1, func(x, y int) color.NRGBA {
			return color.NRGBA{R: 10, G: 20, B: 30, A: 255}
		})
		got, err := ximage.Tint(in, red)
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
		got, err := ximage.Tint(in, red)
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
		got, err := ximage.Tint(in, color.NRGBA{B: 255, A: 128})
		require.NoError(t, err)
		img := decodePNG(t, got)
		assert.Equal(t, uint8(128), color.NRGBAModel.Convert(img.At(0, 0)).(color.NRGBA).A)
		assert.Equal(t, uint8(64), color.NRGBAModel.Convert(img.At(1, 0)).(color.NRGBA).A)
	})

	t.Run("should accept premultiplied target colors", func(t *testing.T) {
		in := makePNGResource(t, "icon.png", 1, 1, func(x, y int) color.NRGBA {
			return color.NRGBA{A: 255}
		})
		got, err := ximage.Tint(in, color.RGBA{R: 100, A: 200})
		require.NoError(t, err)
		img := decodePNG(t, got)
		c := color.NRGBAModel.Convert(img.At(0, 0)).(color.NRGBA)
		assert.Equal(t, uint8(200), c.A)
		assert.InDelta(t, 127, int(c.R), 1)
	})

	t.Run("should accept JPEG and return PNG with .png name", func(t *testing.T) {
		img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
		for y := range 4 {
			for x := range 4 {
				img.SetNRGBA(x, y, color.NRGBA{G: 200, A: 255})
			}
		}
		var buf bytes.Buffer
		require.NoError(t, jpeg.Encode(&buf, img, nil))
		in := fyne.NewStaticResource("photo.jpg", buf.Bytes())

		got, err := ximage.Tint(in, red)
		require.NoError(t, err)
		assert.Equal(t, "photo.png", got.Name())

		out, format, err := image.Decode(bytes.NewReader(got.Content()))
		require.NoError(t, err)
		assert.Equal(t, "png", format)
		assert.Equal(t, red, color.NRGBAModel.Convert(out.At(2, 2)))
	})

	t.Run("should return error when resource is not an image", func(t *testing.T) {
		in := fyne.NewStaticResource("bad.png", []byte("not an image"))
		_, err := ximage.Tint(in, red)
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
