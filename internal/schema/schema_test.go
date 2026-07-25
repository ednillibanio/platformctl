package schema

import (
	"encoding/json"
	"io/fs"
	"testing"
)

func TestEmbeddedSchemasAreValidJSON(t *testing.T) {
	entries, err := fs.ReadDir(Files(), ".")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) < 2 {
		t.Fatalf("expected at least two schemas, got %d", len(entries))
	}
	for _, entry := range entries {
		data, err := fs.ReadFile(Files(), entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		var document map[string]any
		if err := json.Unmarshal(data, &document); err != nil {
			t.Fatalf("%s is invalid JSON: %v", entry.Name(), err)
		}
		if document["$schema"] != "https://json-schema.org/draft/2020-12/schema" {
			t.Fatalf("%s does not declare JSON Schema 2020-12", entry.Name())
		}
	}
}
