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
