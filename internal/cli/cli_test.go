package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestMVPWorkflow(t *testing.T) {
	home := t.TempDir()
	t.Setenv("PLATFORMCTL_HOME", home)

	run := func(args ...string) (string, error) {
		t.Helper()
		var stdout, stderr bytes.Buffer
		command := NewWithIO(&stdout, &stderr)
		command.SetArgs(args)
		err := command.Execute()
		return stdout.String() + stderr.String(), err
	}

	if _, err := run("init"); err != nil {
		t.Fatal(err)
	}
	if _, err := run("init"); err != nil {
		t.Fatalf("init is not idempotent: %v", err)
	}
	if _, err := run("environment", "init", "lab"); err != nil {
		t.Fatal(err)
	}
	if output, err := run("validate", "--env", "lab"); err != nil {
		t.Fatal(err)
	} else if !strings.Contains(output, `Environment "lab" is valid`) {
		t.Fatalf("unexpected validation output: %s", output)
	}
	output, err := run("render", "--env", "lab", "--format", "json")
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal([]byte(output), &document); err != nil {
		t.Fatalf("render returned invalid JSON: %v", err)
	}
	if document["kind"] != "RenderedEnvironment" {
		t.Fatalf("unexpected render: %s", output)
	}
}

func TestRenderRejectsUnknownFormat(t *testing.T) {
	home := t.TempDir()
	t.Setenv("PLATFORMCTL_HOME", home)
	run := func(args ...string) error {
		command := NewWithIO(&bytes.Buffer{}, &bytes.Buffer{})
		command.SetArgs(args)
		return command.Execute()
	}
	if err := run("init"); err != nil {
		t.Fatal(err)
	}
	if err := run("environment", "init", "lab"); err != nil {
		t.Fatal(err)
	}
	if err := run("render", "--env", "lab", "--format", "toml"); err == nil {
		t.Fatal("expected unsupported format error")
	}
}
