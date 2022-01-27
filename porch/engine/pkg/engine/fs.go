// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package engine

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"time"

	"sigs.k8s.io/kustomize/kyaml/filesys"
)

type filedata struct {
	b []byte
}

// node represents file or a directory node in the fs
type node struct {
	mode fs.FileMode
	data *filedata // if file
}

type directory map[string]*node

type memfs struct {
	dirs map[string]directory
}

var _ filesys.FileSystem = &memfs{}

// Create a file.
func (m *memfs) Create(p string) (filesys.File, error) {
	dir, base := split(clean(p))
	if base == "" {
		return nil, &fs.PathError{Op: "Open", Path: p, Err: fs.ErrInvalid}
	}

	d, ok := m.dirs[dir]
	if !ok {
		return nil, &fs.PathError{Op: "Open", Path: p, Err: fs.ErrNotExist}
	}
	f, ok := d[base]
	if !ok {
		f = &node{
			mode: 0,
			data: &filedata{},
		}
		d[base] = f
	} else if f.mode.IsDir() {
		return nil, &fs.PathError{Op: "Open", Path: p, Err: fs.ErrNotExist}
	}

	return &file{
		name: base,
		data: f.data,
	}, nil
}

// MkDir makes a directory.
func (m *memfs) Mkdir(p string) error {
	return m.MkdirAll(p)
}

// MkDirAll makes a directory path, creating intervening directories.
func (m *memfs) MkdirAll(p string) error {
	p = clean(p)

	if _, ok := m.dirs[p]; ok {
		return nil // done
	}

	dir, base := split(p)
	if base != "" {
		if err := m.MkdirAll(dir); err != nil {
			return err
		}

		parent, ok := m.dirs[dir]
		if !ok {
			return fmt.Errorf("internal error: cannot find parent directory %q", dir)
		}

		if n, ok := parent[base]; ok {
			if !n.mode.IsDir() {
				return fmt.Errorf("%q already exists and is not a directory", p)
			}
		} else {
			parent[base] = &node{
				mode: fs.ModeDir,
			}
		}
	}

	if m.dirs == nil {
		m.dirs = map[string]directory{}
	}
	m.dirs[p] = directory{}
	return nil
}

// RemoveAll removes path and any children it contains.
func (m *memfs) RemoveAll(path string) error {
	return errors.New("RemoveAll is not implemented")
}

// Open opens the named file for reading.
func (m *memfs) Open(p string) (filesys.File, error) {
	dir, base := split(clean(p))
	if base == "" {
		return nil, &fs.PathError{Op: "Open", Path: p, Err: fs.ErrInvalid}
	}

	d, ok := m.dirs[dir]
	if !ok {
		return nil, &fs.PathError{Op: "Open", Path: p, Err: fs.ErrNotExist}
	}
	f, ok := d[base]
	if !ok {
		return nil, &fs.PathError{Op: "Open", Path: p, Err: fs.ErrNotExist}
	}
	if f.mode.IsDir() {
		return nil, &fs.PathError{Op: "Open", Path: p, Err: fs.ErrNotExist}
	}

	return &file{
		name: base,
		data: f.data,
	}, nil
}

// IsDir returns true if the path is a directory.
func (m *memfs) IsDir(p string) bool {
	p = path.Clean(p)
	_, ok := m.dirs[p]
	return ok
}

// ReadDir returns a list of files and directories within a directory.
func (m *memfs) ReadDir(path string) ([]string, error) {
	return nil, errors.New("ReadDir is not implemented")
}

// CleanedAbs converts the given path into a
// directory and a file name, where the directory
// is represented as a ConfirmedDir and all that implies.
// If the entire path is a directory, the file component
// is an empty string.
func (m *memfs) CleanedAbs(p string) (filesys.ConfirmedDir, string, error) {
	p = clean(p)
	if m.IsDir(p) {
		return filesys.ConfirmedDir(p), "", nil
	}

	dir, base := split(p)
	if base == "" {
		return "", "", &fs.PathError{
			Op:   "CleanedAbs",
			Path: p,
			Err:  fs.ErrNotExist,
		}
	}

	if m.IsDir(dir) {
		return filesys.ConfirmedDir(dir), base, nil
	}
	return "", "", &fs.PathError{
		Op:   "CleanedAbs",
		Path: p,
		Err:  fs.ErrNotExist,
	}
}

// Exists is true if the path exists in the file system.
func (m *memfs) Exists(p string) bool {
	p = clean(p)
	_, ok := m.dirs[p]
	if ok {
		return true
	}
	dir, base := split(p)
	d, ok := m.dirs[dir]
	if !ok {
		return false
	}
	_, ok = d[base]
	return ok
}

