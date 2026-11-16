package sharedroot

import (
	"bytes"
	"errors"
	"io"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// DeclarationVersion is the current major version of shared-root.yaml. The
// standard lets platformctl also accept the previous major version, with a
// deprecation warning, during an announced transition; version 1 has none.
const DeclarationVersion = 1

// Declaration is a parsed shared-root.yaml ("Key declaration").
type Declaration struct {
	Version           int
	Consumer          string
	AutomationKey     bool
	Target            []KeyEntry
	Host              []KeyEntry
	Credentials       []CredentialEntry
	TargetCredentials []TargetCredentialEntry
	// Warnings are non-fatal findings, such as a deprecated version.
	Warnings []string
}

// KeyEntry is a target or host key.
type KeyEntry struct {
	Name        string
	Required    bool
	Description string
	Example     string
	Line        int
}

// CredentialEntry is a consumer credential file.
type CredentialEntry struct {
	Name        string
	Required    bool
	Description string
	Line        int
}

// TargetCredentialEntry is a target credential file of one identity.
type TargetCredentialEntry struct {
	Name        string
	Identity    string
	Required    bool
	Description string
	Line        int
}

// ParseDeclaration parses a shared-root.yaml strictly: one document, no
// unknown fields, every required field present with its type, names that
// match the standard, no duplicates and the prefixes of each section. It
// reports every problem with file, line and field or key; messages never hold
// a value.
func ParseDeclaration(file string, content []byte) (Declaration, error) {
	p := &declParser{file: file}
	root, err := p.document(content)
	if err != nil {
		return Declaration{}, err
	}
	decl := p.top(root)
	return decl, p.problems.Err()
}

type declParser struct {
	file     string
	problems Problems
}

func (p *declParser) add(line int, key, msg string) {
	p.problems = append(p.problems, Problem{File: p.file, Line: line, Key: key, Msg: msg})
}

func (p *declParser) document(content []byte) (*yaml.Node, error) {
	dec := yaml.NewDecoder(bytes.NewReader(content))
	var doc yaml.Node
	if err := dec.Decode(&doc); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, Problems{{File: p.file, Msg: "empty declaration"}}
		}
		return nil, Problems{{File: p.file, Msg: "invalid YAML: " + yamlMessage(err)}}
	}
	var extra yaml.Node
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, Problems{{File: p.file, Line: extra.Line, Msg: "only one YAML document is allowed"}}
	}
	root := &doc
	if root.Kind == yaml.DocumentNode && len(root.Content) == 1 {
		root = root.Content[0]
	}
	if root.Kind != yaml.MappingNode {
		return nil, Problems{{File: p.file, Line: root.Line, Msg: "the declaration must be a mapping"}}
	}
	return root, nil
}

// yamlMessage keeps the location of a YAML syntax error and drops any quoted
// content, so no value is printed.
func yamlMessage(err error) string {
	msg := strings.TrimPrefix(err.Error(), "yaml: ")
	if i := strings.IndexAny(msg, "`\"'"); i >= 0 {
		msg = strings.TrimSpace(msg[:i])
	}
	return msg
}

var topFields = []string{"version", "consumer", "automation_key", "target", "host", "credentials", "target_credentials"}

