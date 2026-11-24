// Package web embeds the public dashboard page.
package web

import _ "embed"

// Index is the dashboard HTML served at GET /.
//
//go:embed index.html
var Index []byte
