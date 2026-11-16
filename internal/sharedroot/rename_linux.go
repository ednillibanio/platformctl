//go:build linux

package sharedroot

import "golang.org/x/sys/unix"

// renameNoReplace renames from to to inside dirfd and fails with EEXIST when
// to exists.
func renameNoReplace(dirfd int, from, to string) error {
	return unix.Renameat2(dirfd, from, dirfd, to, unix.RENAME_NOREPLACE)
}
