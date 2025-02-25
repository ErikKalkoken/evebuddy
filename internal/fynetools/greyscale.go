package fynetools

import (
	"bytes"
	"image"
	"image/draw"
	"image/png"

	"image/jpeg"

	"fyne.io/fyne/v2"
	"github.com/anthonynsimon/bild/effect"
)

// ImageToGreyscale returns a copy of an image in greyscale.
//
// Will fail if the resource it not a PNG or JPEG image.
func ImageToGreyscale(r fyne.Resource) (fyne.Resource, error) {
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
	}
	if err != nil {
		return nil, err
	}
	return fyne.NewStaticResource(r.Name(), byt.Bytes()), nil
}
