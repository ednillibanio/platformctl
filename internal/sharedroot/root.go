package sharedroot

import (
	"fmt"
	"os"
	"path/filepath"
)

// EnvVar overrides the default shared root location.
const EnvVar = "ASSEMBLEIARR_CONFIG_DIR"

// ResolveRoot returns the shared root: flagValue when not empty, else
// ASSEMBLEIARR_CONFIG_DIR, else ~/.config/assembleiarr. The result is absolute
// and clean; a relative override is a usage error.
func ResolveRoot(flagValue string) (string, error) {
	return resolveRoot(flagValue, os.Getenv(EnvVar), os.UserHomeDir)
}

func resolveRoot(flagValue, envValue string, home func() (string, error)) (string, error) {
	switch {
	case flagValue != "":
		return absoluteRoot("--config-dir", flagValue)
	case envValue != "":
		return absoluteRoot(EnvVar, envValue)
	}
	dir, err := home()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	if dir == "" || !filepath.IsAbs(dir) {
		return "", fmt.Errorf("resolve home directory: %q is not an absolute path", dir)
	}
	return filepath.Join(dir, ".config", "assembleiarr"), nil
}

func absoluteRoot(source, value string) (string, error) {
	if !filepath.IsAbs(value) {
		return "", Usagef("%s must be an absolute path: %q", source, value)
	}
	return filepath.Clean(value), nil
}
