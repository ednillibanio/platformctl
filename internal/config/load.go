package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

var identifierPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

func ValidIdentifier(value string) bool {
	return identifierPattern.MatchString(value)
}

func Load(root, env string) (Bundle, error) {
	if !ValidIdentifier(env) {
		return Bundle{}, fmt.Errorf("invalid environment name %q", env)
	}
	var bundle Bundle
	if err := decodeFile(filepath.Join(root, "defaults.yaml"), &bundle.Defaults); err != nil {
		return Bundle{}, err
	}
	envRoot := filepath.Join(root, "environments", env)
	if err := decodeFile(filepath.Join(envRoot, "environment.yaml"), &bundle.Environment); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Bundle{}, fmt.Errorf("environment %q does not exist", env)
		}
		return Bundle{}, err
	}
	if bundle.Environment.Metadata.Name != env {
		return Bundle{}, fmt.Errorf(
			"%s: metadata.name %q must match requested environment %q",
			filepath.Join(envRoot, "environment.yaml"),
			bundle.Environment.Metadata.Name,
			env,
		)
	}
	var err error
	bundle.Profiles, err = loadNamed[Profile](filepath.Join(envRoot, "profiles"))
	if err != nil {
		return Bundle{}, err
	}
	bundle.Services, err = loadNamed[Service](filepath.Join(envRoot, "services"))
	if err != nil {
		return Bundle{}, err
	}
	bundle.Hosts, err = loadNamed[Host](filepath.Join(envRoot, "hosts"))
	if err != nil {
		return Bundle{}, err
	}
	return bundle, nil
}

func decodeFile(path string, target any) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()
	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("parse %s: multiple YAML documents are not allowed", path)
		}
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

type namedDocument interface {
	Profile | Service | Host
}

func loadNamed[T namedDocument](dir string) (map[string]T, error) {
	result := make(map[string]T)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		var doc T
		path := filepath.Join(dir, entry.Name())
		if err := decodeFile(path, &doc); err != nil {
			return nil, err
		}
		name := documentName(any(doc))
		stem := strings.TrimSuffix(entry.Name(), ".yaml")
		if name != stem {
			return nil, fmt.Errorf("%s: metadata.name %q must match filename %q", path, name, stem)
		}
		if _, exists := result[name]; exists {
			return nil, fmt.Errorf("duplicate document name %q in %s", name, dir)
		}
		result[name] = doc
	}
	return result, nil
}

func documentName(doc any) string {
	switch value := doc.(type) {
	case Profile:
		return value.Metadata.Name
	case Service:
		return value.Metadata.Name
	case Host:
		return value.Metadata.Name
	default:
		return ""
	}
}
