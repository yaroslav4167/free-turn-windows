// Package locale shares JSON translation catalogs with the web interface.
package locale

import (
	"encoding/json"
	"strings"
	"sync/atomic"

	"github.com/samosvalishe/free-turn-proxy/desktop/frontend"
)

type Catalog struct {
	Code     string            `json:"code"`
	Name     string            `json:"name"`
	Source   bool              `json:"source"`
	Messages map[string]string `json:"messages"`
}

var catalogs = loadCatalogs()
var selected atomic.Value

func init() { selected.Store("en") }

func loadCatalogs() map[string]Catalog {
	result := map[string]Catalog{}
	files, err := frontend.Locales.ReadDir("src/locales")
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		data, err := frontend.Locales.ReadFile("src/locales/" + file.Name())
		if err != nil {
			panic(err)
		}
		var catalog Catalog
		if err := json.Unmarshal(data, &catalog); err != nil {
			panic(err)
		}
		if catalog.Code == "" || catalog.Name == "" || catalog.Messages == nil {
			panic("invalid language catalog: " + file.Name())
		}
		result[strings.ToLower(catalog.Code)] = catalog
	}
	return result
}

func Supported(language string) bool { _, ok := catalogs[language]; return ok }
func MatchSystemLanguage(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.SplitN(strings.SplitN(value, ".", 2)[0], "@", 2)[0]
	value = strings.ReplaceAll(value, "_", "-")
	if Supported(value) {
		return value
	}
	base := strings.SplitN(value, "-", 2)[0]
	if Supported(base) {
		return base
	}
	return "en"
}
func Set(language string) {
	if !Supported(language) {
		language = "en"
	}
	selected.Store(language)
}
func Text(ru, en string) string {
	catalog := catalogs[selected.Load().(string)]
	if text, ok := catalog.Messages[ru]; ok {
		return text
	}
	if catalog.Source {
		return ru
	}
	if text, ok := catalogs["en"].Messages[ru]; ok {
		return text
	}
	return en
}
