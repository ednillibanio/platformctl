package sharedroot

import (
	"regexp"
	"strconv"
	"strings"
)

// FileKind is one of the two shared files of a target.
type FileKind int

const (
	// VarsFile is targets/<t>/vars.sh: GOVC_*, TARGET_* and DISPOSABLE_* keys.
	VarsFile FileKind = iota
	// HostsFile is targets/<t>/hosts/<consumer>.sh: HOST_* keys.
	HostsFile
)

// Assignment is one KEY=value line of a shared file.
type Assignment struct {
	Key   string
	Value string
	Line  int
}

// Placeholder is a commented key line (`# KEY=`): the key is not set.
type Placeholder struct {
	Key  string
	Line int
}

// ShellFile is a parsed vars.sh or hosts file.
type ShellFile struct {
	Assignments  []Assignment
	Placeholders []Placeholder
}

// Line grammar of the standard ("File format"), the same as the
// provision-semaphoreui loader (hosts/lib/config.sh, "Key rules") plus the
// `~` rule: an unquoted value has no spaces, shell characters, quotes or `~`;
// a single-quoted value has no `'`; a double-quoted value has no $ ` \ or ".
// Like the loader, an assignment may be followed by whitespace and a comment.
var (
	blankOrComment = regexp.MustCompile(`^\s*(#.*)?$`)
	assignmentLine = regexp.MustCompile("^\\s*(?:export\\s+)?([A-Za-z_][A-Za-z0-9_]*)=" +
		"('[^']*'|\"[^\"$`\\\\]*\"|[^\\s'\"$`;&|<>()\\\\#~]*)" +
		"(?:\\s+#.*)?\\s*$")
	looseKey        = regexp.MustCompile(`^\s*(?:export\s+)?([A-Za-z_][A-Za-z0-9_]*)=`)
	placeholderLine = regexp.MustCompile(`^\s*#\s*(?:export\s+)?([A-Z][A-Z0-9_]*)=`)
)

// ParseShellFile checks every line of a shared file of the given kind and
// returns its assignments and placeholders. It reports every problem: a line
// that is not a comment, blank or literal assignment, a key with an invalid
// name or a prefix the kind does not allow, and a key set twice. Problems
// name the file, line and key, never the value.
func ParseShellFile(file string, kind FileKind, content []byte) (ShellFile, error) {
	var parsed ShellFile
	var problems Problems
	seen := map[string]int{}
	for i, line := range strings.Split(string(content), "\n") {
		n := i + 1
		if strings.ContainsAny(line, "\r\x00") {
			problems = append(problems, Problem{File: file, Line: n, Key: keyOf(line),
				Msg: "carriage return or NUL byte (use LF line endings)"})
			continue
		}
		if blankOrComment.MatchString(line) {
			if m := placeholderLine.FindStringSubmatch(line); m != nil {
				parsed.Placeholders = append(parsed.Placeholders, Placeholder{Key: m[1], Line: n})
			}
			continue
		}
		m := assignmentLine.FindStringSubmatch(line)
		if m == nil {
			problems = append(problems, Problem{File: file, Line: n, Key: keyOf(line),
				Msg: "only KEY=value lines with a literal value are allowed (no declare, printf -v, functions, commands, $ or ` expansion, unquoted ~)"})
			continue
		}
		key := m[1]
		if msg := keyRule(kind, key); msg != "" {
			problems = append(problems, Problem{File: file, Line: n, Key: key, Msg: msg})
			continue
		}
		if first, dup := seen[key]; dup {
			problems = append(problems, Problem{File: file, Line: n, Key: key,
				Msg: "key set twice (first on line " + strconv.Itoa(first) + ")"})
			continue
		}
		seen[key] = n
		parsed.Assignments = append(parsed.Assignments, Assignment{Key: key, Value: unquote(m[2]), Line: n})
	}
	return parsed, problems.Err()
}

// keyRule returns why key may not be set in a file of kind, or "".
func keyRule(kind FileKind, key string) string {
	if ValidateKey(key) != nil {
		return "invalid key name: must match " + keyPattern.String()
	}
	switch kind {
	case VarsFile:
		if !hasTargetPrefix(key) {
			return "vars.sh holds only GOVC_*, TARGET_* and DISPOSABLE_* keys"
		}
	case HostsFile:
		if !strings.HasPrefix(key, "HOST_") {
			return "a hosts file holds only HOST_* keys"
		}
	default:
		return "unknown shared file kind"
	}
	return ""
}

func hasTargetPrefix(key string) bool {
	return strings.HasPrefix(key, "GOVC_") || strings.HasPrefix(key, "TARGET_") || strings.HasPrefix(key, "DISPOSABLE_")
}

func keyOf(line string) string {
	if m := looseKey.FindStringSubmatch(line); m != nil {
		return m[1]
	}
	return ""
}

func unquote(value string) string {
	if len(value) >= 2 && (value[0] == '\'' || value[0] == '"') {
		return value[1 : len(value)-1]
	}
	return value
}
