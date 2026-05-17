package web

import (
	"testing"
	"testing/fstest"
)

// placeholderDist mimics the committed placeholder: only index.html, no marker.
func placeholderDist() fstest.MapFS {
	return fstest.MapFS{
		"frontend/dist/index.html": {Data: []byte("<!doctype html><title>placeholder</title>")},
	}
}

// realDist mimics a genuine Vite build: index.html plus the build marker.
func realDist() fstest.MapFS {
	d := placeholderDist()
	d["frontend/dist/"+BuildMetaName] = &fstest.MapFile{Data: []byte(`{"built":true}`)}
	return d
}

func TestIsRealBuild(t *testing.T) {
	if IsRealBuild(placeholderDist()) {
		t.Error("placeholder dist must not be detected as a real build")
	}
	if !IsRealBuild(realDist()) {
		t.Error("dist with build marker must be detected as a real build")
	}
}

func TestEnsureRealBuild(t *testing.T) {
	tests := []struct {
		name         string
		files        fstest.MapFS
		isProduction bool
		wantErr      bool
	}{
		{
			name:         "production with missing marker fails fast",
			files:        placeholderDist(),
			isProduction: true,
			wantErr:      true,
		},
		{
			name:         "production with marker succeeds",
			files:        realDist(),
			isProduction: true,
			wantErr:      false,
		},
		{
			name:         "development with missing marker is allowed",
			files:        placeholderDist(),
			isProduction: false,
			wantErr:      false,
		},
		{
			name:         "development with marker is allowed",
			files:        realDist(),
			isProduction: false,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := EnsureRealBuild(tt.files, tt.isProduction)
			if (err != nil) != tt.wantErr {
				t.Fatalf("EnsureRealBuild() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
