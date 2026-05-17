// Package web serves the embedded Telegram Mini App SPA.
//
// It is deliberately thin: it takes the embedded filesystem, serves real build
// assets directly, and falls back to index.html for every other path so the
// client-side router (React Router) handles deep links and refreshes. API
// isolation is the caller's job — the API routes are registered on more
// specific mux patterns, so they always win over this handler's "/" mount.
package web

import (
	"io/fs"
	"net/http"
	"strings"
)

// Cache-Control policies for the three classes of response this handler
// produces. Splitting them is the fix for R1: a stale hashed asset is
// impossible (the URL changes with the content), but a stable-named root file
// or the HTML shell must revalidate or a deploy can be missed.
const (
	// cacheImmutable suits content-hashed bundles under assets/: the filename
	// itself changes whenever the content does, so they can be cached forever.
	cacheImmutable = "public, max-age=31536000, immutable"
	// cacheRevalidate suits stable-named root files (favicon, PWA icons,
	// manifest): the URL never changes, so the client must revalidate to pick
	// up a new deploy. A short max-age still cuts most repeat requests.
	cacheRevalidate = "public, max-age=3600, must-revalidate"
	// cacheNone suits the HTML shell: always revalidate so a fresh deploy is
	// picked up on the next navigation.
	cacheNone = "no-cache"
)

// cacheControlFor returns the Cache-Control value for a real file served from
// the embedded dist. path is the request path with the leading slash trimmed.
func cacheControlFor(path string) string {
	switch {
	case strings.HasPrefix(path, "assets/"):
		return cacheImmutable
	case path == "index.html":
		// The shell, whether reached directly or via fallback, is never cached.
		return cacheNone
	default:
		return cacheRevalidate
	}
}

// Handler builds an http.Handler that serves the SPA from the given embedded
// filesystem. files must contain the frontend/dist subtree (see package
// assets). An error is returned only when the build output is missing.
func Handler(files fs.FS) (http.Handler, error) {
	dist, err := fs.Sub(files, "frontend/dist")
	if err != nil {
		return nil, err
	}
	index, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		return nil, err
	}
	fileServer := http.FileServer(http.FS(dist))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Serve a real file when one exists (hashed assets, icons, manifest).
		if p := strings.TrimPrefix(r.URL.Path, "/"); p != "" && exists(dist, p) {
			w.Header().Set("Cache-Control", cacheControlFor(p))
			fileServer.ServeHTTP(w, r)
			return
		}
		// Otherwise hand the SPA shell to the client router. index.html itself
		// is never cached so a new deploy is picked up immediately.
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", cacheNone)
		_, _ = w.Write(index)
	}), nil
}

// exists reports whether name is a regular file in fsys.
func exists(fsys fs.FS, name string) bool {
	f, err := fsys.Open(name)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	return err == nil && !info.IsDir()
}
