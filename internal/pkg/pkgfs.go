package pkg

import (
	"io/fs"
	"path"
	"path/filepath"
)

type FSer interface {
	FS() fs.FS
}
type PkgFS struct {
	fs fs.FS

	pkgPath string
}

func NewPkgFS(path string, fs fs.FS) fs.FS {
	return &PkgFS{
		pkgPath: path,
		fs:      fs,
	}
}

func (pkg PkgFS) rel(name string) string {
	rel, err := filepath.Rel(pkg.pkgPath, name)
	if err != nil {
		panic(err)
	}
	return rel
}

func (pkg PkgFS) join(path string) string {
	return filepath.Join(pkg.pkgPath, path)
}

func (pkg PkgFS) Open(name string) (fs.File, error) {
	if path.IsAbs(name) {
		name = pkg.rel(name)
	}
	return pkg.fs.Open(name)
}

func (pkg PkgFS) Stat(name string) (fs.FileInfo, error) {
	if path.IsAbs(name) {
		name = pkg.rel(name)
	}
	return fs.Stat(pkg.fs, name)
}

func (pkg PkgFS) Sub(sub string) *PkgFS {
	return &PkgFS{
		pkgPath: path.Join(pkg.pkgPath, sub),
		fs:      pkg.fs,
	}
}

func (pkg PkgFS) FS() fs.FS {
	return pkg.fs
}

// interface guards

var (
	_ fs.FS = &PkgFS{}
)
