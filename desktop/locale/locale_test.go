package locale

import "testing"

func TestSharedJSONCatalogs(t *testing.T) {
	t.Cleanup(func() { Set("en") })
	if !Supported("ru") || !Supported("en") || Supported("not-a-locale") {
		t.Fatal("incorrect language registry")
	}
	Set("ru")
	if text := Text("Настройки", "wrong"); text != "Настройки" {
		t.Fatal(text)
	}
	Set("en")
	if text := Text("Настройки", "wrong"); text != "Settings" {
		t.Fatal(text)
	}
	if text := Text("custom diagnostic", "fallback"); text != "fallback" {
		t.Fatal(text)
	}
}
