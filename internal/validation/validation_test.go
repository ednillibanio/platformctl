package validation

import (
	"strings"
	"testing"

	"github.com/ednillibanio/platformctl/internal/config"
)

func TestValidationFailures(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*config.Bundle)
		want   string
	}{
		{"invalid cidr", func(b *config.Bundle) {
			h := b.Hosts["app"]
			h.Spec.Addresses = []string{"not-cidr"}
			b.Hosts["app"] = h
		}, "invalid CIDR"},
		{"invalid port", func(b *config.Bundle) { s := b.Services["web"]; s.Spec.Ports[0].Port = 70000; b.Services["web"] = s }, "port must be between"},
		{"invalid protocol", func(b *config.Bundle) {
			s := b.Services["web"]
			s.Spec.Ports[0].Protocol = "sctp"
			b.Services["web"] = s
		}, "protocol must be"},
		{"missing host", func(b *config.Bundle) {
			s := b.Services["web"]
			s.Spec.Ports[0].SourceHosts = []string{"ghost"}
			b.Services["web"] = s
		}, `source_host "ghost" does not exist`},
		{"missing profile", func(b *config.Bundle) { h := b.Hosts["app"]; h.Spec.Profile = "ghost"; b.Hosts["app"] = h }, `profile "ghost" does not exist`},
		{"profile cycle", func(b *config.Bundle) { p := b.Profiles["base"]; p.Spec.Extends = "child"; b.Profiles["base"] = p }, "inheritance cycle"},
		{"inline credential", func(b *config.Bundle) {
			h := b.Hosts["app"]
			h.Spec.Credentials["password"] = config.SecretRef{}
			b.Hosts["app"] = h
		}, "must set exactly one"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bundle := validBundle()
			test.mutate(&bundle)
			err := Bundle(bundle)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q, got %v", test.want, err)
			}
		})
	}
}

func validBundle() config.Bundle {
	layer := config.LayerSpec{Values: map[string]any{}, Credentials: map[string]config.SecretRef{}}
	return config.Bundle{
		Defaults:    config.Defaults{APIVersion: config.APIVersion, Kind: "Defaults", Metadata: config.Metadata{Name: "defaults"}, Spec: layer},
		Environment: config.Environment{APIVersion: config.APIVersion, Kind: "Environment", Metadata: config.Metadata{Name: "lab"}, Spec: layer},
		Profiles: map[string]config.Profile{
			"base":  {APIVersion: config.APIVersion, Kind: "Profile", Metadata: config.Metadata{Name: "base"}, Spec: config.ProfileSpec{LayerSpec: layer}},
			"child": {APIVersion: config.APIVersion, Kind: "Profile", Metadata: config.Metadata{Name: "child"}, Spec: config.ProfileSpec{Extends: "base", LayerSpec: layer}},
		},
		Services: map[string]config.Service{
			"web": {
				APIVersion: config.APIVersion, Kind: "Service", Metadata: config.Metadata{Name: "web"},
				Spec: config.ServiceSpec{LayerSpec: layer, Ports: []config.Port{{Port: 443, Protocol: "tcp", SourceHosts: []string{"source"}}}},
			},
		},
		Hosts: map[string]config.Host{
			"source": {
				APIVersion: config.APIVersion, Kind: "Host", Metadata: config.Metadata{Name: "source"},
				Spec: config.HostSpec{Profile: "base", Addresses: []string{"192.0.2.1/32"}, LayerSpec: layer},
			},
			"app": {
				APIVersion: config.APIVersion, Kind: "Host", Metadata: config.Metadata{Name: "app"},
				Spec: config.HostSpec{
					Profile: "child", Services: []string{"web"}, Addresses: []string{"192.0.2.2/32"},
					LayerSpec: config.LayerSpec{Values: map[string]any{}, Credentials: map[string]config.SecretRef{}},
				},
			},
		},
	}
}
