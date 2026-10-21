// Package ui holds the web UI, built into the binary so a release runs from one file.
package ui

import "embed"

// Files is served at /ui/ unless -ui-dir names a directory to serve instead.
//
//go:embed *.html *.js *.css
var Files embed.FS
