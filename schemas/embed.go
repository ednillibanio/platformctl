// Package schemas exposes the versioned public schemas embedded in platformctl.
package schemas

import "embed"

// Files contains every published JSON Schema.
//
//go:embed v1alpha1/*.json
var Files embed.FS
