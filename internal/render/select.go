package render

import (
	"fmt"

	"github.com/ednillibanio/platformctl/internal/config"
	"github.com/ednillibanio/platformctl/internal/merge"
	"github.com/ednillibanio/platformctl/internal/validation"
)

// ResolveServices merges defaults, the environment, and each named service in
// the given order, without involving any host. It serves consumers that are
// not host-centric, such as the platform-modules tool chain.
func ResolveServices(bundle config.Bundle, services []string) (config.RenderedConfig, error) {
	if err := validation.Bundle(bundle); err != nil {
		return config.RenderedConfig{}, err
	}
	values := merge.Maps(bundle.Defaults.Spec.Values, bundle.Environment.Spec.Values)
	credentials := mergeCredentials(bundle.Defaults.Spec.Credentials, bundle.Environment.Spec.Credentials)
	seen := map[string]bool{}
	for _, name := range services {
		if seen[name] {
			return config.RenderedConfig{}, fmt.Errorf("service %q selected more than once", name)
		}
		seen[name] = true
		service, ok := bundle.Services[name]
		if !ok {
			return config.RenderedConfig{}, fmt.Errorf("service %q not found in environment %q", name, bundle.Environment.Metadata.Name)
		}
		values = merge.Maps(values, service.Spec.Values)
		credentials = mergeCredentials(credentials, service.Spec.Credentials)
	}
	return config.RenderedConfig{Values: values, Credentials: credentials}, nil
}

// HostConfig returns the resolved configuration of one host from a rendered
// environment.
func HostConfig(env config.RenderedEnvironment, name string) (config.RenderedConfig, error) {
	for _, host := range env.Hosts {
		if host.Name == name {
			return host.Config, nil
		}
	}
	return config.RenderedConfig{}, fmt.Errorf("host %q not found in environment %q", name, env.Metadata.Name)
}
