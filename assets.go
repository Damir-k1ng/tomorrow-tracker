// Package assets embeds the built Telegram Mini App SPA so the whole product
// ships as a single Railway deployable: one Go binary serving both the API and
// the frontend.
//
// The embed directive must live at the module root because //go:embed cannot
// reference paths outside its own directory tree, and the SPA is built into
// frontend/dist (Vite's output). frontend/dist always contains at least the
// committed placeholder index.html, so `go build ./...` works even before the
// frontend is built; a real build overwrites it.
package assets

import "embed"

// SPA holds the built Mini App: frontend/dist and everything under it.
//
//go:embed all:frontend/dist
var SPA embed.FS
