package render

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/ednillibanio/platformctl/internal/config"
	"github.com/ednillibanio/platformctl/internal/merge"
	"github.com/ednillibanio/platformctl/internal/validation"
	"gopkg.in/yaml.v3"
)

func Resolve(bundle config.Bundle) (config.RenderedEnvironment, error) {
	if err := validation.Bundle(bundle); err != nil {
		return config.RenderedEnvironment{}, err
	}
	result := config.RenderedEnvironment{
		APIVersion: config.APIVersion,
		Kind:       "RenderedEnvironment",
		Metadata:   config.Metadata{Name: bundle.Environment.Metadata.Name},
		Hosts:      make([]config.RenderedHost, 0, len(bundle.Hosts)),
	}
	hostNames := sortedKeys(bundle.Hosts)
	for _, hostName := range hostNames {
		host := bundle.Hosts[hostName]
		values := merge.Maps(bundle.Defaults.Spec.Values, bundle.Environment.Spec.Values)
		credentials := mergeCredentials(bundle.Defaults.Spec.Credentials, bundle.Environment.Spec.Credentials)
		for _, profile := range profileChain(bundle.Profiles, host.Spec.Profile) {
			values = merge.Maps(values, profile.Spec.Values)
			credentials = mergeCredentials(credentials, profile.Spec.Credentials)
		}
		var ports []config.RenderedPort
		for _, serviceName := range host.Spec.Services {
			service := bundle.Services[serviceName]
			values = merge.Maps(values, service.Spec.Values)
			credentials = mergeCredentials(credentials, service.Spec.Credentials)
			for _, port := range service.Spec.Ports {
				sources := resolveSources(bundle.Hosts, port.SourceHosts)
				ports = append(ports, config.RenderedPort{
					Service: serviceName, Port: port.Port, Protocol: port.Protocol, Sources: sources,
				})
			}
		}
		values = merge.Maps(values, host.Spec.Values)
		credentials = mergeCredentials(credentials, host.Spec.Credentials)
		result.Hosts = append(result.Hosts, config.RenderedHost{
			Name: hostName, Addresses: cloneStrings(host.Spec.Addresses),
			Profile: host.Spec.Profile, Services: cloneStrings(host.Spec.Services),
			Config: config.RenderedConfig{Values: values, Credentials: credentials},
			Ports:  ports,
		})
	}
	return result, nil
}

func Encode(value config.RenderedEnvironment, format string) ([]byte, error) {
	switch format {
	case "", "yaml":
		data, err := yaml.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("encode YAML: %w", err)
		}
		return data, nil
	case "json":
		data, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("encode JSON: %w", err)
		}
		return append(data, '\n'), nil
	default:
		return nil, fmt.Errorf("unsupported format %q; expected yaml or json", format)
	}
}

func profileChain(profiles map[string]config.Profile, name string) []config.Profile {
	var reverse []config.Profile
	for name != "" {
		profile := profiles[name]
		reverse = append(reverse, profile)
		name = profile.Spec.Extends
	}
	result := make([]config.Profile, len(reverse))
	for i := range reverse {
		result[len(reverse)-1-i] = reverse[i]
	}
	return result
}

func mergeCredentials(base, overlay map[string]config.SecretRef) map[string]config.SecretRef {
	result := make(map[string]config.SecretRef, len(base)+len(overlay))
	for key, value := range base {
		result[key] = value
	}
	for key, value := range overlay {
		result[key] = value
	}
	return result
}

func resolveSources(hosts map[string]config.Host, names []string) []string {
	seen := map[string]bool{}
	for _, name := range names {
		for _, address := range hosts[name].Spec.Addresses {
			seen[address] = true
		}
	}
	result := make([]string, 0, len(seen))
	for address := range seen {
		result = append(result, address)
	}
	sort.Strings(result)
	return result
}

func cloneStrings(source []string) []string {
	return append([]string(nil), source...)
}

func sortedKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
