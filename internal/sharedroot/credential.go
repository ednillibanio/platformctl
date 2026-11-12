package sharedroot

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Credential value lengths of platformctl credential: --length bytes of
// random data before base64 encoding.
const (
	DefaultCredentialLength = 32
	MaxCredentialLength     = 1024
)

// maxStdinValue bounds the value read from standard input for NAME=-.
const maxStdinValue = 64 << 10

// CredentialOptions are the inputs of platformctl credential.
type CredentialOptions struct {
	Root     string
	Env      string
	Consumer string // writes under credentials/<consumer>/
	Target   string // with Identity, writes under credentials/target-<t>/<identity>/
	Identity string
	Specs    []string // NAME, NAME=VALUE or NAME=-
	Length   int      // random bytes before base64; 0 means the default
	Stdin    io.Reader
	DryRun   bool
	Out      io.Writer
}

// credentialSource is where the value of one credential comes from.
type credentialSource int

const (
	sourceRandom credentialSource = iota
	sourceLiteral
	sourceStdin
)

func (s credentialSource) String() string {
	switch s {
	case sourceLiteral:
		return "literal value"
	case sourceStdin:
		return "value from standard input"
	}
	return "random value"
}

type credentialSpec struct {
	name   string
	source credentialSource
	value  []byte // literal only
}

// Credential writes credential files as in the standard section "platformctl
// credential": one file per name, mode 0600, no trailing newline, never
// overwriting an existing file (it is skipped and reported). It validates
// every argument before it writes, reads standard input at most once and
// only for a NAME=- whose file does not exist, and never prints a value.
func Credential(opts CredentialOptions) error {
	if err := ValidateEnv(opts.Env); err != nil {
		return err
	}
	dir, err := credentialDir(opts)
	if err != nil {
		return err
	}
	length := opts.Length
	if length == 0 {
		length = DefaultCredentialLength
	}
	if length < 1 || length > MaxCredentialLength {
		return Usagef("--length must be between 1 and %d, got %d", MaxCredentialLength, length)
	}
	specs, err := parseCredentialSpecs(opts.Specs)
	if err != nil {
		return err
	}

	c := &credentialRun{opts: opts, dir: dir}
	if err := c.openExisting(); err != nil {
		return err
	}
	defer c.close()
	// Decide every skip first, so standard input is read only when the
	// NAME=- file is really written.
	pending := make([]credentialSpec, 0, len(specs))
	var skipped []string
	for _, s := range specs {
		exists, err := c.exists(s.name)
		if err != nil {
			return err
		}
		if exists {
			skipped = append(skipped, s.name)
			continue
		}
		pending = append(pending, s)
	}
	if c.opts.DryRun {
		if err := c.planDir(); err != nil {
			return err
		}
		c.reportSkipped(skipped)
		for _, s := range pending {
			c.say("would create", c.path(s.name)+" ("+s.source.String()+")")
		}
		return nil
	}
	// Resolve every value before anything is created, so an invalid value
	// read from standard input leaves the tree unchanged.
	values := make([][]byte, len(pending))
	for i, s := range pending {
		value, err := c.value(s, length)
		if err != nil {
			return err
		}
		values[i] = value
	}
	if err := c.createDir(); err != nil {
		return err
	}
	c.reportSkipped(skipped)
	for i, s := range pending {
		if err := c.publish(s, values[i]); err != nil {
			return err
		}
	}
	return nil
}

func (c *credentialRun) reportSkipped(names []string) {
	for _, name := range names {
		c.say("exists", c.path(name)+" (skipped, not changed)")
	}
}

