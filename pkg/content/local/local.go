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

package local

import (
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/pkg/content"
	"github.com/GoogleContainerTools/kpt/pkg/content/extensions"
	"github.com/GoogleContainerTools/kpt/pkg/content/paths"
	"github.com/GoogleContainerTools/kpt/pkg/content/provider/dir"
)

type LocalFileSystemResult struct {
	content.Content
	paths.FileSystemPath
	AbsolutePath string
}

type noopCloser struct{}

func (noopCloser) Close() error {
	return nil
}

// LocalFileSystem returns the absolute path of the content if it is
// already a local file. If it is not, then dir.MkdirTemp is used to
// create a temporary content location and the information is copied.
//
// The AbsolutePath is always usable for OS disk operations.
//
// The return content.Content.Close() must always be called which will
// delete the temp folder if needed, but perform no action otherwise.
//
// The returned file system should only be read, not modified, because
// it is ambiguous whether or not the src will be changed as a result.
func LocalFileSystem(src content.Content) (LocalFileSystemResult, error) {
	srcFSP, err := content.FileSystem(src)
	if err != nil {
		return LocalFileSystemResult{}, err
	}

	if path, err := realPath(src); err == nil {
		return LocalFileSystemResult{
			Content:        noopCloser{},
			FileSystemPath: srcFSP,
			AbsolutePath:   path,
		}, nil
	}

	return CopyFileSystem(srcFSP, "kpt-")
}

// CopyFileSystem always uses dir.MkdirTemp to create a temporary content
// location and copies the information from src.
//
// The AbsolutePath is always usable for OS disk operations.
//
// The return content.Content.Close() must always be called which will
// delete the temp folder if needed, but perform no action otherwise. Changing
// the content of the returned FileSystem will never have an effect on src.
func CopyFileSystem(src paths.FileSystemPath, pattern string) (LocalFileSystemResult, error) {
	temp, err := dir.MkdirTemp(pattern)
	if err != nil {
		return LocalFileSystemResult{}, err
	}

	path, err := realPath(temp)
	if err != nil {
		temp.Close()
		return LocalFileSystemResult{}, err
	}

	dst, err := content.FileSystem(temp)
	if err != nil {
		temp.Close()
		return LocalFileSystemResult{}, err
	}

	if err := copyFileSystemPath(src, dst); err != nil {
		temp.Close()
		return LocalFileSystemResult{}, err
	}

	return LocalFileSystemResult{
		Content:        temp,
		FileSystemPath: dst,
		AbsolutePath:   path,
	}, nil
}

func realPath(src content.Content) (string, error) {
	if src, ok := src.(extensions.RealPathProvider); ok {
		return src.ProvideRealPath()
	}
	return "", fmt.Errorf("real path not supported")
}

func copyFileSystemPath(src paths.FileSystemPath, dst paths.FileSystemPath) error {
	return src.FileSystem.Walk(src.Path, func(srcPath string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src.Path, srcPath)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst.Path, relPath)

		if info.IsDir() {
			if err := dst.FileSystem.MkdirAll(dstPath); err != nil {
				return err
			}
			return nil
		}

		b, err := src.FileSystem.ReadFile(srcPath)
		if err != nil {
			return err
		}

		if err := dst.FileSystem.WriteFile(dstPath, b); err != nil {
			return err
		}

		return nil
	})
}
