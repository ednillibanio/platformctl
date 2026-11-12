package cli

import (
	"bytes"
	"encoding/json"
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

func TestCredentialUsageErrors(t *testing.T) {
	root := filepath.Join(t.TempDir(), "assembleiarr")
	cases := [][]string{
		{"credential", "--consumer", "provision-example", "--config-dir", root, "A"},
		{"credential", "--env", "lab", "--consumer", "provision-example", "--config-dir", "relative", "A"},
		{"credential", "--env", "lab", "--consumer", "provision-example", "--config-dir", root, "--bogus", "A"},
		{"credential", "--env", "lab", "--consumer", "provision-example", "--config-dir", root, "--length", "0", "A"},
		{"credential", "--env", "lab", "--consumer", "provision-example", "--config-dir", root, "--length", "x", "A"},
		{"credential", "--env", "lab", "--consumer", "provision-example", "--config-dir", root},
		{"credential", "--env", "lab", "--target", "vsphere-lab", "--config-dir", root, "A"},
		{"credential", "--env", "lab", "--target", "vsphere-lab", "--identity", "packer-builder", "--config-dir", root, "A"},
	}
	for _, args := range cases {
		_, _, err := runCLI(t, "secret", args...)
		if sharedroot.ExitCode(err) != sharedroot.ExitUsage {
			t.Errorf("%v: want usage error, got %v", args, err)
		}
	}
	if _, err := os.Lstat(root); !os.IsNotExist(err) {
		t.Fatal("a usage error wrote something")
	}
}

func TestCredentialCommand(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("shared-root commands are not supported on this operating system")
	}
	root := filepath.Join(t.TempDir(), "assembleiarr")
	t.Setenv(sharedroot.EnvVar, root)
	args := []string{"credential", "--env", "lab", "--target", "vsphere-lab", "--identity", "terraform-provisioner",
		"GOVC_USERNAME=svc", "GOVC_PASSWORD=-", "--length", "16", "RANDOM_KEY"}
	out, _, err := runCLI(t, "pw-from-stdin\n", args...)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "pw-from-stdin") || strings.Contains(out, "svc\n") {
		t.Fatalf("output holds a value:\n%s", out)
	}
	dir := filepath.Join(root, "lab", "credentials", "target-vsphere-lab", "terraform-provisioner")
	if data, err := os.ReadFile(filepath.Join(dir, "GOVC_PASSWORD")); err != nil || string(data) != "pw-from-stdin" {
		t.Fatalf("GOVC_PASSWORD: %q %v", data, err)
	}
	if data, err := os.ReadFile(filepath.Join(dir, "RANDOM_KEY")); err != nil || len(data) != 24 {
		t.Fatalf("RANDOM_KEY: want base64 of 16 bytes, got %q %v", data, err)
	}
	out, _, err = runCLI(t, "other", args...)
	if err != nil || strings.Count(out, "exists") != 3 {
		t.Fatalf("second run: %v\n%s", err, out)
	}
}

func TestCredentialNotSupported(t *testing.T) {
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		t.Skip("supported here")
	}
	_, _, err := runCLI(t, "", "credential", "--env", "lab", "--consumer", "provision-example",
		"--config-dir", filepath.Join(t.TempDir(), "r"), "A")
	if err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("want not supported, got %v", err)
	}
}

func TestDoctorUsageErrors(t *testing.T) {
	root := filepath.Join(t.TempDir(), "assembleiarr")
	for _, args := range [][]string{
		{"doctor"},
		{"doctor", "--env", "lab", "--config-dir", root, "--format", "yaml"},
		{"doctor", "--env", "lab", "--config-dir", root, "extra"},
		{"doctor", "--env", "lab", "--config-dir", root, "--dry-run"},
	} {
		if _, _, err := runCLI(t, "", args...); sharedroot.ExitCode(err) != sharedroot.ExitUsage {
			t.Errorf("%v: want usage error, got %v", args, err)
		}
	}
}

func TestDoctorCommand(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("shared-root commands are not supported on this operating system")
	}
	t.Chdir(t.TempDir())
	root := filepath.Join(t.TempDir(), "assembleiarr")
	t.Setenv(sharedroot.EnvVar, root)
	if _, _, err := runCLI(t, "", "bootstrap-config", "--env", "lab", "--target", "vsphere-lab"); err != nil {
		t.Fatal(err)
	}
	out, _, err := runCLI(t, "", "doctor", "--env", "lab", "--format", "json")
	if err != nil {
		t.Fatalf("doctor on a fresh skeleton: %v\n%s", err, out)
	}
	var report sharedroot.Report
	if err := json.Unmarshal([]byte(out), &report); err != nil || !report.OK {
		t.Fatalf("JSON: %v\n%s", err, out)
	}
	if err := os.Chmod(filepath.Join(root, "lab", "credentials"), 0o755); err != nil {
		t.Fatal(err)
	}
	out, _, err = runCLI(t, "", "doctor", "--env", "lab")
	if sharedroot.ExitCode(err) != sharedroot.ExitProblem || !strings.Contains(out, "mode 0755, want 0700") {
		t.Fatalf("want exit 1 with the mode problem: %v\n%s", err, out)
	}
}

func TestDoctorNotSupported(t *testing.T) {
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		t.Skip("supported here")
	}
	_, _, err := runCLI(t, "", "doctor", "--env", "lab", "--config-dir", filepath.Join(t.TempDir(), "r"))
	if err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("want not supported, got %v", err)
	}
}
