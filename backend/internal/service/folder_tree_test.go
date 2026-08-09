package service

import "testing"

func TestNormalizeFolderPath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"root slash", "/", ""},
		{"leading slash", "/a/b", "a/b"},
		{"trailing slash", "a/b/", "a/b"},
		{"both slashes", "/a/b/", "a/b"},
		{"double slashes", "a//b", "a/b"},
		{"multiple slashes", "///a///b///", "a/b"},
		{"no change", "a/b/c", "a/b/c"},
		{"single level", "a", "a"},
		{"whitespace in path", " a / b ", "a/b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeFolderPath(tt.input)
			if got != tt.want {
				t.Errorf("normalizeFolderPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestPathDepth(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty", "", 0},
		{"root", "/", 2},
		{"one level", "a", 1},
		{"two levels", "a/b", 2},
		{"three levels", "a/b/c", 3},
		{"leading slash", "/a/b", 3},
		{"trailing slash", "a/b/", 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pathDepth(tt.input)
			if got != tt.want {
				t.Errorf("pathDepth(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestParentFolderPath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"one level", "a", ""},
		{"two levels", "a/b", "a"},
		{"three levels", "a/b/c", "a/b"},
		{"leading slash", "/a/b", "/a"},
		{"trailing slash", "a/b/", "a/b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parentFolderPath(tt.input)
			if got != tt.want {
				t.Errorf("parentFolderPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
