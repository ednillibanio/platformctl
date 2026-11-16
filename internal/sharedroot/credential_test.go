//go:build linux || darwin

package sharedroot

import (
	"bytes"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// countingReader records how many times it was read.
type countingReader struct {
	r     *strings.Reader
	reads int
}

func (c *countingReader) Read(p []byte) (int, error) {
	c.reads++
	return c.r.Read(p)
}

func credentialOpts(root string, specs ...string) (CredentialOptions, *bytes.Buffer) {
	var out bytes.Buffer
	return CredentialOptions{
		Root: root, Env: "lab", Consumer: "provision-example", Specs: specs, Out: &out,
	}, &out
}

func readCredential(t *testing.T, path string) string {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 || !info.Mode().IsRegular() {
		t.Fatalf("%s: mode %v, want regular 0600", path, info.Mode())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.HasSuffix(data, []byte("\n")) {
		t.Fatalf("%s ends with a newline", path)
	}
	return string(data)
}

func TestCredentialFormats(t *testing.T) {
	root := filepath.Join(t.TempDir(), "assembleiarr")
	opts, out := credentialOpts(root, "DB_USER=app-user", "DB_PASS", "API_TOKEN=-")
	opts.Stdin = strings.NewReader("s3cret-from-stdin\n")
	if err := Credential(opts); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "lab", "credentials", "provision-example")
	if got := readCredential(t, filepath.Join(dir, "DB_USER")); got != "app-user" {
		t.Errorf("DB_USER = %q", got)
	}
	if got := readCredential(t, filepath.Join(dir, "API_TOKEN")); got != "s3cret-from-stdin" {
		t.Errorf("API_TOKEN = %q", got)
	}
	random := readCredential(t, filepath.Join(dir, "DB_PASS"))
	raw, err := base64.StdEncoding.DecodeString(random)
	if err != nil || len(raw) != DefaultCredentialLength || len(random) != 44 {
		t.Errorf("DB_PASS %q is not base64 of %d bytes: %v", random, DefaultCredentialLength, err)
	}
	for _, d := range []string{filepath.Join(root, "lab"), filepath.Join(root, "lab", "credentials"), dir} {
		if info, err := os.Lstat(d); err != nil || info.Mode().Perm() != 0o700 {
			t.Errorf("%s: %v %v, want 0700", d, info, err)
		}
	}
	for _, secret := range []string{"app-user", "s3cret-from-stdin", random} {
		if strings.Contains(out.String(), secret) {
			t.Errorf("output holds a value: %s", out.String())
		}
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 3 {
		t.Errorf("want 3 files, got %d (temporary file left?)", len(entries))
	}
}

func TestCredentialLength(t *testing.T) {
	root := t.TempDir()
	opts, _ := credentialOpts(root, "KEY")
	opts.Length = 48
	if err := Credential(opts); err != nil {
		t.Fatal(err)
	}
	value := readCredential(t, filepath.Join(root, "lab", "credentials", "provision-example", "KEY"))
	if raw, err := base64.StdEncoding.DecodeString(value); err != nil || len(raw) != 48 {
		t.Fatalf("want 48 random bytes, got %q (%v)", value, err)
	}
}

func TestCredentialNeverOverwrites(t *testing.T) {
	root := t.TempDir()
	opts, _ := credentialOpts(root, "DB_PASS", "DB_USER=first")
	if err := Credential(opts); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "lab", "credentials", "provision-example")
	ageTree(t, root)
	before := snapshot(t, root)
	stdin := &countingReader{r: strings.NewReader("other\n")}
	opts, out := credentialOpts(root, "DB_PASS", "DB_USER=second")
	opts.Stdin = stdin
	if err := Credential(opts); err != nil {
		t.Fatal(err)
	}
	assertSameSnapshot(t, before, snapshot(t, root))
	if strings.Count(out.String(), "exists") != 2 {
		t.Errorf("want two skips reported, got:\n%s", out.String())
	}
	if got := readCredential(t, filepath.Join(dir, "DB_USER")); got != "first" {
		t.Errorf("DB_USER overwritten: %q", got)
	}
	if stdin.reads != 0 {
		t.Errorf("standard input read without NAME=-")
	}
}

func TestCredentialStdinOnlyWhenWritten(t *testing.T) {
	root := t.TempDir()
	opts, _ := credentialOpts(root, "PASS=-")
	opts.Stdin = strings.NewReader("one")
	if err := Credential(opts); err != nil {
		t.Fatal(err)
	}
	stdin := &countingReader{r: strings.NewReader("two")}
	opts, out := credentialOpts(root, "PASS=-")
	opts.Stdin = stdin
	if err := Credential(opts); err != nil {
		t.Fatal(err)
	}
	if stdin.reads != 0 {
		t.Error("standard input read for an existing file")
	}
	if !strings.Contains(out.String(), "exists") {
		t.Errorf("skip not reported: %s", out.String())
	}
	if got := readCredential(t, filepath.Join(root, "lab", "credentials", "provision-example", "PASS")); got != "one" {
		t.Errorf("PASS = %q", got)
	}
}

func TestCredentialStdinValues(t *testing.T) {
	cases := map[string]string{
		"plain":           "value",
		"newline dropped": "value\n",
		"crlf dropped":    "value\r\n",
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			opts, _ := credentialOpts(root, "PASS=-")
			opts.Stdin = strings.NewReader(input)
			if err := Credential(opts); err != nil {
				t.Fatal(err)
			}
			if got := readCredential(t, filepath.Join(root, "lab", "credentials", "provision-example", "PASS")); got != "value" {
				t.Errorf("PASS = %q", got)
			}
		})
	}
	bad := map[string]string{
		"empty":        "",
		"only newline": "\n",
		"two lines":    "a\nb\n",
		"lone cr":      "value\r",
		"two endings":  "value\n\n",
		"crlf then lf": "value\r\n\n",
		"lf then crlf": "value\n\r\n",
		"too long":     strings.Repeat("x", maxStdinValue+1),
	}
	for name, input := range bad {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			opts, _ := credentialOpts(root, "PASS=-")
			opts.Stdin = strings.NewReader(input)
			if err := Credential(opts); ExitCode(err) != ExitUsage {
				t.Fatalf("want usage error, got %v", err)
			}
			if _, err := os.Lstat(filepath.Join(root, "lab", "credentials", "provision-example", "PASS")); !os.IsNotExist(err) {
				t.Error("a rejected value was written")
			}
		})
	}
}

