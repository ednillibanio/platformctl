package sharedroot

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// DoctorOptions are the inputs of platformctl doctor.
type DoctorOptions struct {
	Root         string
	Env          string
	Targets      []string // limits the declared checks; default every target found
	Consumers    []string // consumers whose files must exist, besides the declared ones
	Declarations []string // paths of shared-root.yaml files
	Format       string   // "text" (default) or "json"
	Out          io.Writer
}

// Finding is one problem or warning. It never holds a secret value.
type Finding struct {
	File    string `json:"file"`
	Line    int    `json:"line,omitempty"`
	Key     string `json:"key,omitempty"`
	Message string `json:"message"`
}

func (f Finding) String() string {
	return Problem{File: f.File, Line: f.Line, Key: f.Key, Msg: f.Message}.String()
}

// Report is the result of platformctl doctor. Values are only those of the
// shared files, which hold no secrets; a value of a file where a credential
// value was found is withheld.
type Report struct {
	Env           string                   `json:"env"`
	Root          string                   `json:"root"`
	OK            bool                     `json:"ok"`
	Problems      []Finding                `json:"problems"`
	Warnings      []Finding                `json:"warnings"`
	Targets       map[string]*TargetReport `json:"targets"`
	Credentials   map[string][]string      `json:"credentials"`
	AutomationKey *AutomationReport        `json:"automation_key,omitempty"`
}

// TargetReport holds the non-secret values of one target and the names of
// its credential files.
type TargetReport struct {
	Vars        map[string]string            `json:"vars"`
	Hosts       map[string]map[string]string `json:"hosts"`
	Credentials map[string][]string          `json:"credentials"`
}

// AutomationReport describes the automation key pair without its private key.
type AutomationReport struct {
	PrivateKey  bool   `json:"private_key"`
	PublicKey   bool   `json:"public_key"`
	Fingerprint string `json:"fingerprint,omitempty"`
}

// WithheldValue replaces, in the report, a value of a shared file that holds
// a credential value.
const WithheldValue = "<withheld: the file holds a credential value>"

// minLeakLength is the shortest credential value searched for in the shared
// files; a shorter one would match unrelated text.
const minLeakLength = 8

// Doctor checks the shared root of an environment as in the standard section
// "platformctl doctor". It only reads, reports every finding, writes the
// report to opts.Out and returns an error (exit 1) when there is a problem.
func Doctor(opts DoctorOptions) error {
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
	switch opts.Format {
	case "", "text", "json":
	default:
		return Usagef("--format must be text or json, got %q", opts.Format)
	}
	if !supported() {
		return ErrNotSupported
	}
	d := &doctor{
		opts:     opts,
		euid:     os.Geteuid(),
		vars:     map[string]ShellFile{},
		hosts:    map[string]map[string]ShellFile{},
		credDirs: map[string]map[string]bool{},
		tcreds:   map[string]map[string]map[string]bool{},
		report: Report{
			Env: opts.Env, Root: opts.Root, Problems: []Finding{}, Warnings: []Finding{},
			Targets: map[string]*TargetReport{}, Credentials: map[string][]string{},
		},
	}
	d.readDeclarations()
	d.walk()
	d.checkDeclared(targets, consumers)
	d.checkLeaks()
	d.finish()
	if err := d.write(); err != nil {
		return err
	}
	if n := len(d.report.Problems); n > 0 {
		return fmt.Errorf("doctor found %d problem(s) in %s", n, d.envPath())
	}
	return nil
}

// sharedFile is a file that is not secret (mode 0644), searched for leaked
// credentials.
type sharedFile struct {
	path    string
	content []byte
	target  string // with consumer, the file whose values the report holds
	hosts   string // consumer of a hosts file; "" for vars.sh or another file
}

// credValue is a declared credential found in the tree, kept in memory only
// to search the shared files for it.
type credValue struct {
	name  string
	value []byte
}

