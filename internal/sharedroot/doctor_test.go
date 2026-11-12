//go:build linux || darwin

package sharedroot

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testDBPass   = "db-pass-value-0123456789"
	testGovcPass = "govc-pass-value-9876543210"
)

// healthyTree builds a complete, valid shared root for the two test
// declarations and returns its root.
func healthyTree(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "assembleiarr")
	opts, _, _ := bootstrapOpts(root)
	opts.AutomationKey = true
	if err := Bootstrap(opts); err != nil {
		t.Fatal(err)
	}
	vars := filepath.Join(root, "lab", "targets", "vsphere-lab", "vars.sh")
	writeKeep(t, vars, "GOVC_URL=https://vcenter.lab.test/sdk\nTARGET_FOLDER='/dc/vm/services'\n# TARGET_DATASTORE=\n")
	for _, c := range []string{"provision-semaphoreui", "provision-nexus3"} {
		writeKeep(t, filepath.Join(root, "lab", "targets", "vsphere-lab", "hosts", c+".sh"), "HOST_IP=192.0.2.10\n")
	}
	for _, o := range []CredentialOptions{
		{Consumer: "provision-semaphoreui", Specs: []string{"SEMAPHORE_DB_PASS=-"}, Stdin: strings.NewReader(testDBPass)},
		{Target: "vsphere-lab", Identity: "vcenter-access-admin", Specs: []string{"GOVC_PASSWORD=-"}, Stdin: strings.NewReader(testGovcPass)},
	} {
		o.Root, o.Env, o.Out = root, "lab", &bytes.Buffer{}
		if err := Credential(o); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// writeKeep replaces a file's content and keeps its mode.
func writeKeep(t *testing.T, path, content string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), info.Mode().Perm()); err != nil {
		t.Fatal(err)
	}
}

func doctorDecls(t *testing.T) []string {
	t.Helper()
	dir := t.TempDir()
	semaphore, err := os.ReadFile(testDecls[0])
	if err != nil {
		t.Fatal(err)
	}
	semaphore = append(semaphore, []byte("credentials:\n  - name: SEMAPHORE_DB_PASS\n    required: true\n    description: Database password\n")...)
	path := filepath.Join(dir, "semaphoreui.yaml")
	if err := os.WriteFile(path, semaphore, 0o644); err != nil {
		t.Fatal(err)
	}
	return []string{path, testDecls[1]}
}

func runDoctor(t *testing.T, root string, decls []string, format string) (Report, string, error) {
	t.Helper()
	var out bytes.Buffer
	err := Doctor(DoctorOptions{Root: root, Env: "lab", Declarations: decls, Format: format, Out: &out})
	var report Report
	if format == "json" {
		if jerr := json.Unmarshal(out.Bytes(), &report); jerr != nil {
			t.Fatalf("JSON does not parse: %v\n%s", jerr, out.String())
		}
	}
	return report, out.String(), err
}

func TestDoctorHealthyTree(t *testing.T) {
	root := healthyTree(t)
	ageTree(t, root)
	before := snapshot(t, root)
	report, out, err := runDoctor(t, root, doctorDecls(t), "json")
	if err != nil || !report.OK || len(report.Problems) != 0 {
		t.Fatalf("healthy tree: %v\n%s", err, out)
	}
	assertSameSnapshot(t, before, snapshot(t, root))
	tr := report.Targets["vsphere-lab"]
	if tr == nil || tr.Vars["GOVC_URL"] != "https://vcenter.lab.test/sdk" || tr.Hosts["provision-nexus3"]["HOST_IP"] != "192.0.2.10" {
		t.Errorf("non-secret values missing: %+v", tr)
	}
	if got := tr.Credentials["vcenter-access-admin"]; len(got) != 1 || got[0] != "GOVC_PASSWORD" {
		t.Errorf("target credential names: %v", got)
	}
	if got := report.Credentials["provision-semaphoreui"]; len(got) != 1 || got[0] != "SEMAPHORE_DB_PASS" {
		t.Errorf("credential names: %v", got)
	}
	if report.AutomationKey == nil || !strings.HasPrefix(report.AutomationKey.Fingerprint, "SHA256:") {
		t.Errorf("fingerprint missing: %+v", report.AutomationKey)
	}
	_, text, err := runDoctor(t, root, doctorDecls(t), "text")
	if err != nil || !strings.Contains(text, "0 problem(s)") {
		t.Fatalf("text: %v\n%s", err, text)
	}
}

