package platformhome

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const EnvName = "PLATFORMCTL_HOME"

// Resolve returns the platformctl configuration root for the current process.
func Resolve() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config directory: %w", err)
	}
	return ResolveFor(runtime.GOOS, os.Getenv(EnvName), base)
}

// ResolveFor exposes the platform-specific policy as a pure function for tests.
func ResolveFor(goos, override, userConfigDir string) (string, error) {
	if override != "" {
		if !isAbsolute(goos, override) {
			return "", fmt.Errorf("%s must be an absolute path", EnvName)
		}
		return clean(goos, override), nil
	}
	if userConfigDir == "" {
		return "", fmt.Errorf("user config directory is empty")
	}
	if goos == "windows" {
		return strings.TrimRight(userConfigDir, `\/`) + `\platformctl`, nil
	}
	return filepath.Join(userConfigDir, "platformctl"), nil
}

func isAbsolute(goos, value string) bool {
	if goos != "windows" {
		return filepath.IsAbs(value)
	}
	if strings.HasPrefix(value, `\\`) {
		return true
	}
	return len(value) >= 3 &&
		((value[0] >= 'A' && value[0] <= 'Z') || (value[0] >= 'a' && value[0] <= 'z')) &&
		value[1] == ':' && (value[2] == '\\' || value[2] == '/')
}

func clean(goos, value string) string {
	if goos == "windows" {
		return strings.TrimRight(value, `\/`)
	}
	return filepath.Clean(value)
}
