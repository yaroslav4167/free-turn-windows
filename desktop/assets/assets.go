// Package assets contains FreeTurn's own application branding, independent of
// the installation directory and the user's icon theme.
package assets

import _ "embed"

//go:embed icon.svg
var IconSVG []byte

//go:embed icon.png
var IconPNG []byte

//go:embed icon-64.png
var TrayIconPNG []byte

//go:embed icon-light-64.png
var TrayLightIconPNG []byte
