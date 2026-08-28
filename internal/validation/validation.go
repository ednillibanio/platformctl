package validation

import (
	"fmt"
	"net/netip"
	"regexp"
	"sort"
	"strings"

	"github.com/ednillibanio/platformctl/internal/config"
)

var environmentVariablePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func Bundle(bundle config.Bundle) error {
	var problems []string
	checkHeader := func(kind, version, actualKind, name string) {
		if version != config.APIVersion {
			problems = append(problems, fmt.Sprintf("%s %q: unsupported apiVersion %q", kind, name, version))
		}
		if actualKind != kind {
			problems = append(problems, fmt.Sprintf("%s %q: expected kind %q, got %q", kind, name, kind, actualKind))
		}
		if !config.ValidIdentifier(name) && !(kind == "Defaults" && name == "defaults") {
			problems = append(problems, fmt.Sprintf("%s %q: invalid metadata.name", kind, name))
		}
	}

	checkHeader("Defaults", bundle.Defaults.APIVersion, bundle.Defaults.Kind, bundle.Defaults.Metadata.Name)
	if bundle.Defaults.Metadata.Name != "defaults" {
		problems = append(problems, `Defaults: metadata.name must be "defaults"`)
	}
	checkHeader("Environment", bundle.Environment.APIVersion, bundle.Environment.Kind, bundle.Environment.Metadata.Name)
	if bundle.Environment.Metadata.Name == "" {
		problems = append(problems, "Environment: metadata.name is required")
	}
	checkSecrets("Defaults defaults", bundle.Defaults.Spec.Credentials, &problems)
	checkSecrets("Environment "+bundle.Environment.Metadata.Name, bundle.Environment.Spec.Credentials, &problems)

	profileNames := sortedKeys(bundle.Profiles)
	for _, name := range profileNames {
		profile := bundle.Profiles[name]
		checkHeader("Profile", profile.APIVersion, profile.Kind, name)
		checkSecrets("Profile "+name, profile.Spec.Credentials, &problems)
		if profile.Spec.Extends != "" {
			if _, exists := bundle.Profiles[profile.Spec.Extends]; !exists {
				problems = append(problems, fmt.Sprintf("Profile %q: parent profile %q does not exist", name, profile.Spec.Extends))
			}
		}
	}
	checkProfileCycles(bundle.Profiles, &problems)

	serviceNames := sortedKeys(bundle.Services)
	for _, name := range serviceNames {
		service := bundle.Services[name]
		checkHeader("Service", service.APIVersion, service.Kind, name)
		checkSecrets("Service "+name, service.Spec.Credentials, &problems)
		for index, port := range service.Spec.Ports {
			prefix := fmt.Sprintf("Service %q port[%d]", name, index)
			if port.Port < 1 || port.Port > 65535 {
				problems = append(problems, fmt.Sprintf("%s: port must be between 1 and 65535", prefix))
			}
			if port.Protocol != "tcp" && port.Protocol != "udp" {
				problems = append(problems, fmt.Sprintf("%s: protocol must be tcp or udp", prefix))
			}
			seen := map[string]bool{}
			for _, source := range port.SourceHosts {
				if seen[source] {
					problems = append(problems, fmt.Sprintf("%s: duplicate source_host %q", prefix, source))
					continue
				}
				seen[source] = true
				host, exists := bundle.Hosts[source]
				if !exists {
					problems = append(problems, fmt.Sprintf("%s: source_host %q does not exist", prefix, source))
				} else if len(host.Spec.Addresses) == 0 {
					problems = append(problems, fmt.Sprintf("%s: source_host %q has no addresses", prefix, source))
				}
			}
		}
	}

	hostNames := sortedKeys(bundle.Hosts)
	for _, name := range hostNames {
		host := bundle.Hosts[name]
		checkHeader("Host", host.APIVersion, host.Kind, name)
		checkSecrets("Host "+name, host.Spec.Credentials, &problems)
		if host.Spec.Profile == "" {
			problems = append(problems, fmt.Sprintf("Host %q: profile is required", name))
		} else if _, exists := bundle.Profiles[host.Spec.Profile]; !exists {
			problems = append(problems, fmt.Sprintf("Host %q: profile %q does not exist", name, host.Spec.Profile))
		}
		seenServices := map[string]bool{}
		for _, service := range host.Spec.Services {
			if seenServices[service] {
				problems = append(problems, fmt.Sprintf("Host %q: duplicate service %q", name, service))
				continue
			}
			seenServices[service] = true
			if _, exists := bundle.Services[service]; !exists {
				problems = append(problems, fmt.Sprintf("Host %q: service %q does not exist", name, service))
			}
		}
		seenAddresses := map[string]bool{}
		for _, address := range host.Spec.Addresses {
			if seenAddresses[address] {
				problems = append(problems, fmt.Sprintf("Host %q: duplicate CIDR %q", name, address))
				continue
			}
			seenAddresses[address] = true
			if _, err := netip.ParsePrefix(address); err != nil {
				problems = append(problems, fmt.Sprintf("Host %q: invalid CIDR %q", name, address))
			}
		}
	}

	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("configuration is invalid:\n- %s", strings.Join(problems, "\n- "))
	}
	return nil
}

func checkSecrets(owner string, refs map[string]config.SecretRef, problems *[]string) {
	for name, ref := range refs {
		if !config.ValidIdentifier(name) {
			*problems = append(*problems, fmt.Sprintf("%s: invalid credential name %q", owner, name))
		}
		set := 0
		for _, value := range []string{ref.Env, ref.File, ref.Value} {
			if value != "" {
				set++
			}
		}
		if set != 1 {
			*problems = append(*problems, fmt.Sprintf("%s: credential %q must set exactly one of env, file, or value", owner, name))
		}
		if ref.Env != "" && !environmentVariablePattern.MatchString(ref.Env) {
			*problems = append(*problems, fmt.Sprintf("%s: credential %q has invalid environment variable %q", owner, name, ref.Env))
		}
	}
}

func checkProfileCycles(profiles map[string]config.Profile, problems *[]string) {
	const (
		unseen = iota
		visiting
		done
	)
	state := make(map[string]int)
	var visit func(string)
	visit = func(name string) {
		if state[name] == visiting {
			*problems = append(*problems, fmt.Sprintf("Profile %q: inheritance cycle detected", name))
			return
		}
		if state[name] == done {
			return
		}
		state[name] = visiting
		parent := profiles[name].Spec.Extends
		if _, exists := profiles[parent]; parent != "" && exists {
			visit(parent)
		}
		state[name] = done
	}
	for name := range profiles {
		visit(name)
	}
}

func sortedKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