func TestCredentialTargetIdentity(t *testing.T) {
	root := t.TempDir()
	var out bytes.Buffer
	opts := CredentialOptions{
		Root: root, Env: "lab", Target: "vsphere-lab", Identity: "vcenter-access-admin",
		Specs: []string{"GOVC_USERNAME=svc", "GOVC_PASSWORD=-"}, Stdin: strings.NewReader("pw"), Out: &out,
	}
	if err := Credential(opts); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "lab", "credentials", "target-vsphere-lab", "vcenter-access-admin")
	if got := readCredential(t, filepath.Join(dir, "GOVC_PASSWORD")); got != "pw" {
		t.Errorf("GOVC_PASSWORD = %q", got)
	}
}

func TestCredentialUsageErrors(t *testing.T) {
	cases := map[string]CredentialOptions{
		"no destination":         {Env: "lab", Specs: []string{"A"}},
		"unknown env":            {Env: "dev", Consumer: "provision-example", Specs: []string{"A"}},
		"bad consumer":           {Env: "lab", Consumer: "automation", Specs: []string{"A"}},
		"consumer and target":    {Env: "lab", Consumer: "provision-example", Target: "t", Identity: "vcenter-access-admin", Specs: []string{"A"}},
		"target without id":      {Env: "lab", Target: "vsphere-lab", Specs: []string{"A"}},
		"identity without tgt":   {Env: "lab", Identity: "vcenter-access-admin", Specs: []string{"A"}},
		"reserved identity":      {Env: "lab", Target: "vsphere-lab", Identity: "provision-semaphoreui", Specs: []string{"A"}},
		"deprecated identity":    {Env: "lab", Target: "vsphere-lab", Identity: "terraform-provisioner", Specs: []string{"A"}},
		"unknown identity":       {Env: "lab", Target: "vsphere-lab", Identity: "root", Specs: []string{"A"}},
		"traversal target":       {Env: "lab", Target: "../x", Identity: "vcenter-access-admin", Specs: []string{"A"}},
		"no names":               {Env: "lab", Consumer: "provision-example"},
		"bad name":               {Env: "lab", Consumer: "provision-example", Specs: []string{"db_pass"}},
		"path name":              {Env: "lab", Consumer: "provision-example", Specs: []string{"../A"}},
		"empty name":             {Env: "lab", Consumer: "provision-example", Specs: []string{"=x"}},
		"duplicate name":         {Env: "lab", Consumer: "provision-example", Specs: []string{"A", "A=x"}},
		"two stdin":              {Env: "lab", Consumer: "provision-example", Specs: []string{"A=-", "B=-"}},
		"empty literal":          {Env: "lab", Consumer: "provision-example", Specs: []string{"A="}},
		"literal with newline":   {Env: "lab", Consumer: "provision-example", Specs: []string{"A=x\ny"}},
		"length too large":       {Env: "lab", Consumer: "provision-example", Specs: []string{"A"}, Length: MaxCredentialLength + 1},
		"negative length":        {Env: "lab", Consumer: "provision-example", Specs: []string{"A"}, Length: -1},
		"valid then invalid arg": {Env: "lab", Consumer: "provision-example", Specs: []string{"A", "b"}},
	}
	for name, opts := range cases {
		t.Run(name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "assembleiarr")
			opts.Root = root
			opts.Out = &bytes.Buffer{}
			stdin := &countingReader{r: strings.NewReader("x")}
			opts.Stdin = stdin
			if err := Credential(opts); ExitCode(err) != ExitUsage {
				t.Fatalf("want usage error, got %v", err)
			}
			if _, err := os.Lstat(root); !os.IsNotExist(err) {
				t.Error("a usage error wrote something")
			}
			if stdin.reads != 0 {
				t.Error("a usage error read standard input")
			}
		})
	}
}

