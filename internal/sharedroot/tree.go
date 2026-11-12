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

// EntryType is the kind of an entry of the tree.
type EntryType int

// Entry types; a symbolic link is never followed.
const (
	TypeOther EntryType = iota
	TypeDir
	TypeFile
	TypeSymlink
)

// EntryInfo describes one entry of the tree.
type EntryInfo struct {
	Name    string
	Type    EntryType
	Perm    os.FileMode // permission bits only
	Special bool        // setuid, setgid or sticky bit set
	UID     int
	Size    int64
}
