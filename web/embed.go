// Package web holds the HTML templates and static assets, embedded in the binary.
package web

import "embed"

//go:embed templates static
var FS embed.FS