type doctor struct {
	opts   DoctorOptions
	euid   int
	tree   *Tree
	decls  []Declaration
	report Report

	vars     map[string]ShellFile            // target -> vars.sh
	hosts    map[string]map[string]ShellFile // target -> consumer -> hosts file
	found    []string                        // targets found under targets/
	credDirs map[string]map[string]bool      // consumer -> credential names
	tcreds   map[string]map[string]map[string]bool
	autoDir  bool
	shared   []sharedFile
	readable []string // every regular file readable by others
	creds    []credValue
}

func (d *doctor) problem(file string, line int, key, format string, args ...any) {
	d.report.Problems = append(d.report.Problems, Finding{File: file, Line: line, Key: key, Message: fmt.Sprintf(format, args...)})
}

func (d *doctor) warn(file string, line int, key, format string, args ...any) {
	d.report.Warnings = append(d.report.Warnings, Finding{File: file, Line: line, Key: key, Message: fmt.Sprintf(format, args...)})
}

func (d *doctor) envPath(rel ...string) string {
	return filepath.Join(append([]string{d.opts.Root, d.opts.Env}, rel...)...)
}

func (d *doctor) readDeclarations() {
	seen := map[string]string{}
	for _, path := range d.opts.Declarations {
		data, err := os.ReadFile(path)
		if err != nil {
			d.problem(path, 0, "", "cannot read the declaration: %v", errors.Unwrap(err))
			continue
		}
		decl, err := ParseDeclaration(path, data)
		if err != nil {
			var ps Problems
			if !errors.As(err, &ps) {
				d.problem(path, 0, "", "invalid declaration: %v", err)
				continue
			}
			for _, p := range ps {
				d.problem(p.File, p.Line, p.Key, "invalid declaration: %s", p.Msg)
			}
			continue
		}
		for _, w := range decl.Warnings {
			d.warn(path, 0, "", "%s", w)
		}
		if first, dup := seen[decl.Consumer]; dup {
			d.problem(path, 0, "", "second declaration for consumer %s (first: %s)", decl.Consumer, first)
			continue
		}
		seen[decl.Consumer] = path
		d.decls = append(d.decls, decl)
	}
	if len(d.opts.Declarations) == 0 {
		d.warn(d.envPath(), 0, "", "no declaration given: required keys, credentials and undeclared files are not checked")
	}
}

// entry checks what every path of the tree must satisfy and reports whether
// it can be inspected further: not a symbolic link, not a temporary entry
// left by an interrupted run, the expected type, owned by the current user,
// and the expected mode.
func (d *doctor) entry(path string, e EntryInfo, want EntryType, perm os.FileMode) bool {
	switch {
	case e.Type == TypeSymlink:
		d.problem(path, 0, "", "symbolic link: no path in the shared root may be a symbolic link")
		return false
	case strings.HasPrefix(e.Name, TempPrefix):
		d.problem(path, 0, "", "temporary entry left by an interrupted platformctl run; remove it after checking it")
		return false
	case e.Type != want:
		d.problem(path, 0, "", "is a %s, want a %s", typeName(e.Type), typeName(want))
		return false
	}
	if e.UID != d.euid {
		d.problem(path, 0, "", "owned by uid %d, want the account that runs the tools (uid %d)", e.UID, d.euid)
	}
	if e.Perm != perm || e.Special {
		d.problem(path, 0, "", "mode %04o, want %04o", e.Perm, perm)
	}
	d.noteReadable(path, e)
	return true
}

// noteReadable records a regular file readable by others, searched for
// leaked credentials.
func (d *doctor) noteReadable(path string, e EntryInfo) {
	if e.Type == TypeFile && e.Perm&0o044 != 0 {
		d.readable = append(d.readable, path)
	}
}

func typeName(t EntryType) string {
	switch t {
	case TypeDir:
		return "directory"
	case TypeFile:
		return "regular file"
	case TypeSymlink:
		return "symbolic link"
	}
	return "special file"
}

