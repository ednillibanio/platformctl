package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
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

func TestRenderRedactsInlineSecretsByDefault(t *testing.T) {
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
	if _, err := run("environment", "init", "lab"); err != nil {
		t.Fatal(err)
	}

	envRoot := filepath.Join(home, "environments", "lab")
	mustWriteFile(t, filepath.Join(envRoot, "profiles", "base.yaml"), `apiVersion: platformctl.io/v1alpha1
kind: Profile
metadata:
  name: base
spec: {}
`)
	mustWriteFile(t, filepath.Join(envRoot, "hosts", "app.yaml"), `apiVersion: platformctl.io/v1alpha1
kind: Host
metadata:
  name: app
spec:
  profile: base
  credentials:
    build-password:
      value: s0meSecret
`)

	redacted, err := run("render", "--env", "lab")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(redacted, "s0meSecret") {
		t.Fatalf("expected inline value to be redacted by default: %s", redacted)
	}
	if !strings.Contains(redacted, "<redacted>") {
		t.Fatalf("expected redaction placeholder in output: %s", redacted)
	}

	revealed, err := run("render", "--env", "lab", "--reveal-secrets")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(revealed, "s0meSecret") {
		t.Fatalf("expected --reveal-secrets to include the real value: %s", revealed)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
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

func TestExportCommand(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "examples", "platform-home"))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PLATFORMCTL_HOME", root)
	run := func(args ...string) (string, error) {
		t.Helper()
		var stdout, stderr bytes.Buffer
		command := NewWithIO(&stdout, &stderr)
		command.SetArgs(args)
		err := command.Execute()
		return stdout.String(), err
	}

	output, err := run("export", "--env", "lab", "--service", "http", "--prefix", "PM_")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"export PM_HTTP_LISTEN='0.0.0.0'",
		"export PM_DOMAIN='lab.example.internal'",
		`if [ -z "${PLATFORM_AUTOMATION_TOKEN:-}" ]`,
		`export PM_AUTOMATION_TOKEN="$__platformctl_s1"`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("missing %q in export:\n%s", want, output)
		}
	}

	for name, tc := range map[string]struct {
		args []string
		want string
	}{
		"host and service": {[]string{"export", "--env", "lab", "--host", "app", "--service", "http"}, "mutually exclusive"},
		"format":           {[]string{"export", "--env", "lab", "--format", "json"}, "unsupported format"},
		"unknown host":     {[]string{"export", "--env", "lab", "--host", "nope"}, `host "nope" not found`},
		// The example stores a relative file reference, which the env projection refuses.
		"relative file": {[]string{"export", "--env", "lab", "--host", "app"}, "must be absolute or start with ~/"},
	} {
		if _, err := run(tc.args...); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%s: expected error containing %q, got %v", name, tc.want, err)
		}
	}
}

func TestVersion(t *testing.T) {
	saved := Version
	t.Cleanup(func() { Version = saved })
	Version = "v9.8.7"
	var stdout bytes.Buffer
	command := NewWithIO(&stdout, &bytes.Buffer{})
	command.SetArgs([]string{"--version"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if got, want := stdout.String(), "platformctl version v9.8.7\n"; got != want {
		t.Fatalf("--version printed %q, want %q", got, want)
	}
}
