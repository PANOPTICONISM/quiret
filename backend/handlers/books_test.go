package handlers

import (
	"path/filepath"
	"testing"
)

func TestIsPathAllowed(t *testing.T) {
	dataDir := t.TempDir()
	booksDir := t.TempDir()
	audiobooksDir := t.TempDir()
	outsideDir := t.TempDir()

	// Save and restore package globals so the test doesn't leak state.
	origData, origPaths := DataPath, BookPaths
	t.Cleanup(func() { DataPath, BookPaths = origData, origPaths })

	DataPath = dataDir
	BookPaths = []string{booksDir, audiobooksDir}

	cases := []struct {
		name string
		path string
		want bool
	}{
		{"inside data", filepath.Join(dataDir, "books", "x", "cover.jpg"), true},
		{"inside first book path", filepath.Join(booksDir, "novel.epub"), true},
		{"inside audiobooks path", filepath.Join(audiobooksDir, "book.m4b"), true},
		{"outside all roots", filepath.Join(outsideDir, "secret.epub"), false},
		{"absolute traversal", "/etc/passwd", false},
		{"traversal escape from data", filepath.Join(dataDir, "..", "escape.epub"), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isPathAllowed(tc.path); got != tc.want {
				t.Errorf("isPathAllowed(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}
