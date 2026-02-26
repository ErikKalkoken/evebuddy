package fynetools

import (
	"bytes"
	"image"
	"image/color"
	"image/png"

	"fyne.io/fyne/v2"
)

func ThemedPNG(in fyne.Resource, nc color.Color) (fyne.Resource, error) {
	img, _ := png.Decode(bytes.NewReader(in.Content()))
	bounds := img.Bounds()
	newImg := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			originalColor := img.At(x, y)
			_, _, _, a := originalColor.RGBA()

			if a > 0 {
				newImg.Set(x, y, nc)
			} else {
				newImg.Set(x, y, color.Transparent)
			}
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, newImg); err != nil {
		return nil, err
	}
	r := &fyne.StaticResource{
		StaticName:    in.Name(),
		StaticContent: buf.Bytes(),
	}
	return r, nil
}
