package web

import (
	"errors"
	"io/fs"
)

// BuildMetaName is the marker file a genuine Vite build writes into
// frontend/dist. The committed placeholder dist deliberately omits it, so its
// presence inside the embedded filesystem proves the SPA is a real production
// build rather than the placeholder that exists only so `go build` works.
const BuildMetaName = ".build_meta.json"

// IsRealBuild reports whether the embedded frontend filesystem contains the
// Vite build marker. files must contain the frontend/dist subtree (see package
// assets). It performs no disk access — only embed.FS lookups.
func IsRealBuild(files fs.FS) bool {
	f, err := files.Open("frontend/dist/" + BuildMetaName)
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}

// EnsureRealBuild fails fast when a production deployment embeds the committed
// placeholder SPA instead of a genuine Vite build. Outside production it is a
// no-op, so local development and tests keep working with the placeholder.
func EnsureRealBuild(files fs.FS, isProduction bool) error {
	if !isProduction {
		return nil
	}
	if !IsRealBuild(files) {
		return errors.New("production frontend is the committed placeholder: " +
			"real Vite build marker (" + BuildMetaName + ") is missing — " +
			"run the frontend build before deploying")
	}
	return nil
}
