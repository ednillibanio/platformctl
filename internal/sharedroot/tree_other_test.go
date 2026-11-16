//go:build !linux && !darwin

package sharedroot

import (
	"errors"
	"testing"
)

func TestOpenTreeNotSupported(t *testing.T) {
	if _, err := OpenTree(t.TempDir(), true); !errors.Is(err, ErrNotSupported) {
		t.Fatalf("want ErrNotSupported, got %v", err)
	}
}