func (p *declParser) top(root *yaml.Node) Declaration {
	var decl Declaration
	fields := p.fields(root, "", topFields)
	if n, ok := fields["version"]; !ok {
		p.add(root.Line, "version", "required field is missing")
	} else if v, ok := p.integer(n, "version"); ok {
		decl.Version = v
		switch {
		case v == DeclarationVersion:
		case v == DeclarationVersion-1 && v >= 1:
			decl.Warnings = append(decl.Warnings, p.file+": version "+strconv.Itoa(v)+" is deprecated; move to version "+strconv.Itoa(DeclarationVersion))
		default:
			p.add(n.Line, "version", "unsupported version "+strconv.Itoa(v)+"; supported: "+strconv.Itoa(DeclarationVersion))
		}
	}
	if n, ok := fields["consumer"]; !ok {
		p.add(root.Line, "consumer", "required field is missing")
	} else if v, ok := p.str(n, "consumer"); ok {
		if err := ValidateConsumer(v); err != nil {
			p.add(n.Line, "consumer", "invalid consumer name: must match "+consumerPattern.String())
		} else {
			decl.Consumer = v
		}
	}
	if n, ok := fields["automation_key"]; ok {
		decl.AutomationKey, _ = p.boolean(n, "automation_key")
	}
	if n, ok := fields["target"]; ok {
		decl.Target = p.keyEntries(n, "target", hasTargetPrefix, "target keys start with GOVC_, TARGET_ or DISPOSABLE_")
	}
	if n, ok := fields["host"]; ok {
		decl.Host = p.keyEntries(n, "host", func(k string) bool { return strings.HasPrefix(k, "HOST_") }, "host keys start with HOST_")
	}
	if n, ok := fields["credentials"]; ok {
		decl.Credentials = p.credentialEntries(n)
	}
	if n, ok := fields["target_credentials"]; ok {
		decl.TargetCredentials = p.targetCredentialEntries(n)
	}
	inTarget := map[string]bool{}
	for _, e := range decl.Target {
		inTarget[e.Name] = true
	}
	for _, e := range decl.Host {
		if inTarget[e.Name] {
			p.add(e.Line, e.Name, "declared in both target and host")
		}
	}
	return decl
}