// credentialDir validates the destination flags and returns the directory
// components relative to the root.
func credentialDir(opts CredentialOptions) ([]string, error) {
	switch {
	case opts.Consumer != "" && (opts.Target != "" || opts.Identity != ""):
		return nil, Usagef("--consumer cannot be combined with --target or --identity")
	case opts.Consumer != "":
		if err := ValidateConsumer(opts.Consumer); err != nil {
			return nil, err
		}
		return []string{opts.Env, "credentials", opts.Consumer}, nil
	case opts.Target != "" && opts.Identity == "":
		return nil, Usagef("--target requires --identity")
	case opts.Identity != "" && opts.Target == "":
		return nil, Usagef("--identity requires --target")
	case opts.Target != "":
		if err := ValidateTarget(opts.Target); err != nil {
			return nil, err
		}
		if err := ValidateIdentity(opts.Identity); err != nil {
			return nil, err
		}
		return []string{opts.Env, "credentials", "target-" + opts.Target, opts.Identity}, nil
	}
	return nil, Usagef("one of --consumer or --target with --identity is required")
}

func parseCredentialSpecs(args []string) ([]credentialSpec, error) {
	if len(args) == 0 {
		return nil, Usagef("at least one NAME, NAME=VALUE or NAME=- is required")
	}
	var specs []credentialSpec
	seen := map[string]bool{}
	stdin := ""
	for _, arg := range args {
		name, value, literal := strings.Cut(arg, "=")
		if err := ValidateKey(name); err != nil {
			return nil, err
		}
		if seen[name] {
			return nil, Usagef("credential %s is given more than once", name)
		}
		seen[name] = true
		s := credentialSpec{name: name, source: sourceRandom}
		switch {
		case literal && value == "-":
			if stdin != "" {
				return nil, Usagef("only one NAME=- may read standard input; %s and %s both do", stdin, name)
			}
			stdin = name
			s.source = sourceStdin
		case literal:
			if err := checkCredentialValue(name, []byte(value)); err != nil {
				return nil, err
			}
			s.source, s.value = sourceLiteral, []byte(value)
		}
		specs = append(specs, s)
	}
	return specs, nil
}

// checkCredentialValue rejects a value a credential file cannot hold: an
// empty one (doctor reports an empty credential file) or one with a line
// break (a credential file holds one value with no trailing newline).
func checkCredentialValue(name string, value []byte) error {
	if len(value) == 0 {
		return Usagef("credential %s: empty value", name)
	}
	if bytes.ContainsAny(value, "\n\r") {
		return Usagef("credential %s: the value holds a line break; a credential file holds one value", name)
	}
	return nil
}

// credentialRun writes the credentials of one command, or only prints them
// with DryRun.
type credentialRun struct {
	opts        CredentialOptions
	dir         []string
	tree        *Tree // nil until the root exists or is created
	missingRoot bool  // the root did not exist when the run started
	read        bool  // standard input was read
}

// openExisting opens the root when it exists; nothing is created yet.
func (c *credentialRun) openExisting() error {
	if _, err := os.Lstat(c.opts.Root); errors.Is(err, os.ErrNotExist) {
		c.missingRoot = true
		return nil
	}
	tree, err := OpenTree(c.opts.Root, false)
	if err != nil {
		return err
	}
	c.tree = tree
	return nil
}

func (c *credentialRun) close() {
	if c.tree != nil {
		_ = c.tree.Close()
	}
}

func (c *credentialRun) say(action, path string) {
	fmt.Fprintf(c.opts.Out, "%-12s %s\n", action, path)
}

func (c *credentialRun) path(name ...string) string {
	return filepath.Join(append(append([]string{c.opts.Root}, c.dir...), name...)...)
}

// createDir creates the root and the missing directories of the destination
// (mode 0700).
func (c *credentialRun) createDir() error {
	if c.tree == nil {
		tree, err := OpenTree(c.opts.Root, true)
		if err != nil {
			return err
		}
		c.tree = tree
		if c.missingRoot {
			c.say("created", c.opts.Root+"/")
		}
	}
	created, err := c.tree.EnsureDir(c.dir...)
	for _, rel := range created {
		c.say("created", filepath.Join(c.opts.Root, filepath.FromSlash(rel))+"/")
	}
	return err
}

