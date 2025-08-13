package config

import (
	"os"
	"testing"
)

func TestGetProjectRoot(t *testing.T) {
	root := getProjectRoot()
	if root == "" {
		t.Error("getProjectRoot returned empty string")
	}
	// Optionally check if go.mod exists in root
	if _, err := os.Stat(root + "/go.mod"); err != nil {
		t.Errorf("go.mod not found in project root: %v", err)
	}
}

func TestGetProjectRootError(t *testing.T) {
	getwd := func() (string, error) { return "", os.ErrNotExist }
	stat := func(name string) (os.FileInfo, error) { return nil, nil }
	root := getProjectRootWith(getwd, stat)
	if root != "" {
		t.Errorf("Expected empty string on Getwd error, got %q", root)
	}
}

func TestGetProjectRootFallback(t *testing.T) {
	wd, _ := os.Getwd()
	getwd := func() (string, error) { return wd, nil }
	stat := func(name string) (os.FileInfo, error) { return nil, os.ErrNotExist }
	root := getProjectRootWith(getwd, stat)
	if root != wd {
		t.Errorf("Expected fallback to wd, got %q", root)
	}
}
