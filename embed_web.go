//go:build embed_web

package main

import "embed"

//go:embed all:web/dist
var webDistFS embed.FS
