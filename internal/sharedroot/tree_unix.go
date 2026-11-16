//go:build linux || darwin

package sharedroot

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/sys/unix"
)

// Tree is an open shared root. Every operation walks the tree from the root
// directory descriptor one component at a time with O_NOFOLLOW and acts
// relative to directory descriptors, so a symbolic link inside the tree is
// refused even when it appears after a check.
type Tree struct {
	root string
	fd   int
}

// OpenTree opens the shared root. When create is true, a missing root is
// created with mode 0700 (and its missing parents with 0700); an existing root
// is never changed. A root that is a symbolic link is refused.
func OpenTree(root string, create bool) (*Tree, error) {
	if !filepath.IsAbs(root) {
		return nil, Usagef("shared root must be an absolute path: %q", root)
	}
	root = filepath.Clean(root)
	info, err := os.Lstat(root)
	switch {
	case err == nil:
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("%w: %s", ErrSymlink, root)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("%w: %s", ErrNotDirectory, root)
		}
		fd, err := unix.Open(root, dirOpenFlags, 0)
		if err != nil {
			return nil, classifyOpenError(root, err)
		}
		return &Tree{root: root, fd: fd}, nil
	case errors.Is(err, os.ErrNotExist) && create:
		fd, err := createRoot(root)
		if err != nil {
			return nil, err
		}
		return &Tree{root: root, fd: fd}, nil
	default:
		return nil, fmt.Errorf("inspect shared root %s: %w", root, err)
	}
}

// createRoot creates the missing root and its missing parents one component
// at a time from the nearest existing ancestor, each with mode 0700 set on
// its own descriptor, and returns a descriptor of the root. The existing
// ancestor is outside the tree and may be reached through a symbolic link;
// no created component is.
func createRoot(root string) (int, error) {
	var missing []string
	base := root
	for {
		if _, err := os.Lstat(base); err == nil {
			break
		} else if !errors.Is(err, os.ErrNotExist) {
			return -1, fmt.Errorf("inspect %s: %w", base, err)
		}
		parent := filepath.Dir(base)
		if parent == base {
			return -1, fmt.Errorf("no existing ancestor of shared root %s", root)
		}
		missing = append([]string{filepath.Base(base)}, missing...)
		base = parent
	}
	fd, err := unix.Open(base, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return -1, fmt.Errorf("open %s: %w", base, err)
	}
	for _, component := range missing {
		next, _, err := openDirAt(fd, component, true)
		unix.Close(fd)
		base = filepath.Join(base, component)
		if err != nil {
			if errors.Is(err, ErrSymlink) || errors.Is(err, ErrNotDirectory) {
				return -1, fmt.Errorf("%w: %s", err, base)
			}
			return -1, fmt.Errorf("create %s: %w", base, err)
		}
		fd = next
	}
	return fd, nil
}

// Root returns the absolute path of the shared root.
func (t *Tree) Root() string { return t.root }

// Close releases the root directory descriptor.
func (t *Tree) Close() error { return unix.Close(t.fd) }

// EnsureDir creates every missing directory of dir (components relative to the
// root) with mode 0700, independent of the umask. Existing directories are
// never changed. It returns the slash-joined relative paths it created.
func (t *Tree) EnsureDir(dir ...string) ([]string, error) {
	fd, created, err := t.walk(dir, true)
	if err != nil {
		return created, err
	}
	return created, unix.Close(fd)
}

// PublishFile writes a new file name in dir (an existing directory, components
// relative to the root). The content is written under a temporary name with
// the final mode, flushed, and hard-linked to name only if name does not
// exist; the temporary name is always removed. An existing name is never
// changed and yields ErrExists.
func (t *Tree) PublishFile(dir []string, name string, content []byte, mode os.FileMode) error {
	if err := checkComponent(name); err != nil {
		return err
	}
	dirfd, _, err := t.walk(dir, false)
	if err != nil {
		return err
	}
	defer unix.Close(dirfd)
	tmp, err := createFileAt(dirfd, content, mode)
	if err != nil {
		return fmt.Errorf("write %s: %w", t.display(dir, name), err)
	}
	linkErr := unix.Linkat(dirfd, tmp, dirfd, name, 0)
	if err := unix.Unlinkat(dirfd, tmp, 0); err != nil && linkErr == nil {
		return fmt.Errorf("remove temporary file of %s: %w", t.display(dir, name), err)
	}
	if linkErr != nil {
		if errors.Is(linkErr, unix.EEXIST) {
			return fmt.Errorf("%w: %s", ErrExists, t.display(dir, name))
		}
		return fmt.Errorf("publish %s: %w", t.display(dir, name), linkErr)
	}
	return syncFD(dirfd)
}

