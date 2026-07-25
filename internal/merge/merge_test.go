package merge

import (
	"reflect"
	"testing"
)

func TestMaps(t *testing.T) {
	base := map[string]any{
		"nested": map[string]any{"keep": "yes", "replace": "old"},
		"list":   []any{"old"},
	}
	overlay := map[string]any{
		"nested": map[string]any{"replace": "new"},
		"list":   []any{"new"},
		"null":   nil,
	}
	want := map[string]any{
		"nested": map[string]any{"keep": "yes", "replace": "new"},
		"list":   []any{"new"},
		"null":   nil,
	}
	if got := Maps(base, overlay); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}
