// Package sharedroot implements the provision-* shared root contract of the
// platform-catalog standard docs/standards/provision-shared-root-standard.md:
// root resolution, name validation and safe, no-clobber publication of files
// and directories under <root>/<env>/.
package sharedroot

import (
	"errors"
	"fmt"
)

// Exit codes of the shared-root commands.
const (
	ExitOK      = 0
	ExitProblem = 1
	ExitUsage   = 2
)

var (
	// ErrExists reports that a destination already exists; it was not changed.
	ErrExists = errors.New("destination already exists")
	// ErrSymlink reports a symbolic link inside the shared root.
	ErrSymlink = errors.New("refusing to follow a symbolic link in the shared root")
	// ErrNotDirectory reports a path component that exists but is not a directory.
	ErrNotDirectory = errors.New("path component is not a directory")
	// ErrNotSupported reports an operating system without shared-root support.
	ErrNotSupported = errors.New("shared-root commands are not supported on this operating system")
)

// UsageError is an invalid argument, flag or name. It maps to ExitUsage.
type UsageError struct {
	msg string
}

func (e *UsageError) Error() string { return e.msg }

// Usagef returns a UsageError with a formatted message.
func Usagef(format string, args ...any) error {
	return &UsageError{msg: fmt.Sprintf(format, args...)}
}

// ExitCode maps an error to the exit code of the standard: 0 on success, 2 on
// a usage error, 1 on every other error.
func ExitCode(err error) int {
	if err == nil {
		return ExitOK
	}
	var usage *UsageError
	if errors.As(err, &usage) {
		return ExitUsage
	}
	return ExitProblem
}