// list lists a directory; a failure is a problem and yields no entries.
func (d *doctor) list(rel ...string) []EntryInfo {
	entries, err := d.tree.List(append([]string{d.opts.Env}, rel...))
	if err != nil {
		d.problem(d.envPath(rel...), 0, "", "cannot list: %v", err)
		return nil
	}
	return entries
}

func (d *doctor) read(rel ...string) ([]byte, bool) {
	dir := append([]string{d.opts.Env}, rel[:len(rel)-1]...)
	data, err := d.tree.ReadFile(dir, rel[len(rel)-1])
	if err != nil {
		d.problem(d.envPath(rel...), 0, "", "cannot read: %v", err)
		return nil, false
	}
	return data, true
}

func (d *doctor) walk() {
	if _, err := os.Lstat(d.opts.Root); errors.Is(err, os.ErrNotExist) {
		d.problem(d.opts.Root, 0, "", "the shared root does not exist; run platformctl bootstrap-config")
		return
	}
	tree, err := OpenTree(d.opts.Root, false)
	if err != nil {
		d.problem(d.opts.Root, 0, "", "cannot open the shared root: %v", err)
		return
	}
	d.tree = tree
	env, err := tree.Lstat(nil, d.opts.Env)
	if errors.Is(err, os.ErrNotExist) {
		d.problem(d.envPath(), 0, "", "the environment does not exist in the shared root; run platformctl bootstrap-config --env %s", d.opts.Env)
		return
	}
	if err != nil {
		d.problem(d.envPath(), 0, "", "cannot inspect: %v", err)
		return
	}
	if !d.entry(d.envPath(), env, TypeDir, 0o700) {
		return
	}
	present := map[string]bool{}
	for _, e := range d.list() {
		path := d.envPath(e.Name)
		switch e.Name {
		case "targets":
			if d.entry(path, e, TypeDir, 0o700) {
				present[e.Name] = true
				d.walkTargets()
			}
		case "credentials":
			if d.entry(path, e, TypeDir, 0o700) {
				present[e.Name] = true
				d.walkCredentials()
			}
		case "state":
			if d.entry(path, e, TypeDir, 0o700) {
				d.walkState()
			}
		case "backups":
			if d.entry(path, e, TypeDir, 0o700) {
				d.walkBackups()
			}
		default:
			d.unknown(path, e)
		}
	}
	for _, name := range []string{"targets", "credentials"} {
		if !present[name] {
			d.problem(d.envPath(name), 0, "", "missing directory; run platformctl bootstrap-config")
		}
	}
}

// unknown reports an entry that is not part of the layout: a warning, plus
// the general rules for it and everything below it.
func (d *doctor) unknown(path string, e EntryInfo) {
	if e.Type != TypeSymlink && !strings.HasPrefix(e.Name, TempPrefix) {
		d.warn(path, 0, "", "not part of the shared-root layout")
	}
	d.inspect(path, e)
}

// inspect applies the general rules to an entry outside the known layout and,
// for a directory, to everything below it: no symbolic link, no temporary
// entry, owner, directory mode 0700, only directories and regular files.
func (d *doctor) inspect(path string, e EntryInfo) {
	switch {
	case e.Type == TypeSymlink:
		d.problem(path, 0, "", "symbolic link: no path in the shared root may be a symbolic link")
	case strings.HasPrefix(e.Name, TempPrefix):
		d.problem(path, 0, "", "temporary entry left by an interrupted platformctl run; remove it after checking it")
	case e.Type == TypeDir:
		// Every directory under <root>/<env>/ is 0700; the general rules
		// apply to everything below an unknown one too.
		if d.entry(path, e, TypeDir, 0o700) {
			rel := d.rel(path)
			for _, child := range d.list(rel...) {
				d.unknown(d.envPath(append(append([]string{}, rel...), child.Name)...), child)
			}
		}
	default:
		if e.UID != d.euid {
			d.problem(path, 0, "", "owned by uid %d, want the account that runs the tools (uid %d)", e.UID, d.euid)
		}
		if e.Type == TypeFile && e.Special {
			d.problem(path, 0, "", "setuid, setgid or sticky bit set")
		}
		if e.Type != TypeFile {
			d.problem(path, 0, "", "is a %s; the shared root holds only directories and regular files", typeName(e.Type))
		}
		d.noteReadable(path, e)
	}
}

