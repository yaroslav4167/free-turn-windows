//go:build linux

package tray

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	portalName          = "org.freedesktop.portal.Desktop"
	settingsInterface   = "org.freedesktop.portal.Settings"
	appearanceNamespace = "org.freedesktop.appearance"
)

// The portal reports 1 for a dark appearance, 2 for light and 0 for no
// preference. Some portal implementations wrap the variant a second time.
func lightIconForScheme(value interface{}, fallback bool) bool {
	for depth := 0; depth < 8; depth++ {
		if variant, ok := value.(dbus.Variant); ok {
			value = variant.Value()
			continue
		}
		scheme, ok := value.(uint32)
		if !ok {
			return fallback
		}
		switch scheme {
		case 1:
			return true
		case 2:
			return false
		default:
			return fallback
		}
	}
	return fallback
}

func themeFallback() bool {
	return strings.Contains(strings.ToLower(os.Getenv("GTK_THEME")), "dark")
}

func preferLightTrayIcon(conn *dbus.Conn) bool {
	fallback := themeFallback()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var value dbus.Variant
	err := conn.Object(portalName, "/org/freedesktop/portal/desktop").CallWithContext(
		ctx, settingsInterface+".Read", 0, appearanceNamespace, "color-scheme",
	).Store(&value)
	if err != nil {
		return fallback
	}
	return lightIconForScheme(value, fallback)
}

func (t *Tray) updateIcon(light bool) {
	if t.lightIcon == light {
		return
	}
	t.lightIcon = light
	index := 0
	if light {
		index = 1
	}
	// StatusNotifierItem hosts refresh IconPixmap on NewIcon. The property is
	// read-only to clients but mutable internally (not annotated as constant).
	t.iconProperties.SetMust(itemInterface, "IconPixmap", t.icons[index])
	_ = t.conn.Emit("/StatusNotifierItem", itemInterface+".NewIcon")
}
