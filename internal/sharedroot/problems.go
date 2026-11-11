package sharedroot

import (
	"fmt"
	"sort"
	"strings"
)

// Problem is one finding in a shared-root input, located by file, line and
// key. Its message never holds a value that could be a secret.
type Problem struct {
	File string
	Line int
	Key  string
	Msg  string
}

func (p Problem) String() string {
	var b strings.Builder
	b.WriteString(p.File)
	if p.Line > 0 {
		fmt.Fprintf(&b, ":%d", p.Line)
	}
	b.WriteString(": ")
	if p.Key != "" {
		fmt.Fprintf(&b, "%s: ", p.Key)
	}
	b.WriteString(p.Msg)
	return b.String()
}

// Problems is every finding of one input, as an error. It maps to ExitProblem.
type Problems []Problem

func (ps Problems) Error() string {
	lines := make([]string, len(ps))
	for i, p := range ps {
		lines[i] = p.String()
	}
	return strings.Join(lines, "\n")
}

// Err returns ps sorted by file and line as an error, or nil when empty.
func (ps Problems) Err() error {
	if len(ps) == 0 {
		return nil
	}
	sort.SliceStable(ps, func(i, j int) bool {
		if ps[i].File != ps[j].File {
			return ps[i].File < ps[j].File
		}
		return ps[i].Line < ps[j].Line
	})
	return ps
}
