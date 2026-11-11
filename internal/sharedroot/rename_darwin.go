//go:build darwin

package sharedroot

import "golang.org/x/sys/unix"

// renameNoReplace renames from to to inside dirfd and fails with EEXIST when
// to exists.
func renameNoReplace(dirfd int, from, to string) error {
	return unix.RenameatxNp(dirfd, from, dirfd, to, unix.RENAME_EXCL)
}
