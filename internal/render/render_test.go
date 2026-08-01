package render

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/ednillibanio/platformctl/internal/config"
)

func TestResolvePrecedenceInheritanceSourcesAndCredentials(t *testing.T) {
	bundle := exampleBundle(t)
	bundle.Defaults.Spec.Values["precedence"] = "defaults"
	bundle.Environment.Spec.Values["precedence"] = "environment"
	base := bundle.Profiles["base"]
	base.Spec.Values["precedence"] = "base-profile"
	bundle.Profiles["base"] = base
	application := bundle.Profiles["application"]
	application.Spec.Values["precedence"] = "profile"
	bundle.Profiles["application"] = application
	httpService := bundle.Services["http"]
	httpService.Spec.Values["precedence"] = "service"
	httpService.Spec.Values["service-order"] = "first"
	bundle.Services["http"] = httpService
	bundle.Services["metrics"] = config.Service{
		APIVersion: config.APIVersion,
		Kind:       "Service",
		Metadata:   config.Metadata{Name: "metrics"},
		Spec: config.ServiceSpec{LayerSpec: config.LayerSpec{
			Values: map[string]any{"service-order": "second"},
		}},
	}
	appHost := bundle.Hosts["app"]
	appHost.Spec.Services = append(appHost.Spec.Services, "metrics")
	appHost.Spec.Values["precedence"] = "host"
	bundle.Hosts["app"] = appHost
	rendered, err := Resolve(bundle)
	if err != nil {
		t.Fatal(err)
	}
	var app config.RenderedHost
	for _, host := range rendered.Hosts {
		if host.Name == "app" {
			app = host
		}
	}
	if got := app.Config.Values["workers"]; got != 4 {
		t.Fatalf("host override lost: got %#v", got)
	}
	if got := app.Config.Values["precedence"]; got != "host" {
		t.Fatalf("full precedence failed: got %#v", got)
	}
	if got := app.Config.Values["service-order"]; got != "second" {
		t.Fatalf("service order was not respected: got %#v", got)
	}
	hardening := app.Config.Values["hardening"].(map[string]any)
	if hardening["enabled"] != true {
		t.Fatalf("inherited profile value lost: %#v", hardening)
	}
	http := app.Config.Values["http"].(map[string]any)
	if http["listen"] != "0.0.0.0" {
		t.Fatalf("service value lost: %#v", http)
	}
	if len(app.Ports) != 1 || !reflect.DeepEqual(app.Ports[0].Sources, []string{"192.0.2.10/32"}) {
		t.Fatalf("source_hosts not resolved: %#v", app.Ports)
	}
	ref := app.Config.Credentials["application-password"]
	if ref.File != "credentials/app-password" {
		t.Fatalf("credential reference lost: %#v", ref)
	}
}

func TestDeterministicYAMLAndValidJSONWithoutSecretExpansion(t *testing.T) {
	t.Setenv("PLATFORM_AUTOMATION_TOKEN", "must-not-be-printed")
	rendered, err := Resolve(exampleBundle(t))
	if err != nil {
		t.Fatal(err)
	}
	first, err := Encode(rendered, "yaml")
	if err != nil {
		t.Fatal(err)
	}
	second, err := Encode(rendered, "yaml")
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("YAML output is not deterministic")
	}
	jsonData, err := Encode(rendered, "json")
	if err != nil {
		t.Fatal(err)
	}
	var decoded any
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	for _, output := range [][]byte{first, jsonData} {
		if strings.Contains(string(output), "must-not-be-printed") {
			t.Fatal("render leaked an environment credential")
		}
		if !strings.Contains(string(output), "PLATFORM_AUTOMATION_TOKEN") {
			t.Fatal("render did not preserve the credential reference")
		}
	}
}

func TestRedactMasksInlineValuesOnly(t *testing.T) {
	env := config.RenderedEnvironment{
		Hosts: []config.RenderedHost{
			{
				Name: "app",
				Config: config.RenderedConfig{
					Credentials: map[string]config.SecretRef{
						"build-password": {Value: "s0meSecret"},
						"api-token":      {Env: "PLATFORM_API_TOKEN"},
						"ssh-key":        {File: "credentials/id_ed25519"},
					},
				},
			},
		},
	}
	redacted := Redact(env)
	got := redacted.Hosts[0].Config.Credentials
	if got["build-password"].Value != RedactedValue {
		t.Fatalf("expected inline value to be redacted, got %#v", got["build-password"])
	}
	if got["api-token"].Env != "PLATFORM_API_TOKEN" {
		t.Fatalf("env reference should be untouched, got %#v", got["api-token"])
	}
	if got["ssh-key"].File != "credentials/id_ed25519" {
		t.Fatalf("file reference should be untouched, got %#v", got["ssh-key"])
	}
	if env.Hosts[0].Config.Credentials["build-password"].Value != "s0meSecret" {
		t.Fatal("Redact must not mutate its input")
	}
}

func TestUnsupportedFormat(t *testing.T) {
	if _, err := Encode(config.RenderedEnvironment{}, "toml"); err == nil {
		t.Fatal("expected unsupported format error")
	}
}

func exampleBundle(t *testing.T) config.Bundle {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "examples", "platform-home"))
	if _, err := os.Stat(root); err != nil {
		t.Fatal(err)
	}
	bundle, err := config.Load(root, "lab")
	if err != nil {
		t.Fatal(err)
	}
	return bundle
}
