// Package fynetools contains tools for working with Fyne elements.
package fynetools

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"path"
	"strings"

	"fyne.io/fyne/v2"

	_ "image/jpeg" // required for the images package to support jpeg
	"image/png"
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
