// Package templates embeds the default HTML templates. A project can
// override any of them by placing a file with the same name in its
// own templates/ folder.
package templates

import "embed"

//go:embed *.gohtml
var FS embed.FS