// PublishDir creates the directory name in dir (an existing directory) holding
// exactly files, as one unit: the files are written into a new temporary
// directory (mode 0700) that is renamed to name only if name does not exist
// (renameat2 RENAME_NOREPLACE on Linux, renameatx_np RENAME_EXCL on macOS). A
// failed or losing run removes its temporary directory. An existing name is
// never changed and yields ErrExists.
func (t *Tree) PublishDir(dir []string, name string, files []File) error {
	if err := checkComponent(name); err != nil {
		return err
	}
	for _, file := range files {
		if err := checkComponent(file.Name); err != nil {
			return err
		}
	}
	dirfd, _, err := t.walk(dir, false)
	if err != nil {
		return err
	}
	defer unix.Close(dirfd)
	tmp, err := tempName()
	if err != nil {
		return err
	}
	tmpfd, err := mkdirOpenAt(dirfd, tmp)
	if err != nil {
		return fmt.Errorf("create temporary directory for %s: %w", t.display(dir, name), err)
	}
	written, err := populateDir(tmpfd, files)
	unix.Close(tmpfd)
	if err == nil {
		err = renameNoReplace(dirfd, tmp, name)
		if err == nil {
			return syncFD(dirfd)
		}
		if errors.Is(err, unix.EEXIST) || errors.Is(err, unix.ENOTEMPTY) {
			err = fmt.Errorf("%w: %s", ErrExists, t.display(dir, name))
		} else {
			err = fmt.Errorf("publish %s: %w", t.display(dir, name), err)
		}
	} else {
		err = fmt.Errorf("write %s: %w", t.display(dir, name), err)
	}
	if cleanupErr := removeTempDir(dirfd, tmp, written); cleanupErr != nil {
		return fmt.Errorf("%w (cleanup of temporary directory failed: %v)", err, cleanupErr)
	}
	return err
}

// walk opens dir component by component from the root and returns a
// descriptor of the last one. With create, missing components are created
// with mode 0700.
func (t *Tree) walk(dir []string, create bool) (int, []string, error) {
	fd, err := unix.Dup(t.fd)
	if err != nil {
		return -1, nil, err
	}
	var created []string
	for i, component := range dir {
		if err := checkComponent(component); err != nil {
			unix.Close(fd)
			return -1, created, err
		}
		rel := strings.Join(dir[:i+1], "/")
		next, made, err := openDirAt(fd, component, create)
		unix.Close(fd)
		if err != nil {
			return -1, created, t.wrapPathError(rel, err)
		}
		if made {
			created = append(created, rel)
		}
		fd = next
	}
	return fd, created, nil
}

func (t *Tree) display(dir []string, name string) string {
	return filepath.Join(append([]string{t.root}, append(dir, name)...)...)
}

func (t *Tree) wrapPathError(rel string, err error) error {
	path := filepath.Join(t.root, filepath.FromSlash(rel))
	switch {
	case errors.Is(err, ErrSymlink), errors.Is(err, ErrNotDirectory):
		return fmt.Errorf("%w: %s", err, path)
	case errors.Is(err, unix.ENOENT):
		return fmt.Errorf("%w: directory %s", os.ErrNotExist, path)
	}
	return fmt.Errorf("open %s: %w", path, err)
}

const dirOpenFlags = unix.O_RDONLY | unix.O_DIRECTORY | unix.O_NOFOLLOW | unix.O_CLOEXEC

func openDirAt(parent int, name string, create bool) (int, bool, error) {
	fd, err := unix.Openat(parent, name, dirOpenFlags, 0)
	if err == nil {
		return fd, false, nil
	}
	if !errors.Is(err, unix.ENOENT) || !create {
		return -1, false, classifyAt(parent, name, err)
	}
	fd, err = mkdirOpenAt(parent, name)
	if err == nil {
		return fd, true, nil
	}
	if !errors.Is(err, unix.EEXIST) {
		return -1, false, err
	}
	// Created concurrently by another run: open it, never change it.
	fd, err = unix.Openat(parent, name, dirOpenFlags, 0)
	if err != nil {
		return -1, false, classifyAt(parent, name, err)
	}
	return fd, false, nil
}

// classifyAt turns an open failure into ErrSymlink or ErrNotDirectory when the
// entry is a symbolic link or not a directory.
func classifyAt(parent int, name string, err error) error {
	var st unix.Stat_t
	if statErr := unix.Fstatat(parent, name, &st, unix.AT_SYMLINK_NOFOLLOW); statErr == nil {
		switch st.Mode & unix.S_IFMT {
		case unix.S_IFLNK:
			return ErrSymlink
		case unix.S_IFDIR:
			return err
		default:
			return ErrNotDirectory
		}
	}
	return err
}

