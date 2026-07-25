package config

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestExamplesLoad(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "examples", "platform-home"))
	bundle, err := Load(root, "lab")
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Environment.Metadata.Name != "lab" || len(bundle.Hosts) != 2 {
		t.Fatalf("unexpected example bundle: %#v", bundle)
	}
}
