package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrWouldOverwrite = errors.New("refusing to overwrite an existing managed file without --force")

const defaultsTemplate = `apiVersion: platformctl.io/v1alpha1
kind: Defaults
metadata:
  name: defaults
spec:
  values: {}
  credentials: {}
`

const environmentTemplate = `apiVersion: platformctl.io/v1alpha1
kind: Environment
metadata:
  name: %s
spec:
  values: {}
  credentials: {}
`

func Init(root string, force bool) error {
	if err := ensurePrivateDir(root); err != nil {
		return fmt.Errorf("create platform home: %w", err)
	}
	if err := ensurePrivateDir(filepath.Join(root, "environments")); err != nil {
		return fmt.Errorf("create environments directory: %w", err)
	}
	return writeManaged(filepath.Join(root, "defaults.yaml"), []byte(defaultsTemplate), force)
}

func InitEnvironment(root, env string, force bool) error {
	if !ValidIdentifier(env) {
		return fmt.Errorf("invalid environment name %q", env)
	}
	if _, err := os.Stat(filepath.Join(root, "defaults.yaml")); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("platform home is not initialized; run platformctl init first")
		}
		return fmt.Errorf("inspect platform home: %w", err)
	}
	envRoot := filepath.Join(root, "environments", env)
	for _, dir := range []string{"", "profiles", "services", "hosts"} {
		if err := ensurePrivateDir(filepath.Join(envRoot, dir)); err != nil {
			return fmt.Errorf("create environment structure: %w", err)
		}
	}
	content := []byte(fmt.Sprintf(environmentTemplate, env))
	return writeManaged(filepath.Join(envRoot, "environment.yaml"), content, force)
}

func writeManaged(path string, content []byte, force bool) error {
	existing, err := os.ReadFile(path)
	switch {
	case err == nil:
		if string(existing) == string(content) {
			return os.Chmod(path, 0o600)
		}
		if !force {
			return fmt.Errorf("%w: %s", ErrWouldOverwrite, path)
		}
	case !os.IsNotExist(err):
		return fmt.Errorf("read managed file %s: %w", path, err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		return fmt.Errorf("write managed file %s: %w", path, err)
	}
	return os.Chmod(path, 0o600)
}

func ensurePrivateDir(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	return os.Chmod(path, 0o700)
}
