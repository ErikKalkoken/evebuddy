package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"path"
	"strings"

	"fyne.io/fyne/v2"

	_ "image/jpeg"
	"image/png"
)

type circle struct {
	p image.Point
	r int
}

func (c *circle) ColorModel() color.Model {
	return color.AlphaModel
}

func (c *circle) Bounds() image.Rectangle {
	return image.Rect(c.p.X-c.r, c.p.Y-c.r, c.p.X+c.r, c.p.Y+c.r)
}

func (c *circle) At(x, y int) color.Color {
	xx, yy, rr := float64(x-c.p.X)+0.5, float64(y-c.p.Y)+0.5, float64(c.r)
	if xx*xx+yy*yy < rr*rr {
		return color.Alpha{255}
	}
	return color.Alpha{0}
}

// applyCircleMask creates a new image from a round shape within the original
func applyCircleMask(source image.Image, origin image.Point, r int) image.Image {
	c := &circle{origin, r}
	result := image.NewRGBA(c.Bounds())
	draw.DrawMask(result, source.Bounds(), source, image.Point{}, c, image.Point{}, draw.Over)
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
	w := (b.Max.X - b.Min.X) / 2
	h := (b.Max.Y - b.Min.Y) / 2
	r := min(w, h)
	m2 := applyCircleMask(m, image.Point{X: w, Y: h}, r)

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
