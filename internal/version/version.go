package version

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Build-time variables (set via ldflags)
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

// GetVersion returns the version, preferring build-time version over version file
func GetVersion() string {
	if Version != "dev" {
		return Version
	}

	// Try to read from .version file in project root
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "dev"
	}

	// Go up to project root from internal/version/version.go
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(filename)))
	versionFile := filepath.Join(projectRoot, ".version")

	if data, err := os.ReadFile(versionFile); err == nil {
		return strings.TrimSpace(string(data))
	}

	return "dev"
}

// GetFullVersion returns version with commit and date info
func GetFullVersion() string {
	version := GetVersion()
	if Commit != "unknown" {
		version += "+" + Commit
	}
	return version
}
