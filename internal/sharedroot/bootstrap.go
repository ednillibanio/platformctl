package sharedroot

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// BootstrapOptions are the inputs of platformctl bootstrap-config.
type BootstrapOptions struct {
	Root         string
	Env          string
	Targets      []string
	Consumers    []string
	Declarations []string // paths of shared-root.yaml files
	// AutomationKey generates the automation key pair.
	AutomationKey bool
	// ImportKey, when not nil, is read for the automation private key.
	ImportKey io.Reader
	DryRun    bool
	Out       io.Writer // actions and lines to add
	Err       io.Writer // warnings
}

// Bootstrap creates the shared-root skeleton of the standard section
// "platformctl bootstrap-config": directories, commented placeholders and,
// on request, the automation key pair. It validates every input before it
// writes, never writes a value, and never changes an existing file.
func Bootstrap(opts BootstrapOptions) error {
	if err := ValidateEnv(opts.Env); err != nil {
		return err
	}
	targets, err := unique(opts.Targets, ValidateTarget)
	if err != nil {
		return err
	}
	consumers, err := unique(opts.Consumers, ValidateConsumer)
	if err != nil {
		return err
	}
	if opts.AutomationKey && opts.ImportKey != nil {
		return Usagef("--automation-key and --automation-key-import are mutually exclusive")
	}
	decls, err := readDeclarations(opts.Declarations, opts.Err)
	if err != nil {
		return err
	}
	byConsumer := map[string]Declaration{}
	for _, d := range decls {
		if _, dup := byConsumer[d.Consumer]; dup {
			return Usagef("two declarations for consumer %s", d.Consumer)
		}
		byConsumer[d.Consumer] = d
		if !contains(consumers, d.Consumer) {
			consumers = append(consumers, d.Consumer)
		}
	}
	if len(decls) == 0 {
		warnf(opts.Err, "no declaration given: writing a generic skeleton without key placeholders")
	} else {
		for _, c := range consumers {
			if _, ok := byConsumer[c]; !ok {
				warnf(opts.Err, "no declaration for consumer %s: its hosts files get a generic skeleton", c)
			}
		}
	}
	var pair *KeyPair
	switch {
	case opts.ImportKey != nil:
		k, err := ImportAutomationKey(opts.ImportKey, opts.Env)
		if err != nil {
			return err
		}
		pair = &k
	case opts.AutomationKey:
		pair = &KeyPair{} // generated only when the pair does not exist
	default:
		for _, d := range decls {
			if d.AutomationKey {
				warnf(opts.Err, "%s uses the automation key pair: run with --automation-key on the first controller of %s, --automation-key-import - on the others", d.Consumer, opts.Env)
				break
			}
		}
	}

	plan := planSkeleton(opts.Env, targets, consumers, decls, byConsumer)
	r, err := newRunner(opts)
	if err != nil {
		return err
	}
	defer r.close()
	for _, dir := range plan.dirs {
		if err := r.dir(dir); err != nil {
			return err
		}
	}
	for _, f := range plan.files {
		if err := r.file(f); err != nil {
			return err
		}
	}
	if pair != nil {
		return r.automation(opts.Env, pair)
	}
	return nil
}

func unique(names []string, validate func(string) error) ([]string, error) {
	var out []string
	for _, n := range names {
		if err := validate(n); err != nil {
			return nil, err
		}
		if !contains(out, n) {
			out = append(out, n)
		}
	}
	return out, nil
}

func readDeclarations(paths []string, warn io.Writer) ([]Declaration, error) {
	var decls []Declaration
	var problems Problems
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read declaration: %w", err)
		}
		d, err := ParseDeclaration(path, data)
		if err != nil {
			var ps Problems
			if errors.As(err, &ps) {
				problems = append(problems, ps...)
				continue
			}
			return nil, err
		}
		for _, w := range d.Warnings {
			warnf(warn, "%s", w)
		}
		decls = append(decls, d)
	}
	if err := problems.Err(); err != nil {
		return nil, fmt.Errorf("invalid declaration, nothing written:\n%w", err)
	}
	return decls, nil
}

func warnf(w io.Writer, format string, args ...any) {
	if w != nil {
		fmt.Fprintf(w, "warning: "+format+"\n", args...)
	}
}

// plannedFile is one shared file of the skeleton.
type plannedFile struct {
	dir  []string
	name string
	kind FileKind
	head string
	keys []KeyEntry
}

func (f plannedFile) content() []byte {
	var b strings.Builder
	b.WriteString(f.head)
	for _, k := range f.keys {
		b.WriteString("\n")
		b.WriteString(placeholder(k))
	}
	return []byte(b.String())
}