// rel returns the components of path relative to <root>/<env>/.
func (d *doctor) rel(path string) []string {
	r, err := filepath.Rel(d.envPath(), path)
	if err != nil || r == "." {
		return nil
	}
	return strings.Split(r, string(filepath.Separator))
}

func (d *doctor) walkTargets() {
	for _, e := range d.list("targets") {
		path := d.envPath("targets", e.Name)
		if e.Type != TypeDir || ValidateTarget(e.Name) != nil || strings.HasPrefix(e.Name, TempPrefix) {
			d.unknown(path, e)
			continue
		}
		if !d.entry(path, e, TypeDir, 0o700) {
			continue
		}
		t := e.Name
		d.found = append(d.found, t)
		tr := d.targetReport(t)
		seen := map[string]bool{}
		for _, f := range d.list("targets", t) {
			fpath := d.envPath("targets", t, f.Name)
			switch f.Name {
			case "vars.sh":
				if !d.entry(fpath, f, TypeFile, 0o644) {
					continue
				}
				seen[f.Name] = true
				if data, ok := d.read("targets", t, f.Name); ok {
					parsed := d.parse(fpath, VarsFile, data)
					d.vars[t] = parsed
					tr.Vars = values(parsed)
					d.shared = append(d.shared, sharedFile{path: fpath, content: data, target: t})
				}
			case "ca.pem":
				if d.entry(fpath, f, TypeFile, 0o644) {
					if data, ok := d.read("targets", t, f.Name); ok {
						d.shared = append(d.shared, sharedFile{path: fpath, content: data})
					}
				}
			case "hosts":
				if d.entry(fpath, f, TypeDir, 0o700) {
					seen[f.Name] = true
					d.walkHosts(t, tr)
				}
			default:
				d.unknown(fpath, f)
			}
		}
		if !seen["vars.sh"] {
			d.problem(d.envPath("targets", t, "vars.sh"), 0, "", "missing; run platformctl bootstrap-config --target %s", t)
		}
		if !seen["hosts"] {
			d.problem(d.envPath("targets", t, "hosts"), 0, "", "missing directory; run platformctl bootstrap-config --target %s", t)
		}
	}
}

func (d *doctor) targetReport(t string) *TargetReport {
	tr := d.report.Targets[t]
	if tr == nil {
		tr = &TargetReport{Vars: map[string]string{}, Hosts: map[string]map[string]string{}, Credentials: map[string][]string{}}
		d.report.Targets[t] = tr
	}
	return tr
}

func (d *doctor) walkHosts(t string, tr *TargetReport) {
	for _, f := range d.list("targets", t, "hosts") {
		fpath := d.envPath("targets", t, "hosts", f.Name)
		c, isShell := strings.CutSuffix(f.Name, ".sh")
		if f.Type != TypeFile || !isShell || ValidateConsumer(c) != nil {
			d.unknown(fpath, f)
			continue
		}
		if !d.entry(fpath, f, TypeFile, 0o644) {
			continue
		}
		data, ok := d.read("targets", t, "hosts", f.Name)
		if !ok {
			continue
		}
		parsed := d.parse(fpath, HostsFile, data)
		if d.hosts[t] == nil {
			d.hosts[t] = map[string]ShellFile{}
		}
		d.hosts[t][c] = parsed
		tr.Hosts[c] = values(parsed)
		d.shared = append(d.shared, sharedFile{path: fpath, content: data, target: t, hosts: c})
	}
}

