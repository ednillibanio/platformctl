//go:build linux || darwin

package sharedroot

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

var update = flag.Bool("update", false, "rewrite the golden files")

var testDecls = []string{
	filepath.Join("testdata", "bootstrap", "semaphoreui.yaml"),
	filepath.Join("testdata", "bootstrap", "nexus3.yaml"),
}

func bootstrapOpts(root string) (BootstrapOptions, *bytes.Buffer, *bytes.Buffer) {
	var out, errOut bytes.Buffer
	return BootstrapOptions{
		Root: root, Env: "lab", Targets: []string{"vsphere-lab"},
		Declarations: testDecls, Out: &out, Err: &errOut,
	}, &out, &errOut
}

// snapshot maps every path under root to its mode, mtime and content.
func snapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		entry := info.Mode().String() + " " + info.ModTime().Format(time.RFC3339Nano)
		if info.Mode().IsRegular() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			entry += " " + string(data)
		}
		rel, _ := filepath.Rel(root, path)
		out[rel] = entry
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// ageTree moves every mtime an hour back, so any write shows in a snapshot.
func ageTree(t *testing.T, root string) {
	t.Helper()
	old := time.Now().Add(-time.Hour)
	var paths []string
	_ = filepath.WalkDir(root, func(path string, _ fs.DirEntry, err error) error {
		paths = append(paths, path)
		return err
	})
	sort.Sort(sort.Reverse(sort.StringSlice(paths)))
	for _, p := range paths {
		if err := os.Chtimes(p, old, old); err != nil {
			t.Fatal(err)
		}
	}
}

func assertSameSnapshot(t *testing.T, before, after map[string]string) {
	t.Helper()
	for path, entry := range after {
		if before[path] != entry {
			t.Errorf("changed or created: %s", path)
		}
	}
	for path := range before {
		if _, ok := after[path]; !ok {
			t.Errorf("removed: %s", path)
		}
	}
}

func TestBootstrapGolden(t *testing.T) {
	root := filepath.Join(t.TempDir(), "assembleiarr")
	opts, out, errOut := bootstrapOpts(root)
	if err := Bootstrap(opts); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	files := map[string]string{
		"vars.sh.golden":                     "lab/targets/vsphere-lab/vars.sh",
		"hosts-provision-semaphoreui.golden": "lab/targets/vsphere-lab/hosts/provision-semaphoreui.sh",
		"hosts-provision-nexus3.golden":      "lab/targets/vsphere-lab/hosts/provision-nexus3.sh",
	}
	got := map[string][]byte{"output.golden": []byte(strings.ReplaceAll(out.String(), root, "<root>"))}
	for golden, rel := range files {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		got[golden] = data
	}
	for golden, data := range got {
		path := filepath.Join("testdata", "bootstrap", golden)
		if *update {
			if err := os.WriteFile(path, data, 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%v (run go test -update)", err)
		}
		if !bytes.Equal(data, want) {
			t.Errorf("%s differs:\n--- got\n%s\n--- want\n%s", golden, data, want)
		}
	}
	wantWarn := "warning: provision-semaphoreui uses the automation key pair: run with --automation-key on the first controller of lab, --automation-key-import - on the others\n"
	if errOut.String() != wantWarn {
		t.Errorf("warnings %q, want %q", errOut, wantWarn)
	}
	for _, absent := range []string{"lab/state", "lab/backups", "lab/credentials/automation"} {
		if _, err := os.Lstat(filepath.Join(root, absent)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s exists", absent)
		}
	}
}

func TestBootstrapSkeletonParsesAsPlaceholders(t *testing.T) {
	root := filepath.Join(t.TempDir(), "assembleiarr")
	opts, _, _ := bootstrapOpts(root)
	if err := Bootstrap(opts); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		rel  string
		kind FileKind
		keys []string
	}{
		{"lab/targets/vsphere-lab/vars.sh", VarsFile, []string{"GOVC_URL", "TARGET_FOLDER", "TARGET_DATASTORE"}},
		{"lab/targets/vsphere-lab/hosts/provision-semaphoreui.sh", HostsFile, []string{"HOST_IP"}},
	}
	for _, tc := range cases {
		data, _ := os.ReadFile(filepath.Join(root, tc.rel))
		parsed, err := ParseShellFile(tc.rel, tc.kind, data)
		if err != nil {
			t.Fatalf("%s: %v", tc.rel, err)
		}
		if len(parsed.Assignments) != 0 {
			t.Errorf("%s: assigns values: %+v", tc.rel, parsed.Assignments)
		}
		var keys []string
		for _, p := range parsed.Placeholders {
			keys = append(keys, p.Key)
		}
		if strings.Join(keys, ",") != strings.Join(tc.keys, ",") {
			t.Errorf("%s: placeholders %v, want %v", tc.rel, keys, tc.keys)
		}
	}
}