func TestDoctorNeverPrintsSecrets(t *testing.T) {
	root := healthyTree(t)
	private, err := os.ReadFile(filepath.Join(root, "lab", "credentials", "automation", "ssh_ed25519"))
	if err != nil {
		t.Fatal(err)
	}
	// Leak both secrets into shared files so findings are about them.
	vars := filepath.Join(root, "lab", "targets", "vsphere-lab", "vars.sh")
	writeKeep(t, vars, "GOVC_URL=https://vcenter.lab.test/sdk\nTARGET_FOLDER="+testGovcPass+"\n")
	writeKeep(t, filepath.Join(root, "lab", "targets", "vsphere-lab", "hosts", "provision-semaphoreui.sh"), "HOST_IP="+testDBPass+"\n")
	secrets := []string{testDBPass, testGovcPass, string(bytes.TrimSpace(private))}
	for _, line := range strings.Split(string(private), "\n")[1:3] {
		secrets = append(secrets, line)
	}
	for _, format := range []string{"text", "json"} {
		_, out, err := runDoctor(t, root, doctorDecls(t), format)
		if err == nil {
			t.Fatalf("%s: leaked values not reported", format)
		}
		for _, s := range secrets {
			if s != "" && strings.Contains(out, s) {
				t.Errorf("%s output holds a secret value", format)
			}
		}
		if strings.Count(out, "holds the value of credential") < 2 {
			t.Errorf("%s: leaks not reported:\n%s", format, out)
		}
	}
	report, _, _ := runDoctor(t, root, doctorDecls(t), "json")
	if report.Targets["vsphere-lab"].Vars["GOVC_URL"] != WithheldValue {
		t.Error("values of a leaking file are not withheld")
	}
}

func TestDoctorWithholdsValuesOnNameMatch(t *testing.T) {
	root := healthyTree(t)
	writeKeep(t, filepath.Join(root, "lab", "targets", "vsphere-lab", "vars.sh"),
		"GOVC_URL=https://x\nTARGET_FOLDER=x\n# GOVC_PASSWORD lives here\n")
	report, out, err := runDoctor(t, root, doctorDecls(t), "json")
	if err == nil || report.Targets["vsphere-lab"].Vars["GOVC_URL"] != WithheldValue {
		t.Fatalf("values not withheld on a name match: %v\n%s", err, out)
	}
}