func TestCredentialDryRun(t *testing.T) {
	root := filepath.Join(t.TempDir(), "assembleiarr")
	stdin := &countingReader{r: strings.NewReader("x")}
	opts, out := credentialOpts(root, "A", "B=zq9value", "C=-")
	opts.DryRun, opts.Stdin = true, stdin
	if err := Credential(opts); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(root); !os.IsNotExist(err) {
		t.Fatal("dry run wrote the root")
	}
	if stdin.reads != 0 || strings.Contains(out.String(), "zq9value") {
		t.Errorf("dry run read stdin or printed a value:\n%s", out.String())
	}
	if strings.Count(out.String(), "would create") != 7 { // root, lab, credentials, consumer dir, 3 files
		t.Errorf("unexpected dry-run output:\n%s", out.String())
	}

	// Existing tree: the existing file is reported, the new one would be created.
	opts, _ = credentialOpts(root, "A")
	if err := Credential(opts); err != nil {
		t.Fatal(err)
	}
	ageTree(t, root)
	before := snapshot(t, root)
	opts, out = credentialOpts(root, "A", "B")
	opts.DryRun = true
	if err := Credential(opts); err != nil {
		t.Fatal(err)
	}
	assertSameSnapshot(t, before, snapshot(t, root))
	if !strings.Contains(out.String(), "exists") || strings.Count(out.String(), "would create") != 1 {
		t.Errorf("unexpected dry-run output:\n%s", out.String())
	}
}

