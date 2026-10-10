package fynetools

import (
	"bytes"
	"image"
	"image/color"
	"image/png"

	"fyne.io/fyne/v2"
)

func ThemedPNG(in fyne.Resource, c color.Color) (fyne.Resource, error) {
	img, err := png.Decode(bytes.NewReader(in.Content()))
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
	r := &fyne.StaticResource{
		StaticName:    in.Name(),
		StaticContent: buf.Bytes(),
	}
	return r, nil
}
