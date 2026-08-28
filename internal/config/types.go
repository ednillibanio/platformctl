package config

const APIVersion = "platformctl.io/v1alpha1"

type Metadata struct {
	Name string `yaml:"name" json:"name"`
}

type SecretRef struct {
	Env   string `yaml:"env,omitempty" json:"env,omitempty"`
	File  string `yaml:"file,omitempty" json:"file,omitempty"`
	Value string `yaml:"value,omitempty" json:"value,omitempty"`
}

type LayerSpec struct {
	Values      map[string]any       `yaml:"values,omitempty" json:"values,omitempty"`
	Credentials map[string]SecretRef `yaml:"credentials,omitempty" json:"credentials,omitempty"`
}

type Defaults struct {
	APIVersion string    `yaml:"apiVersion" json:"apiVersion"`
	Kind       string    `yaml:"kind" json:"kind"`
	Metadata   Metadata  `yaml:"metadata" json:"metadata"`
	Spec       LayerSpec `yaml:"spec" json:"spec"`
}

type Environment struct {
	APIVersion string    `yaml:"apiVersion" json:"apiVersion"`
	Kind       string    `yaml:"kind" json:"kind"`
	Metadata   Metadata  `yaml:"metadata" json:"metadata"`
	Spec       LayerSpec `yaml:"spec" json:"spec"`
}

type ProfileSpec struct {
	Extends   string `yaml:"extends,omitempty" json:"extends,omitempty"`
	LayerSpec `yaml:",inline" json:",inline"`
}

type Profile struct {
	APIVersion string      `yaml:"apiVersion" json:"apiVersion"`
	Kind       string      `yaml:"kind" json:"kind"`
	Metadata   Metadata    `yaml:"metadata" json:"metadata"`
	Spec       ProfileSpec `yaml:"spec" json:"spec"`
}

type Port struct {
	Port        int      `yaml:"port" json:"port"`
	Protocol    string   `yaml:"protocol" json:"protocol"`
	SourceHosts []string `yaml:"source_hosts,omitempty" json:"source_hosts,omitempty"`
}

type ServiceSpec struct {
	Ports     []Port `yaml:"ports,omitempty" json:"ports,omitempty"`
	LayerSpec `yaml:",inline" json:",inline"`
}

type Service struct {
	APIVersion string      `yaml:"apiVersion" json:"apiVersion"`
	Kind       string      `yaml:"kind" json:"kind"`
	Metadata   Metadata    `yaml:"metadata" json:"metadata"`
	Spec       ServiceSpec `yaml:"spec" json:"spec"`
}

type HostSpec struct {
	Profile   string   `yaml:"profile" json:"profile"`
	Services  []string `yaml:"services,omitempty" json:"services,omitempty"`
	Addresses []string `yaml:"addresses,omitempty" json:"addresses,omitempty"`
	LayerSpec `yaml:",inline" json:",inline"`
}

type Host struct {
	APIVersion string   `yaml:"apiVersion" json:"apiVersion"`
	Kind       string   `yaml:"kind" json:"kind"`
	Metadata   Metadata `yaml:"metadata" json:"metadata"`
	Spec       HostSpec `yaml:"spec" json:"spec"`
}

type Bundle struct {
	Defaults    Defaults
	Environment Environment
	Profiles    map[string]Profile
	Services    map[string]Service
	Hosts       map[string]Host
}

type RenderedPort struct {
	Service  string   `yaml:"service" json:"service"`
	Port     int      `yaml:"port" json:"port"`
	Protocol string   `yaml:"protocol" json:"protocol"`
	Sources  []string `yaml:"sources,omitempty" json:"sources,omitempty"`
}

type RenderedConfig struct {
	Values      map[string]any       `yaml:"values,omitempty" json:"values,omitempty"`
	Credentials map[string]SecretRef `yaml:"credentials,omitempty" json:"credentials,omitempty"`
}

type RenderedHost struct {
	Name      string         `yaml:"name" json:"name"`
	Addresses []string       `yaml:"addresses,omitempty" json:"addresses,omitempty"`
	Profile   string         `yaml:"profile" json:"profile"`
	Services  []string       `yaml:"services,omitempty" json:"services,omitempty"`
	Config    RenderedConfig `yaml:"config" json:"config"`
	Ports     []RenderedPort `yaml:"ports,omitempty" json:"ports,omitempty"`
}

type RenderedEnvironment struct {
	APIVersion string         `yaml:"apiVersion" json:"apiVersion"`
	Kind       string         `yaml:"kind" json:"kind"`
	Metadata   Metadata       `yaml:"metadata" json:"metadata"`
	Hosts      []RenderedHost `yaml:"hosts" json:"hosts"`
}
