package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ednillibanio/platformctl/internal/sharedroot"
)

func runCLI(t *testing.T, stdin string, args ...string) (string, string, error) {
	t.Helper()
	var out, errOut bytes.Buffer
	cmd := NewWithIO(&out, &errOut)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), errOut.String(), err
}

func TestBootstrapConfigUsageErrors(t *testing.T) {
	root := filepath.Join(t.TempDir(), "assembleiarr")
	cases := [][]string{
		{"bootstrap-config"},
		{"bootstrap-config", "--env", "lab", "--config-dir", "relative"},
		{"bootstrap-config", "--env", "lab", "--config-dir", root, "--bogus"},
		{"bootstrap-config", "--env", "lab", "--config-dir", root, "extra-arg"},
		{"bootstrap-config", "--env", "lab", "--config-dir", root, "--automation-key-import", "/path/key"},
		{"bootstrap-config", "--env", "lab", "--config-dir", root, "--automation-key", "--automation-key-import", "-"},
	}
	for _, args := range cases {
		_, _, err := runCLI(t, "", args...)
		if sharedroot.ExitCode(err) != sharedroot.ExitUsage {
			t.Errorf("%v: want usage error, got %v", args, err)
		}
	}
	if _, err := os.Lstat(root); !os.IsNotExist(err) {
		t.Fatal("a usage error wrote something")
	}
}

func TestBootstrapConfigDefaultDeclaration(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("shared-root commands are not supported on this operating system")
	}
	dir := t.TempDir()
	decl := "version: 1\nconsumer: provision-example\nhost:\n  - {name: HOST_IP, required: true, description: Address}\n"
	if err := os.WriteFile(filepath.Join(dir, "shared-root.yaml"), []byte(decl), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	root := filepath.Join(t.TempDir(), "assembleiarr")
	out, _, err := runCLI(t, "", "bootstrap-config", "--env", "lab", "--target", "vsphere-lab", "--config-dir", root)
	if err != nil {
		t.Fatalf("bootstrap-config: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, "lab", "targets", "vsphere-lab", "hosts", "provision-example.sh"))
	if err != nil || !strings.Contains(string(data), "# HOST_IP=") {
		t.Fatalf("default declaration not used: %v\n%s\n%s", err, data, out)
	}
}

func TestBootstrapConfigEnvVarRoot(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("shared-root commands are not supported on this operating system")
	}
	t.Chdir(t.TempDir())
	root := filepath.Join(t.TempDir(), "assembleiarr")
	t.Setenv(sharedroot.EnvVar, root)
	if _, _, err := runCLI(t, "", "bootstrap-config", "--env", "lab", "--dry-run"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := runCLI(t, "", "bootstrap-config", "--env", "lab"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "lab", "credentials")); err != nil {
		t.Fatalf("root from %s not used: %v", sharedroot.EnvVar, err)
	}
}

func TestBootstrapConfigNotSupported(t *testing.T) {
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		t.Skip("supported here")
	}
	_, _, err := runCLI(t, "", "bootstrap-config", "--env", "lab", "--config-dir", filepath.Join(t.TempDir(), "r"))
	if err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("want not supported, got %v", err)
	}
}
