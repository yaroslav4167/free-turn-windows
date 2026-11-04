package main

import (
	"testing"

	"github.com/samosvalishe/free-turn-proxy/desktop/integration"
	"github.com/samosvalishe/free-turn-proxy/desktop/updates"
)

func TestUpdateStateKeepsInstallationFailuresVisible(t *testing.T) {
	for _, message := range []string{"Обновление не установлено: ошибка записи", "Новая версия не запустилась; выполнен откат"} {
		app := &App{updater: updates.New("v2.3.0", "amd64", "example/repo"), updateResult: integration.UpdateResult{Message: message}}
		if got := app.UpdateState().Error; got != message {
			t.Fatalf("installation failure hidden: %q", got)
		}
	}
	app := &App{updater: updates.New("v2.3.0", "amd64", "example/repo"), updateResult: integration.UpdateResult{Message: "Обновление установлено. Предыдущие бинарники сохранены для отката."}}
	if got := app.UpdateState().Error; got != "" {
		t.Fatalf("success treated as error: %q", got)
	}
}
