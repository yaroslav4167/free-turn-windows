//go:build linux

package tray

import (
	"bufio"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/prop"
)

func TestLightIconFollowsPortalAppearance(t *testing.T) {
	for _, test := range []struct {
		name           string
		value          interface{}
		fallback, want bool
	}{
		{"dark theme", uint32(1), false, true},
		{"light theme", uint32(2), true, false},
		{"dark variant", dbus.MakeVariant(uint32(1)), false, true},
		{"nested light variant", dbus.MakeVariant(dbus.MakeVariant(uint32(2))), true, false},
		{"no preference dark fallback", uint32(0), true, true},
		{"no preference normal fallback", uint32(0), false, false},
		{"unknown scheme", uint32(99), false, false},
		{"invalid value", "dark", false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := lightIconForScheme(test.value, test.fallback); got != test.want {
				t.Fatalf("light icon = %v, want %v", got, test.want)
			}
		})
	}
}

func TestIconChangesNotifyTheTrayHost(t *testing.T) {
	daemon, err := exec.LookPath("dbus-daemon")
	if err != nil {
		t.Skip("dbus-daemon is needed for an isolated test bus")
	}
	cmd := exec.Command(daemon, "--session", "--nofork", "--print-address=1")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	address, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	connection, err := dbus.Connect(strings.TrimSpace(address))
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	observer, err := dbus.Connect(strings.TrimSpace(address))
	if err != nil {
		t.Fatal(err)
	}
	defer observer.Close()
	normal, _ := applicationIcon(false)
	light, _ := applicationIcon(true)
	properties, err := prop.Export(connection, "/StatusNotifierItem", prop.Map{itemInterface: {
		"IconPixmap": {Value: normal, Writable: false, Emit: prop.EmitFalse},
	}})
	if err != nil {
		t.Fatal(err)
	}
	signals := make(chan *dbus.Signal, 4)
	observer.Signal(signals)
	if err := observer.AddMatchSignal(dbus.WithMatchObjectPath("/StatusNotifierItem"), dbus.WithMatchInterface(itemInterface), dbus.WithMatchMember("NewIcon")); err != nil {
		t.Fatal(err)
	}
	tray := &Tray{conn: connection, iconProperties: properties, icons: [2][]iconPixmap{normal, light}}
	for _, preferLight := range []bool{true, false} {
		tray.updateIcon(preferLight)
		select {
		case <-signals:
		case <-time.After(2 * time.Second):
			t.Fatal("tray host did not receive NewIcon")
		}
		var value dbus.Variant
		if err := observer.Object(connection.Names()[0], "/StatusNotifierItem").Call("org.freedesktop.DBus.Properties.Get", 0, itemInterface, "IconPixmap").Store(&value); err != nil {
			t.Fatal(err)
		}
		var got []iconPixmap
		if err := value.Store(&got); err != nil {
			t.Fatal(err)
		}
		want := normal
		if preferLight {
			want = light
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatal("tray host read the wrong icon variant")
		}
	}
}

func TestGTKThemeFallback(t *testing.T) {
	for theme, want := range map[string]bool{"Adwaita:dark": true, "Breeze-Dark": true, "Adwaita": false, "": false} {
		t.Setenv("GTK_THEME", theme)
		if got := themeFallback(); got != want {
			t.Fatalf("GTK_THEME=%s: %v", theme, got)
		}
	}
}

func TestBothTrayVariantsHaveValidContrastingPixmaps(t *testing.T) {
	normal, err := applicationIcon(false)
	if err != nil {
		t.Fatal(err)
	}
	light, err := applicationIcon(true)
	if err != nil {
		t.Fatal(err)
	}
	for _, icons := range [][]iconPixmap{normal, light} {
		if len(icons) != 1 || icons[0].Width != 64 || icons[0].Height != 64 || len(icons[0].Pixels) != 64*64*4 {
			t.Fatal("invalid tray pixmap")
		}
		if dbus.SignatureOf(icons).String() != "a(iiay)" {
			t.Fatal("invalid D-Bus pixmap signature")
		}
		if icons[0].Pixels[0] != 0 {
			t.Fatal("icon corners must remain transparent")
		}
	}
	// A point on the tile, away from the dark lettering and green route.
	offset := (32*64 + 8) * 4
	for channel := 1; channel <= 3; channel++ {
		if light[0].Pixels[offset+channel] < 200 || normal[0].Pixels[offset+channel] > 80 {
			t.Fatal("light variant must have a bright, contrasting tile")
		}
	}
}
