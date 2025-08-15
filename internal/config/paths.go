package config

import (
	"os"
	"path/filepath"
)

// getProjectRootWith allows dependency injection for testability
func getProjectRootWith(getwd func() (string, error), stat func(string) (os.FileInfo, error)) string {
	wd, err := getwd()
	if err != nil {
		return ""
	}
	dir := wd
	for {
		if _, err := stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return wd // fallback
}

// getProjectRoot uses the real os functions
func getProjectRoot() string {
	return getProjectRootWith(os.Getwd, os.Stat)
}

var LogFilePath = filepath.Join(getProjectRoot(), "logs", "log.txt")
