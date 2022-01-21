package pkg

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"sigs.k8s.io/kustomize/kyaml/filesys"
)

// Thanks for Louis for this adapter.

// kyamlFileSystem adapter accepts an abstract fs.FS and
// implements the kyaml/filesys.FileSystem interface.
type kyamlFileSystem struct {
	// underlying file system
	fsys fs.FS

	// the base path where the abstract FS appears.
	// A default values "/" is recommended
	// to make the io/fs.FS relative path "Kptfile"
	// appear as kyaml/filesys.FileSystem absolute path "/Kptfile"
	// and to make the io/fs.FS relative path "hello/world.yaml"
	// appear as kyaml/filesys.FileSystem absolute path "/hello/world.yaml"
	prefix string
}

var _ filesys.FileSystem = &kyamlFileSystem{}

func (a *kyamlFileSystem) rel(path string) string {
	abs, err := filepath.Rel(a.prefix, path)
	if err != nil {
		panic(err)
	}
	return abs
}

func (a *kyamlFileSystem) join(path string) string {
	return filepath.Join(a.prefix, path)
}

// Create a file.
func (a *kyamlFileSystem) Create(path string) (filesys.File, error) {
	return nil, errors.New("not implemented")
}

// MkDir makes a directory.
func (a *kyamlFileSystem) Mkdir(path string) error {
	return errors.New("not implemented")
}

// MkDirAll makes a directory path, creating intervening directories.
func (a *kyamlFileSystem) MkdirAll(path string) error {
	return errors.New("not implemented")
}

// RemoveAll removes path and any children it contains.
func (a *kyamlFileSystem) RemoveAll(path string) error {
	return errors.New("not implemented")
}

// Open opens the named file for reading.
func (a *kyamlFileSystem) Open(path string) (filesys.File, error) {
	f, err := a.fsys.Open(a.rel(path))
	if err != nil {
		return nil, err
	}
	return &file{File: f}, nil
}

// IsDir returns true if the path is a directory.
func (a *kyamlFileSystem) IsDir(path string) bool {
	info, err := fs.Stat(a.fsys, a.rel(path))
	return err == nil && info.IsDir()
}

// ReadDir returns a list of files and directories within a directory.
func (a *kyamlFileSystem) ReadDir(path string) ([]string, error) {
	dirs, err := fs.ReadDir(a.fsys, a.rel(path))
	if err != nil {
		return nil, err
	}
	paths := make([]string, len(dirs))
	for index, dir := range dirs {
		paths[index] = dir.Name()
	}
	return paths, nil
}

// CleanedAbs converts the given path into a
// directory and a file name, where the directory
// is represented as a ConfirmedDir and all that implies.
// If the entire path is a directory, the file component
// is an empty string.
func (a *kyamlFileSystem) CleanedAbs(path string) (filesys.ConfirmedDir, string, error) {
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		// it could be better to return an error when path is relative, rather than
		// assume the base of the abstract FS should be treated as the working dir
		path = a.join(path)
	}
	if a.IsDir(path) {
		return filesys.ConfirmedDir(path), "", nil
	}

	dir, file := filepath.Split(path)
	if a.IsDir(dir) {
		return filesys.ConfirmedDir(dir), file, nil
	}

	return "", "", fmt.Errorf("first part of '%s' not a directory", path)
}

// Exists is true if the path exists in the file system.
func (a *kyamlFileSystem) Exists(path string) bool {
	_, err := fs.Stat(a.fsys, a.rel(path))
	return err == nil
}

// Glob returns the list of matching files,
// emulating https://golang.org/pkg/path/filepath/#Glob
func (a *kyamlFileSystem) Glob(pattern string) ([]string, error) {
	matches, err := fs.Glob(a.fsys, pattern)
	if err != nil {
		return nil, err
	}
	for index, match := range matches {
		matches[index] = a.join(match)
	}
	return matches, nil
}

// ReadFile returns the contents of the file at the given path.
func (a *kyamlFileSystem) ReadFile(path string) ([]byte, error) {
	return fs.ReadFile(a.fsys, a.rel(path))
}

// WriteFile writes the data to a file at the given path,
// overwriting anything that's already there.
func (a *kyamlFileSystem) WriteFile(path string, data []byte) error {
	return errors.New("not implemented")
}

// Walk walks the file system with the given WalkFunc.
func (a *kyamlFileSystem) Walk(root string, walkFn filepath.WalkFunc) error {
	return fs.WalkDir(a.fsys, a.rel(root), func(path string, de fs.DirEntry, err error) error {
		info, infoErr := de.Info()
		if infoErr != nil {
			return infoErr
		}
		return walkFn(a.join(path), info, err)
	})
}

type file struct {
	fs.File
}

var _ filesys.File = &file{}

func (a *file) Write(p []byte) (n int, err error) {
	return 0, errors.New("not implemented")
}
