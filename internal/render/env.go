package render

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ednillibanio/platformctl/internal/config"
)

// EnvOptions controls the environment-variable projection produced by EncodeEnv.
type EnvOptions struct {
	// Prefix is prepended to every variable name, for example "PM_".
	Prefix string
	// RevealSecrets allows inline credential values to be emitted as literals.
	RevealSecrets bool
}

const envFunction = "__platformctl_export"

var shellNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type envEntry struct {
	origin string // value path or credential name, for error messages
	// Exactly one of literal, envSource, or filePath is meaningful.
	literal   string
	envSource string
	filePath  string
	isEnv     bool
	isFile    bool
}

// EncodeEnv projects a resolved configuration into POSIX shell that exports
// one variable per leaf value and per credential. The output is meant to be
// evaluated (eval or source), never stored.
//
// Credential references are resolved by the evaluating shell, never by
// platformctl: env references are copied from the named variable and file
// references are read with cat. All secrets are read before anything is
// exported, and any failure makes the evaluation return non-zero without
// exporting anything or terminating the calling shell.
func EncodeEnv(cfg config.RenderedConfig, opts EnvOptions) ([]byte, error) {
	if opts.Prefix != "" && !shellNamePattern.MatchString(opts.Prefix) {
		return nil, fmt.Errorf("invalid prefix %q: must be a shell variable name prefix", opts.Prefix)
	}
	entries := map[string]envEntry{}
	if err := flattenValues(entries, opts.Prefix, nil, cfg.Values); err != nil {
		return nil, err
	}
	for _, name := range sortedKeys(cfg.Credentials) {
		ref := cfg.Credentials[name]
		variable, err := envName(opts.Prefix, []string{name})
		if err != nil {
			return nil, fmt.Errorf("credential %q: %w", name, err)
		}
		entry := envEntry{origin: "credential " + strconv.Quote(name)}
		switch {
		case ref.Env != "":
			entry.isEnv, entry.envSource = true, ref.Env
		case ref.File != "":
			if _, err := shellPath(ref.File); err != nil {
				return nil, fmt.Errorf("credential %q: %w", name, err)
			}
			entry.isFile, entry.filePath = true, ref.File
		case ref.Value != "":
			if !opts.RevealSecrets {
				return nil, fmt.Errorf("credential %q has an inline value; pass --reveal-secrets to export it", name)
			}
			entry.literal = ref.Value
		default:
			return nil, fmt.Errorf("credential %q has no env, file, or value", name)
		}
		if err := addEntry(entries, variable, entry); err != nil {
			return nil, err
		}
	}
	return writeEnv(entries), nil
}

func flattenValues(entries map[string]envEntry, prefix string, path []string, values map[string]any) error {
	if len(path) > 0 && len(values) == 0 {
		return fmt.Errorf("value %q is an empty map and cannot be exported", strings.Join(path, "."))
	}
	for _, key := range sortedKeys(values) {
		current := append(append([]string(nil), path...), key)
		dotted := strings.Join(current, ".")
		if nested, ok := values[key].(map[string]any); ok {
			if err := flattenValues(entries, prefix, current, nested); err != nil {
				return err
			}
			continue
		}
		literal, err := scalarString(values[key])
		if err != nil {
			return fmt.Errorf("value %q: %w", dotted, err)
		}
		variable, err := envName(prefix, current)
		if err != nil {
			return fmt.Errorf("value %q: %w", dotted, err)
		}
		if err := addEntry(entries, variable, envEntry{origin: "value " + strconv.Quote(dotted), literal: literal}); err != nil {
			return err
		}
	}
	return nil
}

func scalarString(value any) (string, error) {
	switch typed := value.(type) {
	case string:
		return typed, nil
	case bool:
		return strconv.FormatBool(typed), nil
	case int:
		return strconv.Itoa(typed), nil
	case int64:
		return strconv.FormatInt(typed, 10), nil
	case uint64:
		return strconv.FormatUint(typed, 10), nil
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64), nil
	case nil:
		return "", fmt.Errorf("null cannot be exported")
	case []any:
		return "", fmt.Errorf("lists cannot be exported; use render --format json")
	default:
		return "", fmt.Errorf("unsupported type %T", value)
	}
}

func envName(prefix string, path []string) (string, error) {
	segments := make([]string, len(path))
	for i, segment := range path {
		segment = strings.NewReplacer("-", "_", ".", "_").Replace(segment)
		segments[i] = strings.ToUpper(segment)
	}
	name := prefix + strings.Join(segments, "_")
	if !shellNamePattern.MatchString(name) {
		return "", fmt.Errorf("%q is not a valid shell variable name", name)
	}
	return name, nil
}

func addEntry(entries map[string]envEntry, variable string, entry envEntry) error {
	if existing, ok := entries[variable]; ok {
		return fmt.Errorf("%s and %s both export %s", existing.origin, entry.origin, variable)
	}
	entries[variable] = entry
	return nil
}

// shellPath returns a shell word for a secret file reference. Absolute paths
// are single-quoted; "~/" paths are resolved against HOME by the evaluating
// shell, so one configuration works on hosts with different home directories.
func shellPath(path string) (string, error) {
	switch {
	case strings.HasPrefix(path, "/"):
		return shellQuote(path), nil
	case strings.HasPrefix(path, "~/") && len(path) > 2:
		return `"$HOME"/` + shellQuote(path[2:]), nil
	default:
		return "", fmt.Errorf("file reference %q must be absolute or start with ~/ to be exported", path)
	}
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func writeEnv(entries map[string]envEntry) []byte {
	var out strings.Builder
	out.WriteString("# Generated by platformctl export. Evaluate it with eval or source; never store it.\n")
	out.WriteString(envFunction + "() {\n")
	names := sortedKeys(entries)
	locals := map[string]string{}
	var localNames []string
	for _, name := range names {
		if entry := entries[name]; entry.isEnv || entry.isFile {
			local := fmt.Sprintf("__platformctl_s%d", len(localNames)+1)
			locals[name] = local
			localNames = append(localNames, local)
		}
	}
	if len(localNames) > 0 {
		out.WriteString("  local " + strings.Join(localNames, " ") + "\n")
	}
	for _, name := range names {
		entry := entries[name]
		local, ok := locals[name]
		if !ok {
			continue
		}
		if entry.isFile {
			path, _ := shellPath(entry.filePath)
			message := shellQuote(fmt.Sprintf("platformctl export: cannot read secret file %s for %s", entry.filePath, name))
			fmt.Fprintf(&out, "  %s=\"$(cat -- %s)\" || { printf '%%s\\n' %s >&2; return 1; }\n", local, path, message)
			continue
		}
		message := shellQuote(fmt.Sprintf("platformctl export: environment variable %s is not set (for %s)", entry.envSource, name))
		fmt.Fprintf(&out, "  if [ -z \"${%s:-}\" ]; then printf '%%s\\n' %s >&2; return 1; fi\n", entry.envSource, message)
		fmt.Fprintf(&out, "  %s=\"$%s\"\n", local, entry.envSource)
	}
	if len(names) == 0 {
		out.WriteString("  :\n")
	}
	for _, name := range names {
		if local, ok := locals[name]; ok {
			fmt.Fprintf(&out, "  export %s=\"$%s\"\n", name, local)
			continue
		}
		fmt.Fprintf(&out, "  export %s=%s\n", name, shellQuote(entries[name].literal))
	}
	out.WriteString("}\n")
	out.WriteString("if " + envFunction + "; then unset -f " + envFunction + "; else unset -f " + envFunction + "; false; fi\n")
	return []byte(out.String())
}
