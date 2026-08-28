package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrWouldOverwrite = errors.New("refusing to overwrite an existing managed file without --force")
var ErrUnsafeSymlink = errors.New("refusing to follow a symbolic link in the platform home")

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
	if err := rejectSymlink(root); err != nil {
		return err
	}
	defaultsPath := filepath.Join(root, "defaults.yaml")
	if err := rejectSymlink(defaultsPath); err != nil {
		return err
	}
	if _, err := os.Stat(defaultsPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("platform home is not initialized; run platformctl init first")
		}
		return fmt.Errorf("inspect platform home: %w", err)
	}
	if err := ensurePrivateDir(filepath.Join(root, "environments")); err != nil {
		return fmt.Errorf("create environments directory: %w", err)
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
	if err := rejectSymlink(path); err != nil {
		return err
	}
	existing, err := os.ReadFile(path)
	switch {
	case err == nil:
		if string(existing) == string(content) {
			return setPrivateFileMode(path)
		}
		if !force {
			return fmt.Errorf("%w: %s", ErrWouldOverwrite, path)
		}
	case !os.IsNotExist(err):
		return fmt.Errorf("read managed file %s: %w", path, err)
	}
	return replaceManaged(path, content)
}

func ensurePrivateDir(path string) error {
	if err := rejectSymlink(path); err != nil {
		return err
	}
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	if err := rejectSymlink(path); err != nil {
		return err
	}
	return os.Chmod(path, 0o700)
}

func replaceManaged(path string, content []byte) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("replace managed file %s: %w", path, err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create managed file %s: %w", path, err)
	}
	if _, err := file.Write(content); err != nil {
		_ = file.Close()
		return fmt.Errorf("write managed file %s: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close managed file %s: %w", path, err)
	}
	return setPrivateFileMode(path)
}

func setPrivateFileMode(path string) error {
	if err := rejectSymlink(path); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

func rejectSymlink(path string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect platform path %s: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%w: %s", ErrUnsafeSymlink, path)
	}
	return nil
}
