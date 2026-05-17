package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

// fakeDist mimics the embedded layout: everything under frontend/dist.
func fakeDist() fstest.MapFS {
	return fstest.MapFS{
		"frontend/dist/index.html":    {Data: []byte("<!doctype html><title>SPA</title>")},
		"frontend/dist/assets/app.js": {Data: []byte("console.log('app')")},
		"frontend/dist/favicon.svg":   {Data: []byte("<svg/>")},
	}
}

func newHandler(t *testing.T) http.Handler {
	t.Helper()
	h, err := Handler(fakeDist())
	if err != nil {
		t.Fatalf("Handler: %v", err)
	}
	return h
}

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestHandler_ServesIndexAtRoot(t *testing.T) {
	rec := get(t, newHandler(t), "/")
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d", rec.Code)
	}
	if body, _ := io.ReadAll(rec.Body); string(body) != "<!doctype html><title>SPA</title>" {
		t.Errorf("root did not serve index.html: %q", body)
	}
}

func TestHandler_FallsBackToIndexForClientRoutes(t *testing.T) {
	// A deep link with no matching file must serve the SPA shell so the
	// client-side router can take over.
	rec := get(t, newHandler(t), "/admin/users")
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d", rec.Code)
	}
	if body, _ := io.ReadAll(rec.Body); string(body) != "<!doctype html><title>SPA</title>" {
		t.Errorf("client route did not fall back to index.html: %q", body)
	}
}

func TestHandler_ServesRealAssets(t *testing.T) {
	rec := get(t, newHandler(t), "/assets/app.js")
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d", rec.Code)
	}
	if body, _ := io.ReadAll(rec.Body); string(body) != "console.log('app')" {
		t.Errorf("asset body mismatch: %q", body)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "public, max-age=31536000, immutable" {
		t.Errorf("hashed asset should be immutably cached, got %q", cc)
	}
}

func TestHandler_CacheControl(t *testing.T) {
	// R1: each response class must carry the right Cache-Control. Hashed
	// assets are immutable; stable-named root files and the HTML shell must
	// revalidate so a new deploy is never missed.
	tests := []struct {
		name     string
		path     string
		want     string
		wantCode int
	}{
		{
			name:     "hashed asset is immutable",
			path:     "/assets/app.js",
			want:     "public, max-age=31536000, immutable",
			wantCode: http.StatusOK,
		},
		{
			name:     "stable-named root file revalidates",
			path:     "/favicon.svg",
			want:     "public, max-age=3600, must-revalidate",
			wantCode: http.StatusOK,
		},
		{
			// http.FileServer canonicalizes /index.html to a redirect; the
			// redirect itself must still not be cached aggressively.
			name:     "index.html requested directly is not cached",
			path:     "/index.html",
			want:     "no-cache",
			wantCode: http.StatusMovedPermanently,
		},
		{
			name:     "client route fallback is not cached",
			path:     "/admin/users",
			want:     "no-cache",
			wantCode: http.StatusOK,
		},
		{
			name:     "root is not cached",
			path:     "/",
			want:     "no-cache",
			wantCode: http.StatusOK,
		},
	}

	h := newHandler(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := get(t, h, tt.path)
			if rec.Code != tt.wantCode {
				t.Fatalf("status: got %d, want %d", rec.Code, tt.wantCode)
			}
			if cc := rec.Header().Get("Cache-Control"); cc != tt.want {
				t.Errorf("Cache-Control for %s: got %q, want %q", tt.path, cc, tt.want)
			}
		})
	}
}

func TestHandler_MissingError(t *testing.T) {
	// An FS without the build output must surface an error, not panic.
	if _, err := Handler(fstest.MapFS{}); err == nil {
		t.Error("expected an error when frontend/dist is absent")
	}
}
