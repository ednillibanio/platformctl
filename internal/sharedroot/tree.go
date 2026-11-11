package sharedroot

import "os"

// TempPrefix starts the name of every temporary file or directory a run
// creates; one left by an interrupted run is never read.
const TempPrefix = ".platformctl-tmp-"

// File is one file of a directory published by PublishDir.
type File struct {
	Name    string
	Content []byte
	Mode    os.FileMode
}