func TestCredentialRefusesSymlinkAndDirectory(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "lab", "credentials", "provision-example")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "LINKED")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "ADIR"), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"LINKED", "ADIR"} {
		opts, _ := credentialOpts(root, name)
		if err := Credential(opts); err == nil || ExitCode(err) != ExitProblem {
			t.Errorf("%s: want problem, got %v", name, err)
		}
	}
	if data, _ := os.ReadFile(outside); string(data) != "keep" {
		t.Error("file behind the symbolic link changed")
	}

	// A symbolic link as the destination directory is refused too.
	root2 := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root2, "lab", "credentials"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(root2, "lab", "credentials", "provision-example")); err != nil {
		t.Fatal(err)
	}
	opts, _ := credentialOpts(root2, "A")
	if err := Credential(opts); !errors.Is(err, ErrSymlink) {
		t.Errorf("want ErrSymlink, got %v", err)
	}
}

func TestCredentialConcurrent(t *testing.T) {
	root := t.TempDir()
	const runs = 8
	var wg sync.WaitGroup
	errs := make([]error, runs)
	outs := make([]*bytes.Buffer, runs)
	for i := 0; i < runs; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			opts, out := credentialOpts(root, "SHARED")
			outs[i] = out
			errs[i] = Credential(opts)
		}(i)
	}
	wg.Wait()
	created := 0
	for i := range errs {
		if errs[i] != nil {
			t.Fatalf("run %d: %v", i, errs[i])
		}
		created += strings.Count(outs[i].String(), "created      "+filepath.Join(root, "lab", "credentials", "provision-example", "SHARED"))
	}
	if created != 1 {
		t.Errorf("want exactly one run to create the file, got %d", created)
	}
	dir := filepath.Join(root, "lab", "credentials", "provision-example")
	readCredential(t, filepath.Join(dir, "SHARED"))
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("want one file, got %d (temporary file left?)", len(entries))
	}
}

func TestCredentialInvalidStdinWritesNothing(t *testing.T) {
	// Missing root: nothing at all is created.
	root := filepath.Join(t.TempDir(), "assembleiarr")
	opts, _ := credentialOpts(root, "FIRST=lit", "SECOND", "PASS=-")
	opts.Stdin = strings.NewReader("a\nb")
	if err := Credential(opts); ExitCode(err) != ExitUsage {
		t.Fatalf("want usage error, got %v", err)
	}
	if _, err := os.Lstat(root); !os.IsNotExist(err) {
		t.Fatal("an invalid standard input value left the root created")
	}

	// Existing tree: unchanged.
	root = t.TempDir()
	opts, _ = credentialOpts(root, "OLD")
	if err := Credential(opts); err != nil {
		t.Fatal(err)
	}
	ageTree(t, root)
	before := snapshot(t, root)
	opts, _ = credentialOpts(root, "FIRST=lit", "SECOND", "PASS=-")
	opts.Stdin = strings.NewReader("")
	if err := Credential(opts); ExitCode(err) != ExitUsage {
		t.Fatalf("want usage error, got %v", err)
	}
	assertSameSnapshot(t, before, snapshot(t, root))
}

func TestCredentialStdinSizeLimit(t *testing.T) {
	full := strings.Repeat("x", maxStdinValue)
	for name, input := range map[string]string{"no ending": full, "lf": full + "\n", "crlf": full + "\r\n"} {
		t.Run("accepted "+name, func(t *testing.T) {
			root := t.TempDir()
			opts, _ := credentialOpts(root, "PASS=-")
			opts.Stdin = strings.NewReader(input)
			if err := Credential(opts); err != nil {
				t.Fatal(err)
			}
			if got := readCredential(t, filepath.Join(root, "lab", "credentials", "provision-example", "PASS")); got != full {
				t.Errorf("value of %d bytes, want %d", len(got), len(full))
			}
		})
	}
	for name, input := range map[string]string{
		"one byte over":         full + "x",
		"one byte over crlf":    full + "x\r\n",
		"ending then more data": full + "\r\nx",
	} {
		t.Run("rejected "+name, func(t *testing.T) {
			opts, _ := credentialOpts(t.TempDir(), "PASS=-")
			opts.Stdin = strings.NewReader(input)
			if err := Credential(opts); ExitCode(err) != ExitUsage {
				t.Fatalf("want usage error, got %v", err)
			}
		})
	}
}
