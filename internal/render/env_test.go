package render

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ednillibanio/platformctl/internal/config"
)

func envFixture() config.RenderedConfig {
	return config.RenderedConfig{
		Values: map[string]any{
			"vsphere": map[string]any{
				"endpoint":            "vcenter.lab.internal",
				"insecure-connection": true,
				"folder":              "",
			},
			"common.vm_version": 19,
			"ratio":             0.75,
			"motd":              "it's\nmultiline",
		},
		Credentials: map[string]config.SecretRef{
			"vsphere-password": {File: "/srv/secrets/vsphere password"},
			"build-password":   {File: "~/.config/assembleiarr/lab/credentials/platform-modules/BUILD_PASSWORD"},
			"github-token":     {Env: "GITHUB_TOKEN"},
		},
	}
}

func TestEncodeEnvGolden(t *testing.T) {
	output, err := EncodeEnv(envFixture(), EnvOptions{Prefix: "PM_"})
	if err != nil {
		t.Fatal(err)
	}
	golden := filepath.Join("testdata", "export-env.golden")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(golden, output, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if string(output) != string(want) {
		t.Fatalf("export differs from %s (set UPDATE_GOLDEN=1 to refresh):\n%s", golden, output)
	}
	again, err := EncodeEnv(envFixture(), EnvOptions{Prefix: "PM_"})
	if err != nil || string(again) != string(output) {
		t.Fatalf("export is not deterministic: %v", err)
	}
}

func TestEncodeEnvRejectsUnexportableInput(t *testing.T) {
	cases := map[string]struct {
		cfg  config.RenderedConfig
		opts EnvOptions
		want string
	}{
		"list":        {cfg: config.RenderedConfig{Values: map[string]any{"hosts": []any{"a"}}}, want: "lists cannot be exported"},
		"null":        {cfg: config.RenderedConfig{Values: map[string]any{"x": nil}}, want: "null cannot be exported"},
		"empty map":   {cfg: config.RenderedConfig{Values: map[string]any{"x": map[string]any{}}}, want: "empty map"},
		"bad name":    {cfg: config.RenderedConfig{Values: map[string]any{"1st": "a"}}, want: "not a valid shell variable name"},
		"space name":  {cfg: config.RenderedConfig{Values: map[string]any{"a b": "a"}}, opts: EnvOptions{Prefix: "PM_"}, want: "not a valid shell variable name"},
		"bad prefix":  {cfg: config.RenderedConfig{}, opts: EnvOptions{Prefix: "PM-"}, want: "invalid prefix"},
		"relative":    {cfg: config.RenderedConfig{Credentials: map[string]config.SecretRef{"k": {File: "credentials/k"}}}, want: "must be absolute or start with ~/"},
		"inline":      {cfg: config.RenderedConfig{Credentials: map[string]config.SecretRef{"k": {Value: "s3cret"}}}, want: "--reveal-secrets"},
		"collision":   {cfg: config.RenderedConfig{Values: map[string]any{"a": map[string]any{"b": "1"}, "a-b": "2"}}, want: "both export A_B"},
		"value+cred":  {cfg: config.RenderedConfig{Values: map[string]any{"token": "x"}, Credentials: map[string]config.SecretRef{"token": {Env: "T"}}}, want: "both export TOKEN"},
		"unsupported": {cfg: config.RenderedConfig{Values: map[string]any{"x": struct{}{}}}, want: "unsupported type"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := EncodeEnv(tc.cfg, tc.opts)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error containing %q, got %v", tc.want, err)
			}
		})
	}
}