// planDir prints the directories a run would create.
func (c *credentialRun) planDir() error {
	if c.tree == nil {
		c.say("would create", c.opts.Root+"/")
		for j := range c.dir {
			c.say("would create", filepath.Join(append([]string{c.opts.Root}, c.dir[:j+1]...)...)+"/")
		}
		return nil
	}
	for i := range c.dir {
		exists, isDir := false, false
		if c.tree != nil {
			var err error
			if exists, isDir, err = c.tree.Entry(c.dir[:i], c.dir[i]); err != nil {
				return err
			}
		}
		if exists && !isDir {
			return fmt.Errorf("%w: %s", ErrNotDirectory, filepath.Join(append([]string{c.opts.Root}, c.dir[:i+1]...)...))
		}
		if !exists {
			for j := i; j < len(c.dir); j++ {
				c.say("would create", filepath.Join(append([]string{c.opts.Root}, c.dir[:j+1]...)...)+"/")
			}
			return nil
		}
	}
	return nil
}

// exists reports whether the credential file is present. A directory or any
// other entry that is not a regular file there is an error, never skipped.
func (c *credentialRun) exists(name string) (bool, error) {
	if c.tree == nil {
		return false, nil
	}
	exists, isDir, err := c.tree.Entry(c.dir, name)
	if err != nil {
		return false, err
	}
	if isDir {
		return false, fmt.Errorf("%s is a directory, not a credential file", c.path(name))
	}
	return exists, nil
}

func (c *credentialRun) value(s credentialSpec, length int) ([]byte, error) {
	switch s.source {
	case sourceLiteral:
		return s.value, nil
	case sourceStdin:
		return c.readStdin(s.name)
	}
	return randomValue(length)
}

// readStdin reads the whole standard input once and drops one trailing line
// break, so `printf '%s\n' "$value"` and a typed value ended by Enter both
// work.
func (c *credentialRun) readStdin(name string) ([]byte, error) {
	if c.read {
		return nil, fmt.Errorf("standard input was already read")
	}
	c.read = true
	if c.opts.Stdin == nil {
		return nil, Usagef("credential %s: no standard input", name)
	}
	// Read up to the limit plus a \r\n ending and one more byte, so a value
	// of exactly maxStdinValue bytes is accepted with its line ending.
	data, err := io.ReadAll(io.LimitReader(c.opts.Stdin, maxStdinValue+3))
	if err != nil {
		return nil, fmt.Errorf("credential %s: read standard input: %w", name, err)
	}
	// Only one line ending is dropped: \r\n or \n, never a lone \r; any
	// other line break left is rejected below.
	if bytes.HasSuffix(data, []byte("\r\n")) {
		data = data[:len(data)-2]
	} else {
		data = bytes.TrimSuffix(data, []byte("\n"))
	}
	if len(data) > maxStdinValue {
		return nil, Usagef("credential %s: the value from standard input is longer than %d bytes", name, maxStdinValue)
	}
	if err := checkCredentialValue(name, data); err != nil {
		return nil, err
	}
	return data, nil
}

// randomValue returns length bytes from the system's cryptographic random
// source, base64 encoded with padding (the format of `openssl rand -base64`
// used by gen-credential, without its newline).
func randomValue(length int) ([]byte, error) {
	raw := make([]byte, length)
	if _, err := rand.Read(raw); err != nil {
		return nil, fmt.Errorf("read random data: %w", err)
	}
	out := make([]byte, base64.StdEncoding.EncodedLen(length))
	base64.StdEncoding.Encode(out, raw)
	return out, nil
}

func (c *credentialRun) publish(s credentialSpec, value []byte) error {
	err := c.tree.PublishFile(c.dir, s.name, value, 0o600)
	switch {
	case err == nil:
		detail := s.source.String()
		if s.source == sourceRandom {
			detail = fmt.Sprintf("random value, %d characters", len(value))
		}
		c.say("created", c.path(s.name)+" ("+detail+")")
		return nil
	case errors.Is(err, ErrExists):
		// Created concurrently by another run: never overwritten.
		c.say("exists", c.path(s.name)+" (skipped, not changed)")
		return nil
	}
	return err
}
