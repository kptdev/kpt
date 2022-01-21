package pkg

import (
	"io/fs"
	"path"
	"path/filepath"
)

// Thanks to Louis for the initial implementation.

// Sometimes deep down in the call stack, we need
// underlying FS but we may only have access to the
// wrappedFS, so this interface defines capability
// to provide the underlying FS.
type UnwrapFS interface {
	FS() fs.FS
}

// fs.FS implementations (io/DirFS etc) don't work
// with rooted (absolute paths).
// PrefixFS wraps a given fs.FS type to make it work
// with rooted (absolute paths).
// kpt codebase uses absolute paths for pkg, so PrefixFS
// avoids converting absolute paths to relative paths everywhere.
type PrefixFS struct {
	fs fs.FS

	prefix string
}

func NewPrefixFS(path string, fs fs.FS) fs.FS {
	return &PrefixFS{
		prefix: path,
		fs:     fs,
	}
}

func (pkg PrefixFS) rel(name string) string {
	rel, err := filepath.Rel(pkg.prefix, name)
	if err != nil {
		panic(err)
	}
	return rel
}

func (pkg PrefixFS) join(path string) string {
	return filepath.Join(pkg.prefix, path)
}

func (pkg PrefixFS) Open(name string) (fs.File, error) {
	if path.IsAbs(name) {
		name = pkg.rel(name)
	}
	return pkg.fs.Open(name)
}

func (pkg PrefixFS) Stat(name string) (fs.FileInfo, error) {
	if path.IsAbs(name) {
		name = pkg.rel(name)
	}
	return fs.Stat(pkg.fs, name)
}

func (pkg PrefixFS) Sub(sub string) *PrefixFS {
	return &PrefixFS{
		prefix: path.Join(pkg.prefix, sub),
		fs:     pkg.fs,
	}
}

func (pkg PrefixFS) FS() fs.FS {
	return pkg.fs
}

// interface guards

var (
	_ fs.FS = &PrefixFS{}
)