func TestBootstrapSecondRunChangesNothing(t *testing.T) {
	root := filepath.Join(t.TempDir(), "assembleiarr")
	opts, _, _ := bootstrapOpts(root)
	opts.AutomationKey = true
	if err := Bootstrap(opts); err != nil {
		t.Fatal(err)
	}
	ageTree(t, root)
	before := snapshot(t, root)
	opts, out, _ := bootstrapOpts(root)
	opts.AutomationKey = true
	if err := Bootstrap(opts); err != nil {
		t.Fatalf("second run: %v", err)
	}
	assertSameSnapshot(t, before, snapshot(t, root))
	if strings.Contains(out.String(), "created") {
		t.Errorf("second run reports a creation:\n%s", out)
	}
}

func TestBootstrapDryRunTouchesNothing(t *testing.T) {
	root := filepath.Join(t.TempDir(), "assembleiarr")
	opts, out, _ := bootstrapOpts(root)
	opts.DryRun, opts.AutomationKey = true, true
	if err := Bootstrap(opts); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dry run created the root: %v", err)
	}
	if !strings.Contains(out.String(), "would create") || strings.Contains(out.String(), "created ") {
		t.Fatalf("dry run output:\n%s", out)
	}
	// On a partial tree, a dry run changes nothing either.
	if err := os.MkdirAll(filepath.Join(root, "lab", "targets"), 0o700); err != nil {
		t.Fatal(err)
	}
	ageTree(t, root)
	before := snapshot(t, root)
	opts, out, _ = bootstrapOpts(root)
	opts.DryRun, opts.AutomationKey = true, true
	if err := Bootstrap(opts); err != nil {
		t.Fatal(err)
	}
	assertSameSnapshot(t, before, snapshot(t, root))
	if strings.Contains(out.String(), " "+filepath.Join(root, "lab", "targets")+"/\n") {
		t.Fatalf("existing directory reported as missing:\n%s", out)
	}
}

func TestBootstrapInvalidDeclarationWritesNothing(t *testing.T) {
	root := filepath.Join(t.TempDir(), "assembleiarr")
	bad := filepath.Join(t.TempDir(), "shared-root.yaml")
	if err := os.WriteFile(bad, []byte("version: 1\nconsumer: provision-x\nextra: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	opts, _, _ := bootstrapOpts(root)
	opts.Declarations = append(opts.Declarations, bad)
	err := Bootstrap(opts)
	if ExitCode(err) != ExitProblem || !strings.Contains(err.Error(), "extra: unknown field") {
		t.Fatalf("want a declaration problem, got %v", err)
	}
	if _, err := os.Lstat(root); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("something was written")
	}
}

func TestBootstrapUsageErrors(t *testing.T) {
	root := filepath.Join(t.TempDir(), "assembleiarr")
	mutate := []func(*BootstrapOptions){
		func(o *BootstrapOptions) { o.Env = "homol" },
		func(o *BootstrapOptions) { o.Targets = []string{"Bad_Target"} },
		func(o *BootstrapOptions) { o.Consumers = []string{"semaphoreui"} },
		func(o *BootstrapOptions) { o.AutomationKey = true; o.ImportKey = strings.NewReader("") },
		func(o *BootstrapOptions) { o.Declarations = append(o.Declarations, testDecls[0]) },
	}
	for i, m := range mutate {
		opts, _, _ := bootstrapOpts(root)
		m(&opts)
		if err := Bootstrap(opts); ExitCode(err) != ExitUsage {
			t.Errorf("case %d: want usage error, got %v", i, err)
		}
	}
	if _, err := os.Lstat(root); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("a usage error wrote something")
	}
}

