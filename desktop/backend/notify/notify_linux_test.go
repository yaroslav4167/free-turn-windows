//go:build linux

package notify

import (
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
)

func TestCaptchaBrowserLinks(t *testing.T) {
	for _, value := range []string{"http://localhost:8765", "http://127.0.0.1:8765/", "https://example.com/captcha?a=1&b=2"} {
		if got, err := validBrowserURL(value); err != nil || got != value {
			t.Fatalf("valid URL rejected: %q, %v", value, err)
		}
	}
	for _, value := range []string{"", "javascript:alert(1)", "file:///etc/passwd", "ftp://example.com", "https://user:pass@example.com", "http://", "https://example.com/\nattack"} {
		if _, err := validBrowserURL(value); err == nil {
			t.Fatalf("unsafe URL accepted: %q", value)
		}
	}
}

func TestOnlyClickingNotificationOpensBrowser(t *testing.T) {
	for _, action := range []string{"default", "open"} {
		signals := make(chan *dbus.Signal, 4)
		calls := 0
		signals <- &dbus.Signal{Name: notificationService + ".ActionInvoked", Body: []interface{}{uint32(99), action}}
		signals <- &dbus.Signal{Name: notificationService + ".ActionInvoked", Body: []interface{}{uint32(7), action}}
		signals <- &dbus.Signal{Name: notificationService + ".ActionInvoked", Body: []interface{}{uint32(7), action}}
		dispatchNotificationAction(7, signals, make(chan time.Time), func() { calls++ })
		if calls != 1 {
			t.Fatalf("action %s: browser opened %d times", action, calls)
		}
	}
}

func TestDismissalAndUnknownActionsNeverOpenBrowser(t *testing.T) {
	signals := make(chan *dbus.Signal, 4)
	signals <- nil
	signals <- &dbus.Signal{Name: notificationService + ".ActionInvoked", Body: []interface{}{uint32(7), "unexpected"}}
	signals <- &dbus.Signal{Name: notificationService + ".NotificationClosed", Body: []interface{}{uint32(7), uint32(2)}}
	dispatchNotificationAction(7, signals, make(chan time.Time), func() { t.Fatal("dismissal must not launch a browser") })
	timeout := make(chan time.Time, 1)
	timeout <- time.Now()
	dispatchNotificationAction(7, make(chan *dbus.Signal), timeout, func() { t.Fatal("timeout must not launch a browser") })
	closed := make(chan *dbus.Signal)
	close(closed)
	dispatchNotificationAction(7, closed, make(chan time.Time), func() { t.Fatal("closed connection must not launch a browser") })
}
