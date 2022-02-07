package ioutil

import (
	"io/fs"

	"github.com/GoogleContainerTools/kpt/internal/types"
)

// ReadFile reads the file named by filename and returns the contents.
// A successful call returns err == nil, not err == EOF. Because ReadFile
// reads the whole file, it does not treat an EOF from Read as an error
// to be reported.
//
// As of Go 1.16, this function simply calls os.ReadFile.
func ReadFile(filename types.FileSystemPath) ([]byte, error) {
	return filename.FileSystem.ReadFile(filename.Path)
}

// WriteFile writes data to a file named by filename.
// If the file does not exist, WriteFile creates it with permissions perm
// (before umask); otherwise WriteFile truncates it before writing, without changing permissions.
//
// As of Go 1.16, this function simply calls os.WriteFile.
func WriteFile(filename types.FileSystemPath, data []byte, perm fs.FileMode) error {
	return filename.FileSystem.WriteFile(filename.Path, data)
}
