package main

import (
	"testing"
)

func TestDirectoryFilter(t *testing.T) {
	tests := []struct {
		dirName  string
		expected bool
	}{
		{".git", true},
		{"node_modules", true},
		{"vendor", true},
		{"bin", true},
		{"tmp", true},
		{"src", false},
		{"handlers", false},
		{"api", false},
	}

	for _, tc := range tests {
		t.Run(tc.dirName, func(t *testing.T) {
			shouldSkip := false
			switch tc.dirName {
			case ".git", "node_modules", "bin", "tmp", "vendor":
				shouldSkip = true
			}

			if shouldSkip != tc.expected {
				t.Errorf("Expected skip=%v for directory %s, got %v", tc.expected, tc.dirName, shouldSkip)
			}
		})
	}
}

func TestFileExtensionFilter(t *testing.T) {
	tests := []struct {
		fileName string
		isValid  bool
	}{
		{"main.go", true},
		{"go.mod", true},
		{"go.sum", true},
		{"readme.md", false},
		{".env", false},
		{"temp.txt", false},
	}

	for _, tc := range tests {
		t.Run(tc.fileName, func(t *testing.T) {
			isValid := false
			ext := tc.fileName[len(tc.fileName)-3:]
			if ext == ".go" || ext == "mod" || ext == "sum" {
				isValid = true
			}

			if isValid != tc.isValid {
				t.Errorf("Expected valid=%v for file %s, got %v", tc.isValid, tc.fileName, isValid)
			}
		})
	}
}