// TestDoctorChecks breaks a healthy tree in one way per case and expects a
// problem whose text holds want.
func TestDoctorChecks(t *testing.T) {
	env := func(root string, rel ...string) string {
		return filepath.Join(append([]string{root, "lab"}, rel...)...)
	}
	cases := []struct {
		name   string
		decls  bool
		break_ func(t *testing.T, root string)
		want   string
	}{
		{"directory mode", false, func(t *testing.T, r string) { chmod(t, env(r, "credentials"), 0o755) }, "mode 0755, want 0700"},
		{"shared file mode", false, func(t *testing.T, r string) { chmod(t, env(r, "targets", "vsphere-lab", "vars.sh"), 0o600) }, "mode 0600, want 0644"},
		{"credential mode", false, func(t *testing.T, r string) {
			chmod(t, env(r, "credentials", "provision-semaphoreui", "SEMAPHORE_DB_PASS"), 0o644)
		}, "mode 0644, want 0600"},
		{"private key mode", false, func(t *testing.T, r string) { chmod(t, env(r, "credentials", "automation", "ssh_ed25519"), 0o640) }, "mode 0640, want 0600"},
		{"symlink", false, func(t *testing.T, r string) {
			symlink(t, "/etc/hostname", env(r, "targets", "vsphere-lab", "ca.pem"))
		}, "symbolic link"},
		{"symlinked directory", false, func(t *testing.T, r string) {
			symlink(t, t.TempDir(), env(r, "credentials", "provision-other"))
		}, "symbolic link"},
		{"command substitution", false, func(t *testing.T, r string) {
			writeKeep(t, env(r, "targets", "vsphere-lab", "vars.sh"), "GOVC_URL=$(id)\n")
		}, "only KEY=value lines"},
		{"unquoted tilde", false, func(t *testing.T, r string) {
			writeKeep(t, env(r, "targets", "vsphere-lab", "vars.sh"), "TARGET_FOLDER=~/x\n")
		}, "only KEY=value lines"},
		{"duplicate key", false, func(t *testing.T, r string) {
			writeKeep(t, env(r, "targets", "vsphere-lab", "vars.sh"), "GOVC_URL=a\nGOVC_URL=b\n")
		}, "key set twice"},
		{"host key in vars.sh", false, func(t *testing.T, r string) {
			writeKeep(t, env(r, "targets", "vsphere-lab", "vars.sh"), "HOST_IP=192.0.2.1\n")
		}, "vars.sh holds only"},
		{"target key in hosts file", false, func(t *testing.T, r string) {
			writeKeep(t, env(r, "targets", "vsphere-lab", "hosts", "provision-nexus3.sh"), "GOVC_URL=x\n")
		}, "a hosts file holds only HOST_*"},
		{"required key placeholder", true, func(t *testing.T, r string) {
			writeKeep(t, env(r, "targets", "vsphere-lab", "vars.sh"), "GOVC_URL=https://x\n# TARGET_FOLDER=\n")
		}, "TARGET_FOLDER: required key is still a placeholder"},
		{"required key empty", true, func(t *testing.T, r string) {
			writeKeep(t, env(r, "targets", "vsphere-lab", "hosts", "provision-nexus3.sh"), "HOST_IP=\n")
		}, "HOST_IP: required key is empty"},
		{"required key missing", true, func(t *testing.T, r string) {
			writeKeep(t, env(r, "targets", "vsphere-lab", "vars.sh"), "TARGET_FOLDER=x\n")
		}, "GOVC_URL: required key missing"},
		{"required credential missing", true, func(t *testing.T, r string) {
			remove(t, env(r, "credentials", "provision-semaphoreui", "SEMAPHORE_DB_PASS"))
		}, "required credential of provision-semaphoreui missing"},
		{"required target credential missing", true, func(t *testing.T, r string) {
			remove(t, env(r, "credentials", "target-vsphere-lab", "vcenter-access-admin", "GOVC_PASSWORD"))
		}, "required target credential of provision-semaphoreui missing"},
		{"declared key pair missing", true, func(t *testing.T, r string) {
			remove(t, env(r, "credentials", "automation"))
		}, "uses the automation key pair, which is missing"},
		{"credential name in shared file", true, func(t *testing.T, r string) {
			writeKeep(t, env(r, "targets", "vsphere-lab", "vars.sh"), "GOVC_URL=https://x\nTARGET_FOLDER=x\n# GOVC_PASSWORD is secret\n")
		}, "names a declared credential"},
		{"credential value in shared file", true, func(t *testing.T, r string) {
			write(t, env(r, "targets", "vsphere-lab", "ca.pem"), "x"+testDBPass+"x", 0o644)
		}, "holds the value of credential SEMAPHORE_DB_PASS"},
		{"credential value in unknown readable file", true, func(t *testing.T, r string) {
			mkdir(t, env(r, "extra"))
			write(t, env(r, "extra", "notes.txt"), "password: "+testGovcPass, 0o644)
		}, "notes.txt: GOVC_PASSWORD: a file readable by others holds the value"},
		{"credential value in mis-moded credential file", true, func(t *testing.T, r string) {
			write(t, env(r, "credentials", "provision-semaphoreui", "COPY"), testDBPass, 0o640)
		}, "COPY: SEMAPHORE_DB_PASS: a file readable by others holds the value"},
		{"key pair mismatch", false, func(t *testing.T, r string) {
			other, err := GenerateAutomationKey("lab")
			if err != nil {
				t.Fatal(err)
			}
			writeKeep(t, env(r, "credentials", "automation", "ssh_ed25519.pub"), string(other.Public))
		}, "does not match"},
		{"key pair incomplete", false, func(t *testing.T, r string) {
			remove(t, env(r, "credentials", "automation", "ssh_ed25519.pub"))
		}, "incomplete automation key pair"},
		{"temporary file", false, func(t *testing.T, r string) {
			write(t, env(r, "credentials", "provision-semaphoreui", TempPrefix+"abc"), "x", 0o600)
		}, "temporary entry left by an interrupted"},
		{"temporary directory", false, func(t *testing.T, r string) {
			mkdir(t, env(r, "credentials", TempPrefix+"abc"))
		}, "temporary entry left by an interrupted"},
		{"empty credential", false, func(t *testing.T, r string) {
			write(t, env(r, "credentials", "provision-semaphoreui", "EMPTY"), "", 0o600)
		}, "empty credential file"},
		{"file directly under target credentials", false, func(t *testing.T, r string) {
			write(t, env(r, "credentials", "target-vsphere-lab", "GOVC_PASSWORD"), "x", 0o600)
		}, "a file directly under credentials/target-vsphere-lab/"},
		{"reserved identity directory", false, func(t *testing.T, r string) {
			mkdir(t, env(r, "credentials", "target-vsphere-lab", "provision-semaphoreui"))
		}, `identity directory "provision-semaphoreui" is reserved`},
		{"deprecated identity directory", false, func(t *testing.T, r string) {
			mkdir(t, env(r, "credentials", "target-vsphere-lab", "terraform-provisioner"))
		}, `identity directory "terraform-provisioner" is deprecated`},
		{"invalid identity directory name", false, func(t *testing.T, r string) {
			mkdir(t, env(r, "credentials", "target-vsphere-lab", "Admin"))
		}, `identity directory "Admin" is not a valid identity name`},
		{"unknown identity directory", false, func(t *testing.T, r string) {
			mkdir(t, env(r, "credentials", "target-vsphere-lab", "admin"))
		}, "is not an active identity"},
		{"symlink inside inactive identity directory", false, func(t *testing.T, r string) {
			mkdir(t, env(r, "credentials", "target-vsphere-lab", "admin"))
			symlink(t, "/etc/hostname", env(r, "credentials", "target-vsphere-lab", "admin", "PASS"))
		}, "admin/PASS: symbolic link"},
		{"file under target credentials readable by others", true, func(t *testing.T, r string) {
			write(t, env(r, "credentials", "target-vsphere-lab", "LOOSE"), testGovcPass, 0o644)
		}, "LOOSE: GOVC_PASSWORD: a file readable by others holds the value"},
		{"missing vars.sh", false, func(t *testing.T, r string) {
			remove(t, env(r, "targets", "vsphere-lab", "vars.sh"))
		}, "vars.sh: missing"},
		{"missing hosts file of declared consumer", true, func(t *testing.T, r string) {
			remove(t, env(r, "targets", "vsphere-lab", "hosts", "provision-nexus3.sh"))
		}, "missing hosts file of provision-nexus3"},
		{"unknown directory mode", false, func(t *testing.T, r string) {
			mkdir(t, env(r, "extra", "deeper"))
			chmod(t, env(r, "extra", "deeper"), 0o755)
		}, "deeper: mode 0755, want 0700"},
		{"symlink inside unknown directory", false, func(t *testing.T, r string) {
			mkdir(t, env(r, "credentials", "provision-x", "sub"))
			symlink(t, "/etc/hostname", env(r, "credentials", "provision-x", "sub", "link"))
		}, "link: symbolic link"},
		{"temporary entry inside unknown directory", false, func(t *testing.T, r string) {
			mkdir(t, env(r, "targets", "vsphere-lab", "misc"))
			write(t, env(r, "targets", "vsphere-lab", "misc", TempPrefix+"x"), "x", 0o600)
		}, "temporary entry left by an interrupted"},
		{"state file mode", false, func(t *testing.T, r string) {
			mkdir(t, env(r, "state", "provision-semaphoreui", "vsphere-lab"))
			write(t, env(r, "state", "provision-semaphoreui", "vsphere-lab", "terraform.tfstate"), "{}", 0o644)
		}, "mode 0644, want 0600"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := healthyTree(t)
			tc.break_(t, root)
			var decls []string
			if tc.decls {
				decls = doctorDecls(t)
			}
			report, out, err := runDoctor(t, root, decls, "json")
			if ExitCode(err) != ExitProblem || report.OK {
				t.Fatalf("want exit 1, got %v\n%s", err, out)
			}
			found := false
			for _, p := range report.Problems {
				if strings.Contains(p.String(), tc.want) {
					found = true
				}
			}
			if !found {
				t.Errorf("no problem with %q:\n%s", tc.want, out)
			}
		})
	}
}

