// Package schema provides access to the canonical schemas embedded in the
// binary. Parsing and semantic validation live in config and validation.
package schema

import (
	"io/fs"

	"github.com/ednillibanio/platformctl/schemas"
)

const Version = "v1alpha1"

func Files() fs.FS {
	sub, err := fs.Sub(schemas.Files, Version)
	if err != nil {
		panic(err)
	}
	return sub
}
