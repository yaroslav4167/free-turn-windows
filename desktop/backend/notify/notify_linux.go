//go:build linux

package notify

import (
	"context"
	"errors"
	"fmt"
	"html"
	"net/url"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/samosvalishe/free-turn-proxy/desktop/locale"
)

const throttleWindow = 5 * time.Second

var captchaMu sync.Mutex
var lastCaptchaTime time.Time
var lastCaptchaURL string

const notificationService = "org.freedesktop.Notifications"
const notificationPath = dbus.ObjectPath("/org/freedesktop/Notifications")

// ShowCaptchaNotification offers a default click action and an explicit button.
// Merely showing the notification NEVER invokes onOpen or opens a browser.
func ShowCaptchaNotification(captchaURL string, onOpen func()) error {
	link, err := validBrowserURL(captchaURL)
	if err != nil {
		return err
	}
	captchaMu.Lock()
	if lastCaptchaURL == link && time.Since(lastCaptchaTime) < throttleWindow {
		captchaMu.Unlock()
		return nil
	}
	lastCaptchaTime = time.Now()
	lastCaptchaURL = link
	captchaMu.Unlock()

	if onOpen == nil {
		onOpen = func() {
			cmd := exec.Command("xdg-open", link) //nolint:gosec,noctx // browser lifetime is not a notification timeout
			if cmd.Start() == nil {
				go func() { _ = cmd.Wait() }()
			}
		}
	}
	title := locale.Text("FreeTurn: требуется решение капчи VK", "FreeTurn: VK captcha verification required")
	err = showInteractiveNotification(title,
		locale.Text("Нажмите уведомление или «Открыть в браузере», чтобы пройти проверку.", "Click this notification or ‘Open in browser’ to complete verification."), onOpen)
	if err == nil {
		return nil
	}
	// Not all notification servers implement actions. Keep the existing
	// in-application captcha button as the portable fallback.
	fallback := ShowNotification(title,
		locale.Text("Откройте FreeTurn и нажмите «Открыть проверку в браузере».", "Open FreeTurn and click ‘Open verification in browser’."), link)
	if fallback != nil {
		return errors.Join(err, fallback)
	}
	return nil
}

func validBrowserURL(raw string) (string, error) {
	link := strings.TrimSpace(raw)
	u, err := url.Parse(link)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Hostname() == "" || u.User != nil || strings.ContainsAny(link, "\r\n\x00") {
		return "", errors.New("недопустимая ссылка проверки в браузере")
	}
	return link, nil
}

func showInteractiveNotification(title, body string, onOpen func()) error {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return err
	}
	started := false
	defer func() {
		if !started {
			_ = conn.Close()
		}
	}()
	object := conn.Object(notificationService, notificationPath)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var capabilities []string
	if err := object.CallWithContext(ctx, notificationService+".GetCapabilities", 0).Store(&capabilities); err != nil {
		return err
	}
	supportsActions := false
	for _, capability := range capabilities {
		supportsActions = supportsActions || capability == "actions"
	}
	if !supportsActions {
		return errors.New("сервер уведомлений не поддерживает действия")
	}
	signals := make(chan *dbus.Signal, 8)
	conn.Signal(signals)
	for _, member := range []string{"ActionInvoked", "NotificationClosed"} {
		if err := conn.AddMatchSignal(dbus.WithMatchSender(notificationService), dbus.WithMatchObjectPath(notificationPath), dbus.WithMatchInterface(notificationService), dbus.WithMatchMember(member)); err != nil {
			return err
		}
	}
	var id uint32
	actions := []string{"default", locale.Text("Открыть проверку", "Open verification"), "open", locale.Text("Открыть в браузере", "Open in browser")}
	hints := map[string]dbus.Variant{"desktop-entry": dbus.MakeVariant("freeturn")}
	if err := object.CallWithContext(ctx, notificationService+".Notify", 0,
		"FreeTurn", uint32(0), "freeturn", title, body, actions, hints, int32(-1)).Store(&id); err != nil {
		return err
	}
	if id == 0 {
		return errors.New("сервер не создал уведомление")
	}
	started = true
	go waitNotificationAction(conn, id, signals, onOpen)
	return nil
}

func notificationEvent(signal *dbus.Signal, id uint32) (open, done bool) {
	if signal == nil || len(signal.Body) != 2 {
		return false, false
	}
	notificationID, ok := signal.Body[0].(uint32)
	if !ok || notificationID != id {
		return false, false
	}
	switch signal.Name {
	case notificationService + ".ActionInvoked":
		action, _ := signal.Body[1].(string)
		if action == "default" || action == "open" {
			return true, true
		}
	case notificationService + ".NotificationClosed":
		return false, true
	}
	return false, false
}

func waitNotificationAction(conn *dbus.Conn, id uint32, signals <-chan *dbus.Signal, onOpen func()) {
	defer conn.Close()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = conn.Object(notificationService, notificationPath).CallWithContext(ctx, notificationService+".CloseNotification", 0, id).Err
	}()
	timeout := time.NewTimer(5 * time.Minute)
	defer timeout.Stop()
	dispatchNotificationAction(id, signals, timeout.C, onOpen)
}

func dispatchNotificationAction(id uint32, signals <-chan *dbus.Signal, timeout <-chan time.Time, onOpen func()) {
	for {
		select {
		case signal, ok := <-signals:
			if !ok {
				return
			}
			open, done := notificationEvent(signal, id)
			if open && onOpen != nil {
				onOpen()
			}
			if done {
				return
			}
		case <-timeout:
			return
		}
	}
}

// ShowNotification uses the freedesktop notification service via notify-send.
// Used only as a plain-text fallback when interactive D-Bus actions are absent.
func ShowNotification(title, message, url string) error {
	if u := strings.TrimSpace(url); u != "" {
		message += "\n" + u
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "notify-send", "--app-name=FreeTurn", "--", title, html.EscapeString(message)) //nolint:gosec
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("desktop notification: %w", err)
	}
	return nil
}
