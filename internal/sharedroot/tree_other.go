//go:build !linux && !darwin

package sharedroot

import "os"

// Tree is not supported on this operating system.
type Tree struct{}

// OpenTree returns ErrNotSupported.
func OpenTree(string, bool) (*Tree, error) { return nil, ErrNotSupported }

func (t *Tree) Root() string { return "" }

func (t *Tree) Close() error { return nil }

func (t *Tree) EnsureDir(...string) ([]string, error) { return nil, ErrNotSupported }

func (t *Tree) PublishFile([]string, string, []byte, os.FileMode) error { return ErrNotSupported }

func (t *Tree) PublishDir([]string, string, []File) error { return ErrNotSupported }

func (t *Tree) Entry([]string, string) (bool, bool, error) { return false, false, ErrNotSupported }

func (t *Tree) ReadFile([]string, string) ([]byte, error) { return nil, ErrNotSupported }
