// Package frontend exposes the same JSON language catalogs to native UI code.
package frontend

import "embed"

//go:embed src/locales/*.json
var Locales embed.FS
