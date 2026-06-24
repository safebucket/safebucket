package helpers

import "testing"

func TestExtensionFromName(t *testing.T) {
	testCases := []struct {
		name     string
		expected string
	}{
		{"file.txt", "txt"},
		{"archive.tar.gz", "gz"},
		{"Makefile", ""},
		{"README", ""},
		{".gitignore", ""},
		{".env", ""},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtensionFromName(tt.name); got != tt.expected {
				t.Errorf("ExtensionFromName(%q) = %q, want %q", tt.name, got, tt.expected)
			}
		})
	}
}