// fields returns the values of a mapping by key and reports unknown and
// duplicate keys.
func (p *declParser) fields(n *yaml.Node, where string, allowed []string) map[string]*yaml.Node {
	out := map[string]*yaml.Node{}
	for i := 0; i+1 < len(n.Content); i += 2 {
		k, v := n.Content[i], n.Content[i+1]
		name := k.Value
		label := name
		if where != "" {
			label = where + "." + name
		}
		if k.Kind != yaml.ScalarNode || !contains(allowed, name) {
			p.add(k.Line, label, "unknown field")
			continue
		}
		if _, dup := out[name]; dup {
			p.add(k.Line, label, "field set twice")
			continue
		}
		out[name] = v
	}
	return out
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func (p *declParser) scalar(n *yaml.Node, field, tag, kind string) bool {
	if n.Kind != yaml.ScalarNode || n.Tag != tag {
		p.add(n.Line, field, "must be "+kind)
		return false
	}
	return true
}

func (p *declParser) integer(n *yaml.Node, field string) (int, bool) {
	if !p.scalar(n, field, "!!int", "an integer") {
		return 0, false
	}
	v, err := strconv.Atoi(n.Value)
	if err != nil {
		p.add(n.Line, field, "must be an integer")
		return 0, false
	}
	return v, true
}

func (p *declParser) boolean(n *yaml.Node, field string) (bool, bool) {
	if !p.scalar(n, field, "!!bool", "true or false") {
		return false, false
	}
	var v bool
	if err := n.Decode(&v); err != nil {
		p.add(n.Line, field, "must be true or false")
		return false, false
	}
	return v, true
}

func (p *declParser) str(n *yaml.Node, field string) (string, bool) {
	if !p.scalar(n, field, "!!str", "a string") {
		return "", false
	}
	return n.Value, true
}

func (p *declParser) line(n *yaml.Node, field string) (string, bool) {
	v, ok := p.str(n, field)
	if !ok {
		return "", false
	}
	if strings.ContainsAny(v, "\r\n") {
		p.add(n.Line, field, "must be one line")
		return "", false
	}
	return v, true
}

// entry is the common part of every list entry.
type entry struct {
	name        string
	required    bool
	description string
	fields      map[string]*yaml.Node
	line        int
}

func (p *declParser) entries(n *yaml.Node, section string, allowed []string) []entry {
	if n.Kind != yaml.SequenceNode {
		p.add(n.Line, section, "must be a list")
		return nil
	}
	var out []entry
	for _, item := range n.Content {
		if item.Kind != yaml.MappingNode {
			p.add(item.Line, section, "each entry must be a mapping")
			continue
		}
		fields := p.fields(item, section, allowed)
		e := entry{fields: fields, line: item.Line}
		label := section + ".name"
		if v, ok := fields["name"]; !ok {
			p.add(item.Line, label, "required field is missing")
		} else if s, ok := p.str(v, label); ok {
			if ValidateKey(s) != nil {
				p.add(v.Line, label, "invalid name: must match "+keyPattern.String())
			} else {
				e.name = s
			}
		}
		key := e.name
		if key == "" {
			key = section
		}
		if v, ok := fields["required"]; !ok {
			p.add(item.Line, key+".required", "required field is missing")
		} else {
			e.required, _ = p.boolean(v, key+".required")
		}
		if v, ok := fields["description"]; !ok {
			p.add(item.Line, key+".description", "required field is missing")
		} else if s, ok := p.line(v, key+".description"); ok {
			if strings.TrimSpace(s) == "" {
				p.add(v.Line, key+".description", "must not be empty")
			}
			e.description = s
		}
		out = append(out, e)
	}
	return out
}

var keyFields = []string{"name", "required", "description", "example"}

func (p *declParser) keyEntries(n *yaml.Node, section string, prefixOK func(string) bool, prefixMsg string) []KeyEntry {
	var out []KeyEntry
	seen := map[string]int{}
	for _, e := range p.entries(n, section, keyFields) {
		var example string
		if v, ok := e.fields["example"]; ok {
			example, _ = p.line(v, e.name+".example")
		}
		if e.name == "" {
			continue
		}
		if !prefixOK(e.name) {
			p.add(e.line, e.name, prefixMsg)
			continue
		}
		if first, dup := seen[e.name]; dup {
			p.add(e.line, e.name, "declared twice in "+section+" (first on line "+strconv.Itoa(first)+")")
			continue
		}
		seen[e.name] = e.line
		out = append(out, KeyEntry{Name: e.name, Required: e.required, Description: e.description, Example: example, Line: e.line})
	}
	return out
}

func (p *declParser) credentialEntries(n *yaml.Node) []CredentialEntry {
	var out []CredentialEntry
	seen := map[string]int{}
	for _, e := range p.entries(n, "credentials", []string{"name", "required", "description"}) {
		if e.name == "" {
			continue
		}
		if first, dup := seen[e.name]; dup {
			p.add(e.line, e.name, "declared twice in credentials (first on line "+strconv.Itoa(first)+")")
			continue
		}
		seen[e.name] = e.line
		out = append(out, CredentialEntry{Name: e.name, Required: e.required, Description: e.description, Line: e.line})
	}
	return out
}

func (p *declParser) targetCredentialEntries(n *yaml.Node) []TargetCredentialEntry {
	var out []TargetCredentialEntry
	seen := map[string]int{}
	for _, e := range p.entries(n, "target_credentials", []string{"name", "identity", "required", "description"}) {
		var identity string
		if v, ok := e.fields["identity"]; !ok {
			p.add(e.line, e.name+".identity", "required field is missing")
		} else if s, ok := p.str(v, e.name+".identity"); ok {
			if !identityPattern.MatchString(s) {
				p.add(v.Line, e.name+".identity", "invalid identity name: must match "+identityPattern.String())
			} else if why := identityRejection(s); why != "" {
				p.add(v.Line, e.name+".identity", "identity "+why)
			} else {
				identity = s
			}
		}
		if e.name == "" || identity == "" {
			continue
		}
		pair := identity + "/" + e.name
		if first, dup := seen[pair]; dup {
			p.add(e.line, e.name, "declared twice for the same identity (first on line "+strconv.Itoa(first)+")")
			continue
		}
		seen[pair] = e.line
		out = append(out, TargetCredentialEntry{Name: e.name, Identity: identity, Required: e.required, Description: e.description, Line: e.line})
	}
	return out
}
