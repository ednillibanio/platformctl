//go:build linux || darwin

package sharedroot

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
)

func openTestTree(t *testing.T) (*Tree, string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "assembleiarr")
	tree, err := OpenTree(root, true)
	if err != nil {
		t.Fatalf("OpenTree: %v", err)
	}
	t.Cleanup(func() { _ = tree.Close() })
	return tree, root
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("lstat %s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%s: mode %o, want %o", path, got, want)
	}
}

func assertNoTemp(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), TempPrefix) {
			t.Fatalf("temporary entry left behind: %s", filepath.Join(dir, entry.Name()))
		}
	}
}

// withUmask runs fn with the process umask set to mask. Tests that use it are
// not parallel because the umask is process wide.
func withUmask(t *testing.T, mask int, fn func()) {
	t.Helper()
	old := syscall.Umask(mask)
	defer syscall.Umask(old)
	fn()
}

func TestOpenTreeCreatesRootPrivate(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "a", "b", "assembleiarr")
	withUmask(t, 0o077, func() {
		tree, err := OpenTree(root, true)
		if err != nil {
			t.Fatalf("OpenTree: %v", err)
		}
		_ = tree.Close()
	})
	assertMode(t, filepath.Join(base, "a"), 0o700)
	assertMode(t, filepath.Join(base, "a", "b"), 0o700)
	assertMode(t, root, 0o700)
}

func TestOwnerStrippingUmaskFailsCleanly(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses directory permissions; the umask cannot lock it out")
	}
	tree, root := openTestTree(t)
	newRoot := filepath.Join(t.TempDir(), "new")
	withUmask(t, 0o777, func() {
		if _, err := tree.EnsureDir("lab"); !errors.Is(err, ErrUmask) {
			t.Fatalf("EnsureDir: want ErrUmask, got %v", err)
		}
		if err := tree.PublishDir(nil, "automation", pairFiles("x")); !errors.Is(err, ErrUmask) {
			t.Fatalf("PublishDir: want ErrUmask, got %v", err)
		}
		if _, err := OpenTree(newRoot, true); !errors.Is(err, ErrUmask) {
			t.Fatalf("OpenTree: want ErrUmask, got %v", err)
		}
	})
	if entries, _ := os.ReadDir(root); len(entries) != 0 {
		t.Fatalf("entries left behind: %v", entries)
	}
}

func TestOpenTreeLeavesExistingRootUnchanged(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o750); err != nil {
		t.Fatal(err)
	}
	tree, err := OpenTree(root, true)
	if err != nil {
		t.Fatalf("OpenTree: %v", err)
	}
	_ = tree.Close()
	assertMode(t, root, 0o750)
}

func TestOpenTreeMissingWithoutCreate(t *testing.T) {
	if _, err := OpenTree(filepath.Join(t.TempDir(), "missing"), false); err == nil {
		t.Fatal("want error for a missing root without create")
	}
}

