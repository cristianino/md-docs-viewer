// Package assets embeds the static files the binary needs to run
// without network access: Mermaid and the default logo.
package assets

import _ "embed"

//go:embed mermaid.min.js
var MermaidJS []byte

//go:embed logo.svg
var DefaultLogo []byte

//go:embed AGENTS.md
var AgentsGuide []byte