func TestBootstrapExistingFileLacksKeys(t *testing.T) {
	root := filepath.Join(t.TempDir(), "assembleiarr")
	dir := filepath.Join(root, "lab", "targets", "vsphere-lab")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	vars := filepath.Join(dir, "vars.sh")
	content := "GOVC_URL=https://vcenter.example.invalid/sdk\n# TARGET_DATASTORE=\n"
	if err := os.WriteFile(vars, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	opts, out, _ := bootstrapOpts(root)
	if err := Bootstrap(opts); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(vars); string(data) != content {
		t.Fatalf("existing file changed:\n%s", data)
	}
	got := out.String()
	if !strings.Contains(got, vars+" lacks declared keys") || !strings.Contains(got, "# TARGET_FOLDER=") {
		t.Fatalf("missing lines not printed:\n%s", got)
	}
	if strings.Contains(got, "# GOVC_URL=") || strings.Contains(got, "# TARGET_DATASTORE=") {
		t.Fatalf("present keys printed as missing:\n%s", got)
	}
}

func TestBootstrapEmptyExistingFileIsKept(t *testing.T) {
	root := filepath.Join(t.TempDir(), "assembleiarr")
	dir := filepath.Join(root, "lab", "targets", "vsphere-lab")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "vars.sh"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	opts, _, _ := bootstrapOpts(root)
	if err := Bootstrap(opts); err != nil {
		t.Fatal(err)
	}
	ageTree(t, root)
	before := snapshot(t, root)
	opts, out, _ := bootstrapOpts(root)
	if err := Bootstrap(opts); err != nil {
		t.Fatal(err)
	}
	assertSameSnapshot(t, before, snapshot(t, root))
	if data, _ := os.ReadFile(filepath.Join(dir, "vars.sh")); len(data) != 0 {
		t.Fatal("empty file was written")
	}
	if !strings.Contains(out.String(), "vars.sh lacks declared keys") {
		t.Fatalf("missing keys of the empty file not printed:\n%s", out)
	}
}

func TestBootstrapGenericSkeleton(t *testing.T) {
	root := filepath.Join(t.TempDir(), "assembleiarr")
	opts, _, errOut := bootstrapOpts(root)
	opts.Declarations = nil
	opts.Consumers = []string{"provision-example"}
	if err := Bootstrap(opts); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(errOut.String(), "no declaration given") {
		t.Fatalf("no warning: %q", errOut)
	}
	data, _ := os.ReadFile(filepath.Join(root, "lab", "targets", "vsphere-lab", "vars.sh"))
	if !strings.Contains(string(data), "No declaration was given") {
		t.Fatalf("vars.sh is not generic:\n%s", data)
	}
	if _, err := os.Stat(filepath.Join(root, "lab", "targets", "vsphere-lab", "hosts", "provision-example.sh")); err != nil {
		t.Fatal(err)
	}
	// A consumer without a declaration next to declared ones is warned about.
	root2 := filepath.Join(t.TempDir(), "assembleiarr")
	opts, _, errOut = bootstrapOpts(root2)
	opts.Consumers = []string{"provision-example"}
	if err := Bootstrap(opts); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(errOut.String(), "no declaration for consumer provision-example") {
		t.Fatalf("no warning: %q", errOut)
	}
}

func readPair(t *testing.T, root string) (priv, pub []byte) {
	t.Helper()
	dir := filepath.Join(root, "lab", "credentials", "automation")
	priv, err := os.ReadFile(filepath.Join(dir, AutomationPrivateKey))
	if err != nil {
		t.Fatal(err)
	}
	pub, err = os.ReadFile(filepath.Join(dir, AutomationPublicKey))
	if err != nil {
		t.Fatal(err)
	}
	return priv, pub
}

func assertPairMatches(t *testing.T, priv, pub []byte) {
	t.Helper()
	signer, err := ssh.ParsePrivateKey(priv)
	if err != nil {
		t.Fatalf("parse private key: %v", err)
	}
	parsed, comment, _, _, err := ssh.ParseAuthorizedKey(pub)
	if err != nil {
		t.Fatalf("parse public key: %v", err)
	}
	if !bytes.Equal(signer.PublicKey().Marshal(), parsed.Marshal()) {
		t.Fatal("public key does not match the private key")
	}
	if parsed.Type() != ssh.KeyAlgoED25519 || comment != "assembleiarr-lab-automation" {
		t.Fatalf("type %s, comment %q", parsed.Type(), comment)
	}
}

func TestBootstrapAutomationKeyConcurrent(t *testing.T) {
	root := filepath.Join(t.TempDir(), "assembleiarr")
	const runs = 8
	errs := make([]error, runs)
	var wg sync.WaitGroup
	for i := 0; i < runs; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			opts, _, _ := bootstrapOpts(root)
			opts.AutomationKey = true
			errs[i] = Bootstrap(opts)
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
	}
	priv, pub := readPair(t, root)
	assertPairMatches(t, priv, pub)
	entries, _ := os.ReadDir(filepath.Join(root, "lab", "credentials"))
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), TempPrefix) {
			t.Fatalf("temporary directory left: %s", e.Name())
		}
	}
	info, _ := os.Stat(filepath.Join(root, "lab", "credentials", "automation", AutomationPrivateKey))
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("private key mode %o", info.Mode().Perm())
	}
}