func (d *doctor) parse(path string, kind FileKind, data []byte) ShellFile {
	parsed, err := ParseShellFile(path, kind, data)
	var ps Problems
	if errors.As(err, &ps) {
		for _, p := range ps {
			d.problem(p.File, p.Line, p.Key, "%s", p.Msg)
		}
	}
	return parsed
}

func values(f ShellFile) map[string]string {
	out := map[string]string{}
	for _, a := range f.Assignments {
		out[a.Key] = a.Value
	}
	return out
}

func (d *doctor) walkCredentials() {
	for _, e := range d.list("credentials") {
		path := d.envPath("credentials", e.Name)
		t, isTarget := strings.CutPrefix(e.Name, "target-")
		switch {
		case e.Name == AutomationDir:
			if d.entry(path, e, TypeDir, 0o700) {
				d.autoDir = true
				d.walkAutomation()
			}
		case isTarget && e.Type == TypeDir && ValidateTarget(t) == nil:
			if d.entry(path, e, TypeDir, 0o700) {
				d.walkTargetCredentials(t)
			}
		case e.Type == TypeDir && ValidateConsumer(e.Name) == nil:
			if d.entry(path, e, TypeDir, 0o700) {
				names := d.walkCredentialFiles([]string{"credentials", e.Name}, d.declaredCredentials(e.Name))
				d.credDirs[e.Name] = names
				d.report.Credentials[e.Name] = sortedKeys(names)
			}
		default:
			d.unknown(path, e)
		}
	}
}

func (d *doctor) walkTargetCredentials(t string) {
	tr := d.targetReport(t)
	ids := map[string]map[string]bool{}
	for _, e := range d.list("credentials", "target-"+t) {
		path := d.envPath("credentials", "target-"+t, e.Name)
		switch {
		case e.Type == TypeSymlink || strings.HasPrefix(e.Name, TempPrefix):
			d.unknown(path, e)
		case e.Type != TypeDir:
			d.problem(path, 0, "", "a file directly under credentials/target-%s/; target secrets live in an identity directory", t)
			d.inspect(path, e)
		case ValidateIdentity(e.Name) != nil:
			why := "is not a valid identity name"
			if identityPattern.MatchString(e.Name) {
				why = identityRejection(e.Name)
			}
			d.problem(path, 0, "", "identity directory %q %s", e.Name, why)
			d.inspect(path, e)
		default:
			if d.entry(path, e, TypeDir, 0o700) {
				names := d.walkCredentialFiles([]string{"credentials", "target-" + t, e.Name}, d.declaredTargetCredentials(e.Name))
				ids[e.Name] = names
				tr.Credentials[e.Name] = sortedKeys(names)
			}
		}
	}
	d.tcreds[t] = ids
}

// walkCredentialFiles checks the credential files of one directory and keeps
// the values of the declared ones in memory for the leak search.
func (d *doctor) walkCredentialFiles(rel []string, declared map[string]bool) map[string]bool {
	names := map[string]bool{}
	for _, f := range d.list(rel...) {
		path := d.envPath(append(append([]string{}, rel...), f.Name)...)
		if f.Type == TypeFile && ValidateKey(f.Name) != nil && !strings.HasPrefix(f.Name, TempPrefix) {
			d.unknown(path, f)
			continue
		}
		if f.Type == TypeDir && !strings.HasPrefix(f.Name, TempPrefix) {
			d.unknown(path, f)
			continue
		}
		if !d.entry(path, f, TypeFile, 0o600) {
			continue
		}
		names[f.Name] = true
		if f.Size == 0 {
			d.problem(path, 0, "", "empty credential file")
			continue
		}
		if declared[f.Name] {
			if data, ok := d.read(append(append([]string{}, rel...), f.Name)...); ok {
				d.creds = append(d.creds, credValue{name: f.Name, value: data})
			}
		}
	}
	return names
}