func TestDoctorInvalidDeclaration(t *testing.T) {
	root := healthyTree(t)
	bad := filepath.Join(t.TempDir(), "shared-root.yaml")
	write(t, bad, "version: 1\nconsumer: provision-x\nbogus: true\n", 0o644)
	report, out, err := runDoctor(t, root, []string{bad}, "json")
	if ExitCode(err) != ExitProblem || len(report.Problems) == 0 || !strings.Contains(report.Problems[0].Message, "invalid declaration") {
		t.Fatalf("want invalid declaration problem: %v\n%s", err, out)
	}
}

func TestDoctorWarnings(t *testing.T) {
	root := healthyTree(t)
	env := func(rel ...string) string { return filepath.Join(append([]string{root, "lab"}, rel...)...) }
	writeKeep(t, env("targets", "vsphere-lab", "vars.sh"), "GOVC_URL=https://x\nTARGET_FOLDER=x\nTARGET_EXTRA=1\n")
	write(t, env("credentials", "provision-semaphoreui", "UNDECLARED"), "value", 0o600)
	write(t, env("notes.txt"), "x", 0o600)
	report, out, err := runDoctor(t, root, doctorDecls(t), "json")
	if err != nil {
		t.Fatalf("warnings alone must not fail: %v\n%s", err, out)
	}
	for _, want := range []string{"TARGET_EXTRA: key not declared", "UNDECLARED: credential not declared", "not part of the shared-root layout"} {
		found := false
		for _, w := range report.Warnings {
			if strings.Contains(w.String(), want) {
				found = true
			}
		}
		if !found {
			t.Errorf("no warning %q:\n%s", want, out)
		}
	}
}