func TestBootstrapPartialPairIsAnError(t *testing.T) {
	root := filepath.Join(t.TempDir(), "assembleiarr")
	dir := filepath.Join(root, "lab", "credentials", "automation")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, AutomationPublicKey), []byte("ssh-ed25519 AAAA x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	opts, _, _ := bootstrapOpts(root)
	opts.AutomationKey = true
	err := Bootstrap(opts)
	if ExitCode(err) != ExitProblem || !strings.Contains(err.Error(), "incomplete automation key pair, missing or not a regular file: ssh_ed25519;") {
		t.Fatalf("want a partial pair problem, got %v", err)
	}
	if _, err := os.Lstat(filepath.Join(dir, AutomationPrivateKey)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("private key created in an existing automation directory")
	}
	// A non-regular entry in place of a key file is not a key file.
	if err := os.Mkdir(filepath.Join(dir, AutomationPrivateKey), 0o700); err != nil {
		t.Fatal(err)
	}
	opts, _, _ = bootstrapOpts(root)
	opts.AutomationKey = true
	if err := Bootstrap(opts); ExitCode(err) != ExitProblem || !strings.Contains(err.Error(), "ssh_ed25519;") {
		t.Fatalf("directory as private key: want a partial pair problem, got %v", err)
	}
}

func TestBootstrapAutomationKeyImport(t *testing.T) {
	first := filepath.Join(t.TempDir(), "assembleiarr")
	opts, _, _ := bootstrapOpts(first)
	opts.AutomationKey = true
	if err := Bootstrap(opts); err != nil {
		t.Fatal(err)
	}
	priv, pub := readPair(t, first)

	// The private key is kept byte for byte, with or without a final newline.
	for name, input := range map[string][]byte{
		"as generated":     priv,
		"no final newline": bytes.TrimRight(priv, "\n"),
		"extra newline":    append(append([]byte{}, priv...), '\n'),
	} {
		second := filepath.Join(t.TempDir(), "assembleiarr")
		opts, _, _ = bootstrapOpts(second)
		opts.ImportKey = bytes.NewReader(input)
		if err := Bootstrap(opts); err != nil {
			t.Fatalf("%s: import: %v", name, err)
		}
		priv2, pub2 := readPair(t, second)
		if !bytes.Equal(input, priv2) || !bytes.Equal(pub, pub2) {
			t.Fatalf("%s: imported pair differs from the input", name)
		}
		assertPairMatches(t, priv2, pub2)
	}
}

func TestBootstrapAutomationKeyImportRefusals(t *testing.T) {
	_, edPriv, _ := ed25519.GenerateKey(rand.Reader)
	withPass, err := ssh.MarshalPrivateKeyWithPassphrase(edPriv, "c", []byte("passphrase"))
	if err != nil {
		t.Fatal(err)
	}
	ecKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	ecBlock, err := ssh.MarshalPrivateKey(ecKey, "c")
	if err != nil {
		t.Fatal(err)
	}
	plain, _ := ssh.MarshalPrivateKey(edPriv, "c")
	twoKeys := append(pem.EncodeToMemory(plain), pem.EncodeToMemory(plain)...)
	cases := map[string][]byte{
		"passphrase":  pem.EncodeToMemory(withPass),
		"not ed25519": pem.EncodeToMemory(ecBlock),
		"garbage":     []byte("not a key"),
		"two keys":    twoKeys,
		"preamble":    append([]byte("some text\n"), pem.EncodeToMemory(plain)...),
		"empty":       nil,
		"too large":   bytes.Repeat([]byte("A"), maxKeyInput+1),
	}
	for name, input := range cases {
		root := filepath.Join(t.TempDir(), "assembleiarr")
		opts, _, _ := bootstrapOpts(root)
		opts.ImportKey = bytes.NewReader(input)
		err := Bootstrap(opts)
		if ExitCode(err) != ExitUsage {
			t.Errorf("%s: want usage error, got %v", name, err)
		}
		if err != nil && strings.Contains(err.Error(), "PRIVATE KEY-----\n") {
			t.Errorf("%s: error holds key material", name)
		}
		if _, err := os.Lstat(root); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s: something was written", name)
		}
	}
}
