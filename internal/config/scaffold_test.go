package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestScaffoldIdempotencyAndForce(t *testing.T) {
	root := t.TempDir()
	if err := Init(root, false); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(root, "defaults.yaml"))
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Fatalf("defaults mode is %04o, want 0600", got)
		}
	}
	if err := Init(root, false); err != nil {
		t.Fatalf("second init must be idempotent: %v", err)
	}
	if err := InitEnvironment(root, "lab", false); err != nil {
		t.Fatal(err)
	}
	extra := filepath.Join(root, "environments", "lab", "operator-note.yaml")
	if err := os.WriteFile(extra, []byte("preserve me\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := InitEnvironment(root, "lab", false); err != nil {
		t.Fatalf("second environment init must be idempotent: %v", err)
	}
	path := filepath.Join(root, "environments", "lab", "environment.yaml")
	if err := os.WriteFile(path, []byte("operator content\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := InitEnvironment(root, "lab", false); !errors.Is(err, ErrWouldOverwrite) {
		t.Fatalf("expected ErrWouldOverwrite, got %v", err)
	}
	if err := InitEnvironment(root, "lab", true); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) == "operator content\n" {
		t.Fatal("--force did not replace the managed file")
	}
	if _, err := os.Stat(extra); err != nil {
		t.Fatalf("--force removed an unmanaged file: %v", err)
	}
}

func TestInitEnvironmentRequiresRoot(t *testing.T) {
	if err := InitEnvironment(t.TempDir(), "lab", false); err == nil {
		t.Fatal("expected uninitialized home error")
	}
}

func TestInitRejectsSymbolicLinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symbolic-link creation requires extra privileges on Windows")
	}

	t.Run("platform home", func(t *testing.T) {
		target := t.TempDir()
		marker := filepath.Join(target, "must-stay.txt")
		if err := os.WriteFile(marker, []byte("preserve me\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		root := filepath.Join(t.TempDir(), "platformctl")
		if err := os.Symlink(target, root); err != nil {
			t.Fatal(err)
		}

		err := Init(root, true)
		if !errors.Is(err, ErrUnsafeSymlink) {
			t.Fatalf("expected ErrUnsafeSymlink, got %v", err)
		}
		assertUnchangedFile(t, marker, "preserve me\n", 0o600)
	})

	t.Run("managed file", func(t *testing.T) {
		root := t.TempDir()
		target := filepath.Join(t.TempDir(), "outside.yaml")
		if err := os.WriteFile(target, []byte("preserve me\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, filepath.Join(root, "defaults.yaml")); err != nil {
			t.Fatal(err)
		}

		err := Init(root, true)
		if !errors.Is(err, ErrUnsafeSymlink) {
			t.Fatalf("expected ErrUnsafeSymlink, got %v", err)
		}
		assertUnchangedFile(t, target, "preserve me\n", 0o600)
	})
}

func TestInitEnvironmentRejectsSymbolicLinkedManagedFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symbolic-link creation requires extra privileges on Windows")
	}
	root := t.TempDir()
	if err := Init(root, false); err != nil {
		t.Fatal(err)
	}
	if err := InitEnvironment(root, "lab", false); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "environments", "lab", "environment.yaml")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "outside.yaml")
	if err := os.WriteFile(target, []byte("preserve me\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}

	err := InitEnvironment(root, "lab", true)
	if !errors.Is(err, ErrUnsafeSymlink) {
		t.Fatalf("expected ErrUnsafeSymlink, got %v", err)
	}
	assertUnchangedFile(t, target, "preserve me\n", 0o600)
}

func TestInitEnvironmentRejectsSymbolicLinkedManagedDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symbolic-link creation requires extra privileges on Windows")
	}
	root := t.TempDir()
	if err := Init(root, false); err != nil {
		t.Fatal(err)
	}
	managedDir := filepath.Join(root, "environments")
	if err := os.Remove(managedDir); err != nil {
		t.Fatal(err)
	}
	target := t.TempDir()
	marker := filepath.Join(target, "must-stay.txt")
	if err := os.WriteFile(marker, []byte("preserve me\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, managedDir); err != nil {
		t.Fatal(err)
	}

	err := InitEnvironment(root, "lab", true)
	if !errors.Is(err, ErrUnsafeSymlink) {
		t.Fatalf("expected ErrUnsafeSymlink, got %v", err)
	}
	assertUnchangedFile(t, marker, "preserve me\n", 0o600)
}

func assertUnchangedFile(t *testing.T, path, want string, mode os.FileMode) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != want {
		t.Fatalf("%s content = %q, want %q", path, content, want)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != mode {
		t.Fatalf("%s mode = %04o, want %04o", path, got, mode)
	}
}