func (d *doctor) walkAutomation() {
	dir := []string{"credentials", AutomationDir}
	report := &AutomationReport{}
	d.report.AutomationKey = report
	var private, public []byte
	for _, f := range d.list(dir...) {
		path := d.envPath("credentials", AutomationDir, f.Name)
		switch f.Name {
		case AutomationPrivateKey:
			if d.entry(path, f, TypeFile, 0o600) {
				if data, ok := d.read("credentials", AutomationDir, f.Name); ok {
					private, report.PrivateKey = data, true
				}
			}
		case AutomationPublicKey:
			if d.entry(path, f, TypeFile, 0o644) {
				if data, ok := d.read("credentials", AutomationDir, f.Name); ok {
					public, report.PublicKey = data, true
					d.shared = append(d.shared, sharedFile{path: path, content: data})
				}
			}
		default:
			d.unknown(path, f)
		}
	}
	path := d.envPath("credentials", AutomationDir)
	if !report.PrivateKey || !report.PublicKey {
		var missing []string
		if !report.PrivateKey {
			missing = append(missing, AutomationPrivateKey)
		}
		if !report.PublicKey {
			missing = append(missing, AutomationPublicKey)
		}
		d.problem(path, 0, "", "incomplete automation key pair: %s missing or unreadable", strings.Join(missing, " and "))
		return
	}
	fingerprint, msg := checkAutomationPair(private, public)
	report.Fingerprint = fingerprint
	if msg != "" {
		d.problem(path, 0, "", "%s", msg)
	}
	d.creds = append(d.creds, credValue{name: AutomationPrivateKey, value: private})
}

func (d *doctor) walkState() {
	for _, e := range d.list("state") {
		path := d.envPath("state", e.Name)
		if e.Type != TypeDir || ValidateConsumer(e.Name) != nil {
			d.unknown(path, e)
			continue
		}
		if d.entry(path, e, TypeDir, 0o700) {
			d.walkStateTree([]string{"state", e.Name})
		}
	}
}

// walkStateTree checks the modes of a state holder's tree (directories 0700,
// files 0600) without reading any file.
func (d *doctor) walkStateTree(rel []string) {
	for _, e := range d.list(rel...) {
		child := append(append([]string{}, rel...), e.Name)
		path := d.envPath(child...)
		switch e.Type {
		case TypeDir:
			if d.entry(path, e, TypeDir, 0o700) {
				d.walkStateTree(child)
			}
		default:
			d.entry(path, e, TypeFile, 0o600)
		}
	}
}

// walkBackups checks backups/<consumer>/ only; its content belongs to that
// consumer.
func (d *doctor) walkBackups() {
	for _, e := range d.list("backups") {
		path := d.envPath("backups", e.Name)
		if e.Type != TypeDir || ValidateConsumer(e.Name) != nil {
			d.unknown(path, e)
			continue
		}
		d.entry(path, e, TypeDir, 0o700)
	}
}

func (d *doctor) declaredCredentials(consumer string) map[string]bool {
	out := map[string]bool{}
	for _, decl := range d.decls {
		if decl.Consumer == consumer {
			for _, c := range decl.Credentials {
				out[c.Name] = true
			}
		}
	}
	return out
}

func (d *doctor) declaredTargetCredentials(identity string) map[string]bool {
	out := map[string]bool{}
	for _, decl := range d.decls {
		for _, c := range decl.TargetCredentials {
			if c.Identity == identity {
				out[c.Name] = true
			}
		}
	}
	return out
}

