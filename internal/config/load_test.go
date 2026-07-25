package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStrictParsingRejectsUnknownField(t *testing.T) {
	root := t.TempDir()
	if err := Init(root, false); err != nil {
		t.Fatal(err)
	}
	if err := InitEnvironment(root, "lab", false); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "environments", "lab", "environment.yaml")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content = append(content, []byte("unknown: true\n")...)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = Load(root, "lab")
	if err == nil || !strings.Contains(err.Error(), "field unknown not found") {
		t.Fatalf("expected unknown field error, got %v", err)
	}
}

func TestMissingEnvironment(t *testing.T) {
	root := t.TempDir()
	if err := Init(root, false); err != nil {
		t.Fatal(err)
	}
	_, err := Load(root, "missing")
	if err == nil || !strings.Contains(err.Error(), `environment "missing" does not exist`) {
		t.Fatalf("unexpected error: %v", err)
	}
}
