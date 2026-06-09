package common

import "testing"

func TestEmbedFileSystemExistsWithPrefix(t *testing.T) {
	fs := &embedFileSystem{FileSystem: nil}

	tests := []struct {
		name       string
		prefix     string
		path       string
		wantExists bool
		wantOpen   string
	}{
		{
			name:       "root static path",
			path:       "/assets/index.js",
			wantExists: true,
			wantOpen:   "/assets/index.js",
		},
		{
			name:       "prefixed static path",
			prefix:     "/default",
			path:       "/default/assets/index.js",
			wantExists: true,
			wantOpen:   "/assets/index.js",
		},
		{
			name:       "prefix mismatch",
			prefix:     "/default",
			path:       "/assets/index.js",
			wantExists: false,
		},
		{
			name:       "prefix partial mismatch",
			prefix:     "/default",
			path:       "/default-assets/index.js",
			wantExists: false,
		},
		{
			name:       "directory root stays disabled",
			prefix:     "/default",
			path:       "/default",
			wantExists: false,
			wantOpen:   "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, gotExists := fs.normalizeStaticPathForExists(tt.prefix, tt.path)
			if gotExists != tt.wantExists {
				t.Fatalf("exists mismatch: got %v want %v", gotExists, tt.wantExists)
			}
			if gotPath != tt.wantOpen {
				t.Fatalf("open path mismatch: got %q want %q", gotPath, tt.wantOpen)
			}
		})
	}
}