// checkDeclared reports required keys, credentials and key pairs that are
// missing, empty or still a placeholder, and undeclared keys and files as
// warnings.
func (d *doctor) checkDeclared(targets, consumers []string) {
	if d.tree == nil {
		return
	}
	scope := targets
	if len(scope) == 0 {
		scope = d.found
	}
	for _, t := range targets {
		if !contains(d.found, t) {
			d.problem(d.envPath("targets", t), 0, "", "target directory missing; run platformctl bootstrap-config --target %s", t)
		}
	}
	byConsumer := map[string]Declaration{}
	for _, decl := range d.decls {
		byConsumer[decl.Consumer] = decl
		if !contains(consumers, decl.Consumer) {
			consumers = append(consumers, decl.Consumer)
		}
	}
	targetKeys := map[string]bool{}
	var requiredTarget []KeyEntry
	for _, decl := range d.decls {
		for _, k := range decl.Target {
			targetKeys[k.Name] = true
			if k.Required {
				requiredTarget = append(requiredTarget, k)
			}
		}
	}
	for _, c := range consumers {
		if _, ok := d.credDirs[c]; !ok {
			d.problem(d.envPath("credentials", c), 0, "", "missing credentials directory of %s; run platformctl bootstrap-config --consumer %s", c, c)
		}
		decl, declared := byConsumer[c]
		if !declared {
			continue
		}
		for _, cred := range decl.Credentials {
			if cred.Required && !d.credDirs[c][cred.Name] {
				d.problem(d.envPath("credentials", c, cred.Name), 0, cred.Name, "required credential of %s missing; run platformctl credential --consumer %s", c, c)
			}
		}
		for name := range d.credDirs[c] {
			if !d.declaredCredentials(c)[name] {
				d.warn(d.envPath("credentials", c, name), 0, name, "credential not declared by %s", c)
			}
		}
		if decl.AutomationKey && !d.autoDir {
			d.problem(d.envPath("credentials", AutomationDir), 0, "", "%s uses the automation key pair, which is missing; run platformctl bootstrap-config --automation-key (first controller) or --automation-key-import -", c)
		}
	}
	if len(d.decls) > 0 {
		for c := range d.credDirs {
			if _, ok := byConsumer[c]; !ok {
				d.warn(d.envPath("credentials", c), 0, "", "no declaration for consumer %s", c)
			}
		}
	}
	reported := map[string]bool{}
	for _, t := range scope {
		if !contains(d.found, t) {
			continue
		}
		varsPath := d.envPath("targets", t, "vars.sh")
		if vars, ok := d.vars[t]; ok {
			for _, k := range requiredTarget {
				if !reported[t+"/"+k.Name] {
					reported[t+"/"+k.Name] = true
					d.required(varsPath, vars, k.Name)
				}
			}
			if len(d.decls) > 0 {
				for _, a := range vars.Assignments {
					if !targetKeys[a.Key] {
						d.warn(varsPath, a.Line, a.Key, "key not declared by any consumer")
					}
				}
			}
		}
		for _, c := range consumers {
			hostsPath := d.envPath("targets", t, "hosts", c+".sh")
			hosts, ok := d.hosts[t][c]
			if !ok {
				d.problem(hostsPath, 0, "", "missing hosts file of %s; run platformctl bootstrap-config --target %s --consumer %s", c, t, c)
			}
			decl, declared := byConsumer[c]
			if !declared || !ok {
				continue
			}
			hostKeys := map[string]bool{}
			for _, k := range decl.Host {
				hostKeys[k.Name] = true
				if k.Required {
					d.required(hostsPath, hosts, k.Name)
				}
			}
			for _, a := range hosts.Assignments {
				if !hostKeys[a.Key] {
					d.warn(hostsPath, a.Line, a.Key, "key not declared by %s", c)
				}
			}
		}
		if len(d.decls) > 0 {
			for c := range d.hosts[t] {
				if _, ok := byConsumer[c]; !ok {
					d.warn(d.envPath("targets", t, "hosts", c+".sh"), 0, "", "no declaration for consumer %s", c)
				}
			}
		}
		for _, decl := range d.decls {
			for _, tc := range decl.TargetCredentials {
				if tc.Required && !d.tcreds[t][tc.Identity][tc.Name] {
					d.problem(d.envPath("credentials", "target-"+t, tc.Identity, tc.Name), 0, tc.Name,
						"required target credential of %s missing; run platformctl credential --target %s --identity %s", decl.Consumer, t, tc.Identity)
				}
			}
		}
		if len(d.decls) > 0 {
			for id, names := range d.tcreds[t] {
				declared := d.declaredTargetCredentials(id)
				for name := range names {
					if !declared[name] {
						d.warn(d.envPath("credentials", "target-"+t, id, name), 0, name, "target credential not declared by any consumer for identity %s", id)
					}
				}
			}
		}
	}
}

