package sharedroot

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"
)

func TestResolveRootPrecedence(t *testing.T) {
	flagDir := filepath.Join(t.TempDir(), "flag")
	envDir := filepath.Join(t.TempDir(), "env")
	homeDir := t.TempDir()
	home := func() (string, error) { return homeDir, nil }

	cases := []struct {
		name, flag, env, want string
	}{
		{"flag wins", flagDir, envDir, flagDir},
		{"env when no flag", "", envDir, envDir},
		{"default under home", "", "", filepath.Join(homeDir, ".config", "assembleiarr")},
		{"cleaned", flagDir + string(filepath.Separator) + ".", "", flagDir},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveRoot(tc.flag, tc.env, home)
			if err != nil {
				t.Fatalf("resolveRoot: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolveRootRejectsRelative(t *testing.T) {
	home := func() (string, error) { return t.TempDir(), nil }
	for _, tc := range []struct{ flag, env string }{{"relative/dir", ""}, {"", "relative/dir"}} {
		_, err := resolveRoot(tc.flag, tc.env, home)
		if ExitCode(err) != ExitUsage {
			t.Fatalf("flag=%q env=%q: want usage error, got %v", tc.flag, tc.env, err)
		}
	}
}

func TestResolveRootHomeFailure(t *testing.T) {
	_, err := resolveRoot("", "", func() (string, error) { return "", errors.New("no home") })
	if err == nil || ExitCode(err) != ExitProblem {
		t.Fatalf("want problem error, got %v", err)
	}
}

func TestResolveRootUsesEnvVar(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvVar, dir)
	got, err := ResolveRoot("")
	if err != nil || got != filepath.Clean(dir) {
		t.Fatalf("got %q, %v; want %q", got, err, dir)
	}
}

func TestExitCode(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{nil, ExitOK},
		{Usagef("bad flag"), ExitUsage},
		{fmt.Errorf("wrapped: %w", Usagef("bad name")), ExitUsage},
		{ErrExists, ExitProblem},
		{ErrNotSupported, ExitProblem},
		{errors.New("other"), ExitProblem},
	}
	for _, tc := range cases {
		if got := ExitCode(tc.err); got != tc.want {
			t.Errorf("ExitCode(%v) = %d, want %d", tc.err, got, tc.want)
		}
	}
}