func TestOpenTreeRejectsSymlinkAndRelative(t *testing.T) {
	base := t.TempDir()
	real := filepath.Join(base, "real")
	link := filepath.Join(base, "link")
	if err := os.Mkdir(real, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenTree(link, true); !errors.Is(err, ErrSymlink) {
		t.Fatalf("want ErrSymlink, got %v", err)
	}
	if _, err := OpenTree("relative", true); ExitCode(err) != ExitUsage {
		t.Fatalf("want usage error, got %v", err)
	}
}

func TestEnsureDirCreatesPrivateDirsIndependentOfUmask(t *testing.T) {
	tree, root := openTestTree(t)
	var created []string
	var err error
	withUmask(t, 0o077, func() {
		created, err = tree.EnsureDir("lab", "targets", "vsphere-lab", "hosts")
	})
	if err != nil {
		t.Fatalf("EnsureDir: %v", err)
	}
	want := []string{"lab", "lab/targets", "lab/targets/vsphere-lab", "lab/targets/vsphere-lab/hosts"}
	if fmt.Sprint(created) != fmt.Sprint(want) {
		t.Fatalf("created %v, want %v", created, want)
	}
	for _, rel := range want {
		assertMode(t, filepath.Join(root, rel), 0o700)
	}
	again, err := tree.EnsureDir("lab", "targets", "vsphere-lab", "hosts")
	if err != nil || len(again) != 0 {
		t.Fatalf("second EnsureDir: created %v, err %v; want nothing", again, err)
	}
}

func TestEnsureDirLeavesExistingDirUnchanged(t *testing.T) {
	tree, root := openTestTree(t)
	if err := os.Mkdir(filepath.Join(root, "lab"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(root, "lab"), 0o755); err != nil {
		t.Fatal(err)
	}
	created, err := tree.EnsureDir("lab", "credentials")
	if err != nil {
		t.Fatalf("EnsureDir: %v", err)
	}
	if fmt.Sprint(created) != "[lab/credentials]" {
		t.Fatalf("created %v", created)
	}
	assertMode(t, filepath.Join(root, "lab"), 0o755)
}

func TestEnsureDirRejectsSymlinkComponent(t *testing.T) {
	tree, root := openTestTree(t)
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "lab")); err != nil {
		t.Fatal(err)
	}
	_, err := tree.EnsureDir("lab", "targets")
	if !errors.Is(err, ErrSymlink) {
		t.Fatalf("want ErrSymlink, got %v", err)
	}
	if entries, _ := os.ReadDir(outside); len(entries) != 0 {
		t.Fatalf("wrote through the symlink: %v", entries)
	}
}

