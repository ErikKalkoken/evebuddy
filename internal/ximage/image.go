// Package ximage contains tools for transforming image resources.
package ximage

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"path"
	"strings"

	"fyne.io/fyne/v2"
	"github.com/anthonynsimon/bild/effect"
)

// circle is an alpha mask of a circle filling a square.
type circle struct {
	rect image.Rectangle
}

func (c *circle) ColorModel() color.Model {
	return color.AlphaModel
}

func (c *circle) Bounds() image.Rectangle {
	return c.rect
}

func (c *circle) At(x, y int) color.Color {
	// center can be fractional, so odd-sized squares keep all pixels
	r := float64(c.rect.Dx()) / 2
	xx := float64(x-c.rect.Min.X) + 0.5 - r
	yy := float64(y-c.rect.Min.Y) + 0.5 - r
	if xx*xx+yy*yy < r*r {
		return color.Alpha{255}
	}
	return color.Alpha{0}
}

// applyCircleMask creates a new image from a round shape within the square rect of the original.
func applyCircleMask(source image.Image, rect image.Rectangle) image.Image {
	c := &circle{rect}
	result := image.NewRGBA(rect)
	draw.DrawMask(result, rect, source, rect.Min, c, rect.Min, draw.Over)
	return result
}

// MakeAvatar creates a rounded avatar style image from a resource and returns it.
func MakeAvatar(in fyne.Resource) (fyne.Resource, error) {
	// decode
	reader := bytes.NewReader(in.Content())
	m, _, err := image.Decode(reader)
	if err != nil {
		return nil, err
	}

	// convert
	b := m.Bounds()
	s := min(b.Dx(), b.Dy())
	x0 := b.Min.X + (b.Dx()-s)/2
	y0 := b.Min.Y + (b.Dy()-s)/2
	m2 := applyCircleMask(m, image.Rect(x0, y0, x0+s, y0+s))

	// encode new image
	var buf bytes.Buffer
	if err := png.Encode(&buf, m2); err != nil {
		return nil, err
	}
	name := in.Name()
	name = strings.TrimSuffix(name, path.Ext(name))
	name += "_avatar.png"
	out := fyne.NewStaticResource(name, buf.Bytes())
	return out, nil
}

// ToGrayscale returns a copy of an image in grayscale.
//
// Will fail if the resource is not a PNG or JPEG image.
func ToGrayscale(r fyne.Resource) (fyne.Resource, error) {
	j, format, err := image.Decode(bytes.NewReader(r.Content()))
	if err != nil {
		return nil, err
	}
	b := j.Bounds()
	m := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(m, m.Bounds(), j, b.Min, draw.Src)
	m = effect.Grayscale(m)
	var byt bytes.Buffer
	switch format {
	case "jpeg":
		err = jpeg.Encode(&byt, m, nil)
	case "png":
		err = png.Encode(&byt, m)
	default:
		err = fmt.Errorf("unsupported image format: %s", format)
	}
	if err != nil {
		return nil, err
	}
	return fyne.NewStaticResource(r.Name(), byt.Bytes()), nil
}

// Tint returns a copy of an image with all pixels set to color c, keeping their alpha.
// The result is always a PNG and its name gets a .png extension.
func Tint(in fyne.Resource, c color.Color) (fyne.Resource, error) {
	img, _, err := image.Decode(bytes.NewReader(in.Content()))
	if err != nil {
		return nil, err
	}
	bounds := img.Bounds()
	newImg := image.NewNRGBA(bounds)
	target := color.NRGBAModel.Convert(c).(color.NRGBA)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			// keep the original alpha so anti-aliased edges stay smooth
			_, _, _, a := img.At(x, y).RGBA()
			px := target
			px.A = uint8(uint32(target.A) * a / 0xffff)
			newImg.SetNRGBA(x, y, px)
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, newImg); err != nil {
		return nil, err
	}
	name := strings.TrimSuffix(in.Name(), path.Ext(in.Name())) + ".png"
	return fyne.NewStaticResource(name, buf.Bytes()), nil
}