// placeholder is the commented block of one key: its description, an
// optional example, and the commented assignment `# KEY=`.
func placeholder(k KeyEntry) string {
	need := "optional"
	if k.Required {
		need = "required"
	}
	block := fmt.Sprintf("# %s (%s): %s\n", k.Name, need, k.Description)
	if k.Example != "" {
		block += "# Example: " + k.Example + "\n"
	}
	return block + "# " + k.Name + "=\n"
}

type skeleton struct {
	dirs  [][]string
	files []plannedFile
}

func planSkeleton(env string, targets, consumers []string, decls []Declaration, byConsumer map[string]Declaration) skeleton {
	var s skeleton
	s.dirs = append(s.dirs, []string{env, "targets"}, []string{env, "credentials"})
	var targetKeys []KeyEntry
	var identities []string
	for _, d := range decls {
		for _, k := range d.Target {
			if i := indexKey(targetKeys, k.Name); i >= 0 {
				targetKeys[i].Required = targetKeys[i].Required || k.Required
				continue
			}
			targetKeys = append(targetKeys, k)
		}
		for _, tc := range d.TargetCredentials {
			if !contains(identities, tc.Identity) {
				identities = append(identities, tc.Identity)
			}
		}
	}
	for _, t := range targets {
		s.dirs = append(s.dirs,
			[]string{env, "targets", t, "hosts"},
			[]string{env, "credentials", "target-" + t})
		for _, id := range identities {
			s.dirs = append(s.dirs, []string{env, "credentials", "target-" + t, id})
		}
		s.files = append(s.files, plannedFile{
			dir: []string{env, "targets", t}, name: "vars.sh", kind: VarsFile,
			head: varsHeader(env, t, len(decls) == 0), keys: targetKeys,
		})
	}
	for _, c := range consumers {
		s.dirs = append(s.dirs, []string{env, "credentials", c})
	}
	for _, t := range targets {
		for _, c := range consumers {
			d, declared := byConsumer[c]
			s.files = append(s.files, plannedFile{
				dir: []string{env, "targets", t, "hosts"}, name: c + ".sh", kind: HostsFile,
				head: hostsHeader(env, t, c, !declared), keys: d.Host,
			})
		}
	}
	return s
}

func indexKey(keys []KeyEntry, name string) int {
	for i, k := range keys {
		if k.Name == name {
			return i
		}
	}
	return -1
}

const fileRules = `# Only comments, blank lines and KEY=value lines with a literal value are
# allowed: unquoted without spaces, shell characters or ~, 'single quoted',
# or "double quoted" without $ ` + "`" + ` \ or ". To set a key, write its value after
# the = and remove the leading "# ". See the platform-catalog
# provision-shared-root-standard.md.
`

func varsHeader(env, target string, generic bool) string {
	h := fmt.Sprintf("# Shared values of target %s in environment %s.\n# Created by platformctl bootstrap-config. Keys: GOVC_*, TARGET_* and\n# DISPOSABLE_* only.\n", target, env) + fileRules
	if generic {
		h += "#\n# No declaration was given: add the target keys the consumers declare.\n"
	}
	return h
}

func hostsHeader(env, target, consumer string, generic bool) string {
	h := fmt.Sprintf("# Values of %s on target %s in environment %s.\n# Created by platformctl bootstrap-config. Keys: HOST_* only.\n", consumer, target, env) + fileRules
	if generic {
		h += "#\n# No declaration was given: add the host keys " + consumer + " declares.\n"
	}
	return h
}

// runner applies the plan, or only prints it with DryRun.
type runner struct {
	opts BootstrapOptions
	tree *Tree // nil in a dry run when the root does not exist
	seen map[string]bool
}

func newRunner(opts BootstrapOptions) (*runner, error) {
	r := &runner{opts: opts}
	_, statErr := os.Lstat(opts.Root)
	missing := errors.Is(statErr, os.ErrNotExist)
	if opts.DryRun {
		if missing {
			r.say("would create", opts.Root+"/")
			return r, nil
		}
		tree, err := OpenTree(opts.Root, false)
		if err != nil {
			return nil, err
		}
		r.tree = tree
		return r, nil
	}
	tree, err := OpenTree(opts.Root, true)
	if err != nil {
		return nil, err
	}
	if missing {
		r.say("created", opts.Root+"/")
	}
	r.tree = tree
	return r, nil
}

func (r *runner) close() {
	if r.tree != nil {
		_ = r.tree.Close()
	}
}

func (r *runner) say(action, path string) {
	fmt.Fprintf(r.opts.Out, "%-12s %s\n", action, path)
}