func TestEncodeEnvRevealSecrets(t *testing.T) {
	cfg := config.RenderedConfig{Credentials: map[string]config.SecretRef{"k": {Value: "it's"}}}
	output, err := EncodeEnv(cfg, EnvOptions{RevealSecrets: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(output), `export K='it'\''s'`) {
		t.Fatalf("inline value not quoted as expected:\n%s", output)
	}
}

// TestEncodeEnvEvaluatesInShells runs the projection through real shells to
// prove the contract: values survive quoting, ~/ resolves against HOME, and a
// missing secret exports nothing, returns non-zero, and keeps the shell alive.
func TestEncodeEnvEvaluatesInShells(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell evaluation is not exercised on Windows")
	}
	shells := []string{}
	for _, shell := range []string{"bash", "zsh", "sh"} {
		if _, err := exec.LookPath(shell); err == nil {
			shells = append(shells, shell)
		}
	}
	if len(shells) == 0 {
		t.Skip("no POSIX shell available")
	}
	home := t.TempDir()
	secretDir := filepath.Join(home, ".config", "assembleiarr", "lab", "credentials", "platform-modules")
	if err := os.MkdirAll(secretDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(secretDir, "BUILD_PASSWORD"), []byte("b'uild pass\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	absSecret := filepath.Join(t.TempDir(), "vsphere password")
	if err := os.WriteFile(absSecret, []byte("v$phere"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := envFixture()
	cfg.Credentials["vsphere-password"] = config.SecretRef{File: absSecret}
	good, err := EncodeEnv(cfg, EnvOptions{Prefix: "PM_"})
	if err != nil {
		t.Fatal(err)
	}
	cfg.Credentials["vsphere-password"] = config.SecretRef{File: filepath.Join(home, "missing")}
	missingFile, err := EncodeEnv(cfg, EnvOptions{Prefix: "PM_"})
	if err != nil {
		t.Fatal(err)
	}

	for _, shell := range shells {
		t.Run(shell, func(t *testing.T) {
			// "rc", not "status": status is read-only in zsh.
			script := `eval "$EXPORT"; rc=$?
printf 'status=%s\n' "$rc"
printf 'endpoint=[%s]\n' "${PM_VSPHERE_ENDPOINT-unset}"
printf 'insecure=[%s]\n' "${PM_VSPHERE_INSECURE_CONNECTION-unset}"
printf 'folder=[%s]\n' "${PM_VSPHERE_FOLDER-unset}"
printf 'version=[%s]\n' "${PM_COMMON_VM_VERSION-unset}"
printf 'motd=[%s]\n' "${PM_MOTD-unset}"
printf 'vsphere=[%s]\n' "${PM_VSPHERE_PASSWORD-unset}"
printf 'build=[%s]\n' "${PM_BUILD_PASSWORD-unset}"
printf 'github=[%s]\n' "${PM_GITHUB_TOKEN-unset}"
if command -v __platformctl_export >/dev/null 2>&1; then printf 'leaked\n'; fi
printf 'alive\n'
`
			run := func(export []byte, extraEnv ...string) string {
				t.Helper()
				command := exec.Command(shell, "-c", script)
				command.Env = append([]string{"HOME=" + home, "PATH=" + os.Getenv("PATH"), "EXPORT=" + string(export)}, extraEnv...)
				output, _ := command.CombinedOutput()
				return string(output)
			}

			output := run(good, "GITHUB_TOKEN=gh-token")
			for _, want := range []string{
				"status=0", "endpoint=[vcenter.lab.internal]", "insecure=[true]", "folder=[]",
				"version=[19]", "motd=[it's\nmultiline]", "vsphere=[v$phere]",
				"build=[b'uild pass]", "github=[gh-token]", "alive",
			} {
				if !strings.Contains(output, want) {
					t.Fatalf("missing %q in output:\n%s", want, output)
				}
			}
			if strings.Contains(output, "leaked") {
				t.Fatalf("helper function still defined after evaluation:\n%s", output)
			}

			for label, failing := range map[string]struct {
				export []byte
				env    []string
				reason string
			}{
				"missing file": {missingFile, []string{"GITHUB_TOKEN=gh-token"}, "cannot read secret file"},
				"unset env":    {good, nil, "GITHUB_TOKEN is not set"},
			} {
				output := run(failing.export, failing.env...)
				for _, want := range []string{"status=1", "endpoint=[unset]", "build=[unset]", failing.reason, "alive"} {
					if !strings.Contains(output, want) {
						t.Fatalf("%s: missing %q in output:\n%s", label, want, output)
					}
				}
				if strings.Contains(output, "leaked") {
					t.Fatalf("%s: helper function still defined after failure:\n%s", label, output)
				}
			}
		})
	}
}

func TestResolveServicesOrderAndErrors(t *testing.T) {
	bundle := exampleBundle(t)
	bundle.Services["metrics"] = config.Service{
		APIVersion: config.APIVersion,
		Kind:       "Service",
		Metadata:   config.Metadata{Name: "metrics"},
		Spec: config.ServiceSpec{LayerSpec: config.LayerSpec{
			Values: map[string]any{"http": map[string]any{"listen": "127.0.0.1"}},
		}},
	}
	resolved, err := ResolveServices(bundle, []string{"http", "metrics"})
	if err != nil {
		t.Fatal(err)
	}
	httpValues := resolved.Values["http"].(map[string]any)
	if httpValues["listen"] != "127.0.0.1" || httpValues["tls"] != false {
		t.Fatalf("service order or deep merge lost: %#v", httpValues)
	}
	if resolved.Values["domain"] != "lab.example.internal" || resolved.Values["locale"] != "en_US.UTF-8" {
		t.Fatalf("defaults/environment layers missing: %#v", resolved.Values)
	}
	if _, ok := resolved.Credentials["automation-token"]; !ok {
		t.Fatal("defaults credentials missing")
	}
	if _, err := ResolveServices(bundle, []string{"missing"}); err == nil || !strings.Contains(err.Error(), `service "missing" not found`) {
		t.Fatalf("expected missing service error, got %v", err)
	}
	if _, err := ResolveServices(bundle, []string{"http", "http"}); err == nil || !strings.Contains(err.Error(), "more than once") {
		t.Fatalf("expected duplicate service error, got %v", err)
	}
	rendered, err := Resolve(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := HostConfig(rendered, "nope"); err == nil {
		t.Fatal("expected missing host error")
	}
	app, err := HostConfig(rendered, "app")
	if err != nil || app.Values["workers"] != 4 {
		t.Fatalf("unexpected host config: %#v, %v", app, err)
	}
}
