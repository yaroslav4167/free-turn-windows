package assets

import (
	"bytes"
	"image/png"
	"testing"
)

func TestApplicationImages(t *testing.T) {
	for _, icon := range []struct {
		size int
		data []byte
	}{{512, IconPNG}, {64, TrayIconPNG}, {64, TrayLightIconPNG}} {
		image, err := png.Decode(bytes.NewReader(icon.data))
		if err != nil {
			t.Fatal(err)
		}
		if image.Bounds().Dx() != icon.size || image.Bounds().Dy() != icon.size {
			t.Fatalf("expected %dx%d icon, got %v", icon.size, icon.size, image.Bounds())
		}
	}
	if !bytes.Contains(IconSVG, []byte("<svg")) {
		t.Fatal("missing SVG application icon")
	}
}
