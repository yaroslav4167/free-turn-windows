//go:build linux

package tray

import (
	"bytes"
	"image/color"
	"image/png"

	appassets "github.com/samosvalishe/free-turn-proxy/desktop/assets"
)

type iconPixmap struct {
	Width  int32
	Height int32
	Pixels []byte
}

func applicationIcon(light bool) ([]iconPixmap, error) {
	data := appassets.TrayIconPNG
	if light {
		data = appassets.TrayLightIconPNG
	}
	image, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	bounds := image.Bounds()
	pixels := make([]byte, 0, bounds.Dx()*bounds.Dy()*4)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			pixel := color.NRGBAModel.Convert(image.At(x, y)).(color.NRGBA)
			// StatusNotifierItem uses ARGB32 in network (big-endian) order.
			pixels = append(pixels, pixel.A, pixel.R, pixel.G, pixel.B)
		}
	}
	return []iconPixmap{{int32(bounds.Dx()), int32(bounds.Dy()), pixels}}, nil
}