func classifyOpenError(path string, err error) error {
	if errors.Is(err, unix.ELOOP) {
		return fmt.Errorf("%w: %s", ErrSymlink, path)
	}
	if errors.Is(err, unix.ENOTDIR) {
		return fmt.Errorf("%w: %s", ErrNotDirectory, path)
	}
	return fmt.Errorf("open shared root %s: %w", path, err)
}

// createFileAt writes content to a new temporary file in dirfd with mode set
// on the descriptor (so independent of the umask), flushes it and returns its
// name.
func createFileAt(dirfd int, content []byte, mode os.FileMode) (string, error) {
	name, err := tempName()
	if err != nil {
		return "", err
	}
	if err := writeNewFile(dirfd, name, content, mode); err != nil {
		_ = unix.Unlinkat(dirfd, name, 0)
		return "", err
	}
	return name, nil
}

func writeNewFile(dirfd int, name string, content []byte, mode os.FileMode) error {
	fd, err := unix.Openat(dirfd, name, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0o600)
	if err != nil {
		return err
	}
	file := os.NewFile(uintptr(fd), name)
	defer file.Close()
	if err := unix.Fchmod(fd, uint32(mode.Perm())); err != nil {
		return err
	}
	if _, err := file.Write(content); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	return file.Close()
}

func populateDir(fd int, files []File) ([]string, error) {
	var written []string
	for _, file := range files {
		if err := writeNewFile(fd, file.Name, file.Content, file.Mode); err != nil {
			_ = unix.Unlinkat(fd, file.Name, 0)
			return written, err
		}
		written = append(written, file.Name)
	}
	return written, syncFD(fd)
}

func removeTempDir(parent int, tmp string, files []string) error {
	fd, err := unix.Openat(parent, tmp, dirOpenFlags, 0)
	if err != nil {
		if errors.Is(err, unix.ENOENT) {
			return nil
		}
		return err
	}
	for _, name := range files {
		if err := unix.Unlinkat(fd, name, 0); err != nil && !errors.Is(err, unix.ENOENT) {
			unix.Close(fd)
			return err
		}
	}
	unix.Close(fd)
	return unix.Unlinkat(parent, tmp, unix.AT_REMOVEDIR)
}

func syncFD(fd int) error {
	if err := unix.Fsync(fd); err != nil && !errors.Is(err, unix.EINVAL) {
		return err
	}
	return nil
}

func tempName() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate temporary name: %w", err)
	}
	return TempPrefix + hex.EncodeToString(b[:]), nil
}

// checkComponent rejects a path component that is empty, ".", "..", or holds a
// separator or NUL. Callers validate names against the standard first; this
// is a second guard.
func checkComponent(name string) error {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\x00") {
		return Usagef("invalid path component %q", name)
	}
	return nil
}

// mkdirOpenAt creates name in parent with mode 0700, opens it with
// O_NOFOLLOW and sets mode 0700 on the descriptor, so the group and other bits
// of the umask never matter. A umask that removes owner bits (for example
// 0777) leaves a directory its owner cannot open; then the new, empty
// directory is removed and ErrUmask is returned. The process umask is never
// changed and no mode is set by path. An existing name yields EEXIST.
func mkdirOpenAt(parent int, name string) (int, error) {
	if err := unix.Mkdirat(parent, name, 0o700); err != nil {
		return -1, err
	}
	fd, err := unix.Openat(parent, name, dirOpenFlags, 0)
	if err != nil {
		if errors.Is(err, unix.EACCES) {
			_ = unix.Unlinkat(parent, name, unix.AT_REMOVEDIR)
			return -1, ErrUmask
		}
		return -1, classifyAt(parent, name, err)
	}
	if err := unix.Fchmod(fd, 0o700); err != nil {
		unix.Close(fd)
		return -1, err
	}
	return fd, nil
}

// Entry reports whether name exists in dir (components relative to the root)
// and whether it is a directory, without following a symbolic link. A missing
// dir counts as a missing entry.
func (t *Tree) Entry(dir []string, name string) (exists, isDir bool, err error) {
	if err := checkComponent(name); err != nil {
		return false, false, err
	}
	dirfd, _, err := t.walk(dir, false)
	if errors.Is(err, os.ErrNotExist) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	defer unix.Close(dirfd)
	var st unix.Stat_t
	if err := unix.Fstatat(dirfd, name, &st, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		if errors.Is(err, unix.ENOENT) {
			return false, false, nil
		}
		return false, false, fmt.Errorf("inspect %s: %w", t.display(dir, name), err)
	}
	switch st.Mode & unix.S_IFMT {
	case unix.S_IFLNK:
		return true, false, fmt.Errorf("%w: %s", ErrSymlink, t.display(dir, name))
	case unix.S_IFDIR:
		return true, true, nil
	}
	return true, false, nil
}

