package backend

import "testing"

func TestCaptchaURLSurvivesConnectedPhaseUntilResolvedOrDisconnected(t *testing.T) {
	e := &Engine{}
	e.OnCaptcha("http://localhost:8765")
	e.OnState("connected", 1, 2, "")
	if e.captchaURL != "http://localhost:8765" {
		t.Fatal("working streams hid the pending verification")
	}
	e.OnCaptcha("")
	if e.captchaURL != "" {
		t.Fatal("completed verification was not cleared")
	}
	e.OnCaptcha("http://localhost:8765")
	e.OnState("idle", 0, 2, "")
	if e.captchaURL != "" || e.Status().CaptchaURL != "" {
		t.Fatal("disconnect retained an expired challenge")
	}
}

func TestCaptchaNotificationOpensOnlyCurrentChallenge(t *testing.T) {
	e := &Engine{captchaURL: "http://localhost:8765"}
	calls := 0
	open := func() { calls++ }
	e.openCurrentCaptcha("http://localhost:8765", open)
	if calls != 1 {
		t.Fatal("current captcha action ignored")
	}
	e.captchaURL = ""
	e.openCurrentCaptcha("http://localhost:8765", open)
	e.captchaURL = "http://localhost:8766"
	e.openCurrentCaptcha("http://localhost:8765", open)
	e.OnState("error", 0, 0, "disconnected")
	e.openCurrentCaptcha("http://localhost:8766", open)
	if calls != 1 {
		t.Fatal("stale notification opened an expired captcha")
	}
}