func (r *runner) path(dir []string, name ...string) string {
	return filepath.Join(append(append([]string{r.opts.Root}, dir...), name...)...)
}

func (r *runner) exists(dir []string, name string) (bool, bool, error) {
	if r.tree == nil {
		return false, false, nil
	}
	return r.tree.Entry(dir, name)
}

func (r *runner) dir(dir []string) error {
	if r.opts.DryRun {
		for i := range dir {
			exists, isDir, err := r.exists(dir[:i], dir[i])
			if err != nil {
				return err
			}
			if exists && !isDir {
				return fmt.Errorf("%w: %s", ErrNotDirectory, r.path(dir[:i+1]))
			}
			if !exists {
				for j := i; j < len(dir); j++ {
					r.sayOnce("would create", r.path(dir[:j+1])+"/")
				}
				return nil
			}
		}
		return nil
	}
	created, err := r.tree.EnsureDir(dir...)
	for _, rel := range created {
		r.say("created", filepath.Join(r.opts.Root, filepath.FromSlash(rel))+"/")
	}
	return err
}

// sayOnce avoids repeating a "would create" line for a parent directory
// shared by several planned directories.
func (r *runner) sayOnce(action, path string) {
	if r.seen == nil {
		r.seen = map[string]bool{}
	}
	if r.seen[path] {
		return
	}
	r.seen[path] = true
	r.say(action, path)
}

func (r *runner) file(f plannedFile) error {
	path := r.path(f.dir, f.name)
	var existing []byte
	found := false
	if r.tree != nil {
		data, err := r.tree.ReadFile(f.dir, f.name)
		switch {
		case err == nil:
			existing, found = data, true
		case !errors.Is(err, os.ErrNotExist):
			return err
		}
	}
	if !found {
		if r.opts.DryRun {
			r.say("would create", path)
			return nil
		}
		err := r.tree.PublishFile(f.dir, f.name, f.content(), 0o644)
		if err == nil {
			r.say("created", path)
			return nil
		}
		if !errors.Is(err, ErrExists) {
			return err
		}
		// Created concurrently by another run: check it as an existing file.
		if existing, err = r.tree.ReadFile(f.dir, f.name); err != nil {
			return err
		}
	}
	r.say("exists", path)
	parsed, perr := ParseShellFile(path, f.kind, existing)
	if perr != nil {
		warnf(r.opts.Err, "%s has format problems; run platformctl doctor", path)
	}
	present := map[string]bool{}
	for _, a := range parsed.Assignments {
		present[a.Key] = true
	}
	for _, p := range parsed.Placeholders {
		present[p.Key] = true
	}
	var missing []string
	for _, k := range f.keys {
		if !present[k.Name] {
			missing = append(missing, placeholder(k))
		}
	}
	if len(missing) > 0 {
		fmt.Fprintf(r.opts.Out, "%s lacks declared keys; it was not changed. Add these lines:\n\n%s\n",
			path, strings.Join(missing, "\n"))
	}
	return nil
}

func (r *runner) automation(env string, pair *KeyPair) error {
	parent := []string{env, "credentials"}
	dir := append(append([]string{}, parent...), AutomationDir)
	path := r.path(dir) + "/"
	exists, isDir, err := r.exists(parent, AutomationDir)
	if err != nil {
		return err
	}
	if !exists {
		if r.opts.DryRun {
			r.say("would create", path+" (automation key pair)")
			return nil
		}
		if len(pair.Private) == 0 {
			generated, err := GenerateAutomationKey(env)
			if err != nil {
				return err
			}
			*pair = generated
		}
		err := r.tree.PublishDir(parent, AutomationDir, pair.Files())
		if err == nil {
			r.say("created", path+" (automation key pair)")
			return nil
		}
		if !errors.Is(err, ErrExists) {
			return err
		}
		exists, isDir = true, true // published concurrently by another run
	}
	if !isDir {
		return fmt.Errorf("%w: %s", ErrNotDirectory, path)
	}
	var missing []string
	for _, name := range []string{AutomationPrivateKey, AutomationPublicKey} {
		if r.tree == nil {
			missing = append(missing, name)
			continue
		}
		// ReadFile accepts only a regular file reached without a symbolic
		// link; the content is never printed.
		if _, err := r.tree.ReadFile(dir, name); err != nil {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return Problems{{File: path, Msg: "incomplete automation key pair, missing or not a regular file: " + strings.Join(missing, " and ") + "; nothing was created in it"}}
	}
	r.say("exists", path+" (automation key pair)")
	return nil
}