// maxReadSize bounds a shared file read; the files hold a few keys.
const maxReadSize = 1 << 20

// ReadFile reads the regular file name in dir without following a symbolic
// link. A missing file yields an error matching os.ErrNotExist.
func (t *Tree) ReadFile(dir []string, name string) ([]byte, error) {
	if err := checkComponent(name); err != nil {
		return nil, err
	}
	dirfd, _, err := t.walk(dir, false)
	if err != nil {
		return nil, err
	}
	defer unix.Close(dirfd)
	fd, err := unix.Openat(dirfd, name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		switch {
		case errors.Is(err, unix.ENOENT):
			return nil, fmt.Errorf("%w: %s", os.ErrNotExist, t.display(dir, name))
		case errors.Is(err, unix.ELOOP):
			return nil, fmt.Errorf("%w: %s", ErrSymlink, t.display(dir, name))
		}
		return nil, fmt.Errorf("open %s: %w", t.display(dir, name), err)
	}
	file := os.NewFile(uintptr(fd), name)
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file: %s", t.display(dir, name))
	}
	data, err := io.ReadAll(io.LimitReader(file, maxReadSize+1))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", t.display(dir, name), err)
	}
	if len(data) > maxReadSize {
		return nil, fmt.Errorf("file larger than %d bytes: %s", maxReadSize, t.display(dir, name))
	}
	return data, nil
}

// Lstat describes name in dir without following a symbolic link. A missing
// entry or directory yields an error matching os.ErrNotExist. With an empty
// name it describes the directory dir itself (the root for an empty dir).
func (t *Tree) Lstat(dir []string, name string) (EntryInfo, error) {
	if name == "" {
		fd, _, err := t.walk(dir, false)
		if err != nil {
			return EntryInfo{}, err
		}
		defer unix.Close(fd)
		var st unix.Stat_t
		if err := unix.Fstat(fd, &st); err != nil {
			return EntryInfo{}, err
		}
		base := ""
		if len(dir) > 0 {
			base = dir[len(dir)-1]
		}
		return entryInfo(base, &st), nil
	}
	if err := checkComponent(name); err != nil {
		return EntryInfo{}, err
	}
	dirfd, _, err := t.walk(dir, false)
	if err != nil {
		return EntryInfo{}, err
	}
	defer unix.Close(dirfd)
	var st unix.Stat_t
	if err := unix.Fstatat(dirfd, name, &st, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		if errors.Is(err, unix.ENOENT) {
			return EntryInfo{}, fmt.Errorf("%w: %s", os.ErrNotExist, t.display(dir, name))
		}
		return EntryInfo{}, fmt.Errorf("inspect %s: %w", t.display(dir, name), err)
	}
	return entryInfo(name, &st), nil
}

// List describes every entry of dir, sorted by name, without following a
// symbolic link.
func (t *Tree) List(dir []string) ([]EntryInfo, error) {
	dirfd, _, err := t.walk(dir, false)
	if err != nil {
		return nil, err
	}
	// The *os.File owns dirfd and closes it.
	file := os.NewFile(uintptr(dirfd), filepath.Join(append([]string{t.root}, dir...)...))
	defer file.Close()
	names, err := file.Readdirnames(-1)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", file.Name(), err)
	}
	sort.Strings(names)
	out := make([]EntryInfo, 0, len(names))
	for _, name := range names {
		var st unix.Stat_t
		if err := unix.Fstatat(dirfd, name, &st, unix.AT_SYMLINK_NOFOLLOW); err != nil {
			if errors.Is(err, unix.ENOENT) {
				continue // removed meanwhile
			}
			return nil, fmt.Errorf("inspect %s: %w", t.display(dir, name), err)
		}
		out = append(out, entryInfo(name, &st))
	}
	return out, nil
}

func entryInfo(name string, st *unix.Stat_t) EntryInfo {
	info := EntryInfo{Name: name, Perm: os.FileMode(st.Mode) & os.ModePerm, UID: int(st.Uid), Size: int64(st.Size)}
	switch st.Mode & unix.S_IFMT {
	case unix.S_IFDIR:
		info.Type = TypeDir
	case unix.S_IFREG:
		info.Type = TypeFile
	case unix.S_IFLNK:
		info.Type = TypeSymlink
	default:
		info.Type = TypeOther
	}
	if st.Mode&(unix.S_ISUID|unix.S_ISGID|unix.S_ISVTX) != 0 {
		info.Special = true
	}
	return info
}

func supported() bool { return true }