func TestDoctorMissingRootAndEnv(t *testing.T) {
	root := filepath.Join(t.TempDir(), "missing")
	if _, out, err := runDoctor(t, root, nil, "text"); ExitCode(err) != ExitProblem || !strings.Contains(out, "does not exist") {
		t.Fatalf("missing root: %v\n%s", err, out)
	}
	if _, err := os.Lstat(root); !os.IsNotExist(err) {
		t.Fatal("doctor created the root")
	}
	root = t.TempDir()
	if _, out, err := runDoctor(t, root, nil, "text"); ExitCode(err) != ExitProblem || !strings.Contains(out, "environment does not exist") {
		t.Fatalf("missing env: %v\n%s", err, out)
	}
}

func TestDoctorTargetScope(t *testing.T) {
	root := healthyTree(t)
	var out bytes.Buffer
	err := Doctor(DoctorOptions{Root: root, Env: "lab", Targets: []string{"other"}, Out: &out})
	if ExitCode(err) != ExitProblem || !strings.Contains(out.String(), "target directory missing") {
		t.Fatalf("unknown target: %v\n%s", err, out.String())
	}
	out.Reset()
	err = Doctor(DoctorOptions{Root: root, Env: "lab", Consumers: []string{"provision-glpi"}, Out: &out})
	if ExitCode(err) != ExitProblem || !strings.Contains(out.String(), "missing credentials directory of provision-glpi") {
		t.Fatalf("consumer: %v\n%s", err, out.String())
	}
}

func TestDoctorUsageErrors(t *testing.T) {
	for name, opts := range map[string]DoctorOptions{
		"env":      {Env: "dev"},
		"target":   {Env: "lab", Targets: []string{"../x"}},
		"consumer": {Env: "lab", Consumers: []string{"automation"}},
		"format":   {Env: "lab", Format: "yaml"},
	} {
		opts.Root, opts.Out = t.TempDir(), &bytes.Buffer{}
		if err := Doctor(opts); ExitCode(err) != ExitUsage {
			t.Errorf("%s: want usage error, got %v", name, err)
		}
	}
}

func TestDoctorOwner(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("changing the owner needs root")
	}
	root := healthyTree(t)
	path := filepath.Join(root, "lab", "targets", "vsphere-lab", "vars.sh")
	if err := os.Lchown(path, 4242, 4242); err != nil {
		t.Fatal(err)
	}
	_, out, err := runDoctor(t, root, nil, "text")
	if ExitCode(err) != ExitProblem || !strings.Contains(out, "owned by uid 4242") {
		t.Fatalf("owner not checked: %v\n%s", err, out)
	}
}

func chmod(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}

func symlink(t *testing.T, target, path string) {
	t.Helper()
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
}

func remove(t *testing.T, path string) {
	t.Helper()
	if err := os.RemoveAll(path); err != nil {
		t.Fatal(err)
	}
}

func write(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
	chmod(t, path, mode)
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o700); err != nil {
		t.Fatal(err)
	}
}