// required reports a required key that is not set, set empty, or still a
// placeholder.
func (d *doctor) required(path string, f ShellFile, key string) {
	for _, a := range f.Assignments {
		if a.Key == key {
			if a.Value == "" {
				d.problem(path, a.Line, key, "required key is empty")
			}
			return
		}
	}
	for _, p := range f.Placeholders {
		if p.Key == key {
			d.problem(path, p.Line, key, "required key is still a placeholder")
			return
		}
	}
	d.problem(path, 0, key, "required key missing")
}

// checkLeaks reports a declared credential name, or the value of a declared
// credential, found in a file that is not secret. The report then withholds
// every value of that file.
func (d *doctor) checkLeaks() {
	names := map[string]bool{}
	for _, decl := range d.decls {
		for _, c := range decl.Credentials {
			names[c.Name] = true
		}
		for _, c := range decl.TargetCredentials {
			names[c.Name] = true
		}
	}
	files := d.shared
	known := map[string]bool{}
	for _, f := range d.shared {
		known[f.path] = true
	}
	for _, path := range d.readable {
		if known[path] || d.tree == nil {
			continue
		}
		known[path] = true
		rel := d.rel(path)
		data, err := d.tree.ReadFile(append([]string{d.opts.Env}, rel[:len(rel)-1]...), rel[len(rel)-1])
		if err != nil {
			d.warn(path, 0, "", "not searched for leaked credentials: %v", err)
			continue
		}
		files = append(files, sharedFile{path: path, content: data})
	}
	for _, f := range files {
		leaked := false
		for _, name := range sortedKeys(names) {
			if wordPattern(name).Match(f.content) {
				leaked = true
				d.problem(f.path, 0, name, "a file readable by others names a declared credential; credentials live only in credentials/")
			}
		}
		for _, c := range d.creds {
			v := bytes.TrimRight(c.value, "\n")
			if len(v) >= minLeakLength && bytes.Contains(f.content, v) {
				leaked = true
				d.problem(f.path, 0, c.name, "a file readable by others holds the value of credential %s", c.name)
			}
		}
		if leaked && f.target != "" {
			tr := d.report.Targets[f.target]
			m := tr.Vars
			if f.hosts != "" {
				m = tr.Hosts[f.hosts]
			}
			for k := range m {
				m[k] = WithheldValue
			}
		}
	}
	d.creds = nil
	d.shared, d.readable = nil, nil
}

func wordPattern(name string) *regexp.Regexp {
	return regexp.MustCompile(`(^|[^A-Za-z0-9_])` + regexp.QuoteMeta(name) + `($|[^A-Za-z0-9_])`)
}

func (d *doctor) finish() {
	if d.tree != nil {
		_ = d.tree.Close()
	}
	for _, list := range []*[]Finding{&d.report.Problems, &d.report.Warnings} {
		l := *list
		sort.SliceStable(l, func(i, j int) bool {
			if l[i].File != l[j].File {
				return l[i].File < l[j].File
			}
			if l[i].Line != l[j].Line {
				return l[i].Line < l[j].Line
			}
			if l[i].Key != l[j].Key {
				return l[i].Key < l[j].Key
			}
			return l[i].Message < l[j].Message
		})
		*list = dedupe(l)
	}
	d.report.OK = len(d.report.Problems) == 0
}

func dedupe(l []Finding) []Finding {
	out := l[:0]
	for i, f := range l {
		if i > 0 && f == l[i-1] {
			continue
		}
		out = append(out, f)
	}
	return out
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
