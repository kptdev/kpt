// Copyright 2021 Google LLC
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

package os

import (
	"os"
	"path/filepath"
	"time"

	"github.com/GoogleContainerTools/kpt/internal/types"
)

// A FileInfo describes a file and is returned by Stat and Lstat.
type FileInfo = os.FileInfo

// A FileMode represents a file's mode and permission bits.
// The bits have the same definition on all systems, so that
// information about files can be moved from one system
// to another portably. Not all bits apply to all systems.
// The only required bit is ModeDir for directories.
type FileMode = os.FileMode

var ModePerm = os.ModePerm // Unix permission bits, 0o777

// IsNotExist returns a boolean indicating whether the error is known to
// report that a file or directory does not exist. It is satisfied by
// ErrNotExist as well as some syscall errors.
//
// This function predates errors.Is. It only supports errors returned by
// the os package. New code should use errors.Is(err, fs.ErrNotExist).
var IsNotExist = os.IsNotExist

var ErrNotExist = os.ErrNotExist // "file does not exist"

func Stdout() *os.File {
	return os.Stdout
}

// Stat returns a FileInfo describing the named file.
// If there is an error, it will be of type *PathError.
func Stat(name types.FileSystemPath) (FileInfo, error) {
	// NOTE: filesys.FileSystem does not support Stat()
	if !name.FileSystem.Exists(name.Path) {
		return nil, os.ErrNotExist
	}
	if name.FileSystem.IsDir(name.Path) {
		return &info{
			name: filepath.Base(name.Path),
			size: 0,
			mode: os.ModeDir | os.ModePerm,
		}, nil
	}
	return &info{
		name: filepath.Base(name.Path),
		size: 0,
		mode: os.ModePerm,
	}, nil
}

// RemoveAll removes path and any children it contains.
// It removes everything it can but returns the first error
// it encounters. If the path does not exist, RemoveAll
// returns nil (no error).
// If there is an error, it will be of type *PathError.
func RemoveAll(path types.FileSystemPath) error {
	return path.FileSystem.RemoveAll(path.Path)
}

// Remove removes the named file or (empty) directory.
// If there is an error, it will be of type *PathError.
func Remove(path types.FileSystemPath) error {
	// NOTE: filesys.FileSystem does not support Remove()
	return path.FileSystem.RemoveAll(path.Path)
}

// MkdirAll creates a directory named path,
// along with any necessary parents, and returns nil,
// or else returns an error.
// The permission bits perm (before umask) are used for all
// directories that MkdirAll creates.
// If path is already a directory, MkdirAll does nothing
// and returns nil.
func MkdirAll(path types.FileSystemPath, perm FileMode) error {
	return path.FileSystem.MkdirAll(path.Path)
}

// Mkdir creates a new directory with the specified name and permission
// bits (before umask).
// If there is an error, it will be of type *PathError.
func Mkdir(path types.FileSystemPath, perm FileMode) error {
	return path.FileSystem.Mkdir(path.Path)
}

type info struct {
	name string
	size int
	mode os.FileMode
}

var _ os.FileInfo = (*info)(nil)

func (fi *info) Name() string {
	return fi.name
}

func (fi *info) Size() int64 {
	return int64(fi.size)
}

func (fi *info) Mode() os.FileMode {
	return fi.mode
}

func (*info) ModTime() time.Time {
	return time.Now()
}

func (fi *info) IsDir() bool {
	return fi.mode.IsDir()
}

func (*info) Sys() interface{} {
	return nil
}