// Glob returns the list of matching files,
// emulating https://golang.org/pkg/path/filepath/#Glob
func (m *memfs) Glob(pattern string) ([]string, error) {
	return nil, errors.New("Glob is not implemented")
}

// ReadFile returns the contents of the file at the given path.
func (m *memfs) ReadFile(p string) ([]byte, error) {
	f, err := m.Open(p)
	if err != nil {
		return nil, err
	}
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if fi.IsDir() {
		return nil, &fs.PathError{Op: "ReadFile", Path: p, Err: fs.ErrPermission}
	}
	buf := make([]byte, fi.Size())
	_, err = f.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

// WriteFile writes the data to a file at the given path,
// overwriting anything that's already there.
func (m *memfs) WriteFile(p string, data []byte) error {
	dname, fname := split(p)
	if fname == "" {
		return &fs.PathError{Op: "WriteFile", Path: p, Err: fs.ErrNotExist}
	}
	dname = clean(dname)

	d, ok := m.dirs[dname]
	if !ok {
		return fmt.Errorf("directory doesn't exist: %q", dname)
	}

	f, ok := d[fname]
	if !ok {
		f = &node{
			mode: 0, // File
			data: &filedata{},
		}
		d[fname] = f
	}

	new := make([]byte, len(data))
	copy(new, data)
	f.data.b = new

	return nil
}

// Walk walks the file system with the given WalkFunc.
func (m *memfs) Walk(p string, walkFn filepath.WalkFunc) error {
	p = clean(p)

	dir, ok := m.dirs[p]
	if !ok {
		return &fs.PathError{
			Op:   "Walk",
			Path: p,
			Err:  fs.ErrNotExist,
		}
	}
	walkFn(p, &nodeinfo{
		name: path.Base(p),
		size: 0,
		mode: fs.ModeDir,
	}, nil)

	for k, v := range dir {
		fullpath := path.Join(p, k)
		if v.mode.IsDir() {
			return m.Walk(fullpath, walkFn)
		}

		var size int
		if v.data != nil {
			size = len(v.data.b)
		}

		if err := walkFn(path.Join(p, k), &nodeinfo{
			name: k,
			size: size,
			mode: v.mode,
		}, nil); err != nil {
			return err
		}
	}
	return nil
}

func split(p string) (string, string) {
	i := len(p) - 1
	for i >= 0 && p[i] != '/' {
		i--
	}
	if i <= 0 {
		return "/", p[i+1:]
	} else {
		return clean(p[:i]), p[i+1:]
	}
}

func clean(p string) string {
	if p == "" || p == "." {
		return "/"
	}
	p = path.Clean(p)
	if !path.IsAbs(p) {
		return "/" + p
	}
	return p
}

type nodeinfo struct {
	name string
	size int
	mode fs.FileMode
}

var _ fs.FileInfo = &nodeinfo{}

// base name of the file
func (n *nodeinfo) Name() string {
	return n.name
}

// length in bytes for regular files; system-dependent for others
func (n *nodeinfo) Size() int64 {
	return int64(n.size)
}

// file mode bits
func (n *nodeinfo) Mode() fs.FileMode {
	return n.mode
}

// modification time
func (n *nodeinfo) ModTime() time.Time {
	return time.Now()
}

// abbreviation for Mode().IsDir()
func (n *nodeinfo) IsDir() bool {
	return n.mode.IsDir()
}

// underlying data source (can return nil)
func (n *nodeinfo) Sys() interface{} {
	return nil
}

type file struct {
	name   string
	data   *filedata
	pos    int
	closed bool
}

var _ filesys.File = &file{}

func (f *file) Read(p []byte) (n int, err error) {
	if f.closed {
		return 0, fs.ErrClosed
	}
	src := f.data.b[f.pos:]
	if len(src) == 0 {
		return 0, io.EOF
	}

	n = copy(p, src)
	f.pos += n
	return n, nil
}

func (f *file) Write(p []byte) (n int, err error) {
	if f.closed {
		return 0, fs.ErrClosed
	}
	f.data.b = append(f.data.b, p...)
	return len(p), nil
}

func (f *file) Close() error {
	if f.closed {
		return fs.ErrClosed
	}
	f.closed = true
	return nil
}

func (f *file) Stat() (os.FileInfo, error) {
	if f.closed {
		return nil, fs.ErrClosed
	}
	return &nodeinfo{
		name: f.name,
		size: len(f.data.b),
	}, nil
}