func TestEnsureDirRejectsFileComponent(t *testing.T) {
	tree, root := openTestTree(t)
	if err := os.WriteFile(filepath.Join(root, "lab"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := tree.EnsureDir("lab", "targets"); !errors.Is(err, ErrNotDirectory) {
		t.Fatalf("want ErrNotDirectory, got %v", err)
	}
}

func TestInvalidComponentsAreUsageErrors(t *testing.T) {
	tree, _ := openTestTree(t)
	for _, bad := range []string{"", ".", "..", "a/b", "a\x00b"} {
		if _, err := tree.EnsureDir("lab", bad); ExitCode(err) != ExitUsage {
			t.Errorf("EnsureDir %q: want usage error, got %v", bad, err)
		}
		if err := tree.PublishFile(nil, bad, nil, 0o600); ExitCode(err) != ExitUsage {
			t.Errorf("PublishFile %q: want usage error, got %v", bad, err)
		}
		if err := tree.PublishDir(nil, bad, nil); ExitCode(err) != ExitUsage {
			t.Errorf("PublishDir %q: want usage error, got %v", bad, err)
		}
	}
}

func TestPublishFileModesIndependentOfUmask(t *testing.T) {
	tree, root := openTestTree(t)
	if _, err := tree.EnsureDir("lab"); err != nil {
		t.Fatal(err)
	}
	withUmask(t, 0o077, func() {
		if err := tree.PublishFile([]string{"lab"}, "vars.sh", []byte("# KEY=\n"), 0o644); err != nil {
			t.Fatalf("PublishFile 0644: %v", err)
		}
		if err := tree.PublishFile([]string{"lab"}, "SECRET", []byte("s3cret"), 0o600); err != nil {
			t.Fatalf("PublishFile 0600: %v", err)
		}
	})
	assertMode(t, filepath.Join(root, "lab", "vars.sh"), 0o644)
	assertMode(t, filepath.Join(root, "lab", "SECRET"), 0o600)
	got, err := os.ReadFile(filepath.Join(root, "lab", "SECRET"))
	if err != nil || string(got) != "s3cret" {
		t.Fatalf("content %q, err %v", got, err)
	}
	assertNoTemp(t, filepath.Join(root, "lab"))
}

func TestPublishFileNeverChangesExisting(t *testing.T) {
	tree, root := openTestTree(t)
	if _, err := tree.EnsureDir("lab"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "lab", "vars.sh")
	if err := os.WriteFile(path, []byte("GOVC_URL=https://operator\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	err := tree.PublishFile([]string{"lab"}, "vars.sh", []byte("# GOVC_URL=\n"), 0o644)
	if !errors.Is(err, ErrExists) {
		t.Fatalf("want ErrExists, got %v", err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "GOVC_URL=https://operator\n" {
		t.Fatalf("existing file changed: %q", got)
	}
	assertMode(t, path, 0o640)
	assertNoTemp(t, filepath.Join(root, "lab"))
}

func TestPublishFileRefusesSymlinks(t *testing.T) {
	tree, root := openTestTree(t)
	outside := t.TempDir()
	if _, err := tree.EnsureDir("lab"); err != nil {
		t.Fatal(err)
	}
	// A symbolic link at the destination is an existing entry: never replaced
	// and never followed.
	target := filepath.Join(outside, "victim")
	if err := os.Symlink(target, filepath.Join(root, "lab", "vars.sh")); err != nil {
		t.Fatal(err)
	}
	if err := tree.PublishFile([]string{"lab"}, "vars.sh", []byte("x"), 0o644); !errors.Is(err, ErrExists) {
		t.Fatalf("want ErrExists for a symlink destination, got %v", err)
	}
	if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("wrote through the destination symlink: %v", err)
	}
	// A symbolic link as a directory component is refused.
	if err := os.Symlink(outside, filepath.Join(root, "lab", "credentials")); err != nil {
		t.Fatal(err)
	}
	if err := tree.PublishFile([]string{"lab", "credentials"}, "SECRET", []byte("x"), 0o600); !errors.Is(err, ErrSymlink) {
		t.Fatalf("want ErrSymlink, got %v", err)
	}
	if entries, _ := os.ReadDir(outside); len(entries) != 0 {
		t.Fatalf("wrote through the directory symlink: %v", entries)
	}
}

func TestPublishFileMissingDirectory(t *testing.T) {
	tree, _ := openTestTree(t)
	if err := tree.PublishFile([]string{"lab"}, "vars.sh", nil, 0o644); err == nil {
		t.Fatal("want error for a missing directory")
	}
}

func TestPublishFileConcurrentOnlyOneWins(t *testing.T) {
	tree, root := openTestTree(t)
	if _, err := tree.EnsureDir("lab"); err != nil {
		t.Fatal(err)
	}
	const runs = 16
	errs := make([]error, runs)
	var wg sync.WaitGroup
	for i := 0; i < runs; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			content := bytes.Repeat([]byte{byte('a' + i)}, 4096)
			errs[i] = tree.PublishFile([]string{"lab"}, "SECRET", content, 0o600)
		}(i)
	}
	wg.Wait()
	winner := -1
	for i, err := range errs {
		switch {
		case err == nil:
			if winner >= 0 {
				t.Fatalf("runs %d and %d both published", winner, i)
			}
			winner = i
		case !errors.Is(err, ErrExists):
			t.Fatalf("run %d: unexpected error %v", i, err)
		}
	}
	if winner < 0 {
		t.Fatal("no run published")
	}
	got, _ := os.ReadFile(filepath.Join(root, "lab", "SECRET"))
	if !bytes.Equal(got, bytes.Repeat([]byte{byte('a' + winner)}, 4096)) {
		t.Fatal("published content is not the winner's whole content")
	}
	assertNoTemp(t, filepath.Join(root, "lab"))
}

func pairFiles(tag string) []File {
	return []File{
		{Name: "ssh_ed25519", Content: []byte("private-" + tag), Mode: 0o600},
		{Name: "ssh_ed25519.pub", Content: []byte("public-" + tag), Mode: 0o644},
	}
}

func TestPublishDirPublishesWholeUnit(t *testing.T) {
	tree, root := openTestTree(t)
	if _, err := tree.EnsureDir("lab", "credentials"); err != nil {
		t.Fatal(err)
	}
	withUmask(t, 0o077, func() {
		if err := tree.PublishDir([]string{"lab", "credentials"}, "automation", pairFiles("x")); err != nil {
			t.Fatalf("PublishDir: %v", err)
		}
	})
	dir := filepath.Join(root, "lab", "credentials", "automation")
	assertMode(t, dir, 0o700)
	assertMode(t, filepath.Join(dir, "ssh_ed25519"), 0o600)
	assertMode(t, filepath.Join(dir, "ssh_ed25519.pub"), 0o644)
	assertNoTemp(t, filepath.Join(root, "lab", "credentials"))
}

func TestPublishDirNeverChangesExisting(t *testing.T) {
	tree, root := openTestTree(t)
	dir := filepath.Join(root, "lab", "credentials", "automation")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	// An existing directory, empty or not, is never replaced.
	err := tree.PublishDir([]string{"lab", "credentials"}, "automation", pairFiles("x"))
	if !errors.Is(err, ErrExists) {
		t.Fatalf("empty existing dir: want ErrExists, got %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ssh_ed25519.pub"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	err = tree.PublishDir([]string{"lab", "credentials"}, "automation", pairFiles("y"))
	if !errors.Is(err, ErrExists) {
		t.Fatalf("partial existing dir: want ErrExists, got %v", err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 || entries[0].Name() != "ssh_ed25519.pub" {
		t.Fatalf("existing dir changed: %v", entries)
	}
	assertNoTemp(t, filepath.Join(root, "lab", "credentials"))
}

func TestPublishDirFailureRemovesTemp(t *testing.T) {
	tree, root := openTestTree(t)
	if _, err := tree.EnsureDir("lab", "credentials"); err != nil {
		t.Fatal(err)
	}
	// A duplicate file name fails the second write inside the temporary dir.
	files := []File{{Name: "a", Content: []byte("1"), Mode: 0o600}, {Name: "a", Content: []byte("2"), Mode: 0o600}}
	if err := tree.PublishDir([]string{"lab", "credentials"}, "automation", files); err == nil {
		t.Fatal("want error for a failed write")
	}
	if _, err := os.Lstat(filepath.Join(root, "lab", "credentials", "automation")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("destination created by a failed run: %v", err)
	}
	assertNoTemp(t, filepath.Join(root, "lab", "credentials"))
}

func TestPublishDirConcurrentOnlyOneWins(t *testing.T) {
	tree, root := openTestTree(t)
	if _, err := tree.EnsureDir("lab", "credentials"); err != nil {
		t.Fatal(err)
	}
	const runs = 16
	errs := make([]error, runs)
	var wg sync.WaitGroup
	for i := 0; i < runs; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = tree.PublishDir([]string{"lab", "credentials"}, "automation", pairFiles(fmt.Sprint(i)))
		}(i)
	}
	wg.Wait()
	winner := -1
	for i, err := range errs {
		switch {
		case err == nil:
			if winner >= 0 {
				t.Fatalf("runs %d and %d both published", winner, i)
			}
			winner = i
		case !errors.Is(err, ErrExists):
			t.Fatalf("run %d: unexpected error %v", i, err)
		}
	}
	if winner < 0 {
		t.Fatal("no run published")
	}
	dir := filepath.Join(root, "lab", "credentials", "automation")
	for _, file := range pairFiles(fmt.Sprint(winner)) {
		got, err := os.ReadFile(filepath.Join(dir, file.Name))
		if err != nil || !bytes.Equal(got, file.Content) {
			t.Fatalf("%s: got %q, err %v; pair is not the winner's whole pair", file.Name, got, err)
		}
	}
	assertNoTemp(t, filepath.Join(root, "lab", "credentials"))
}
