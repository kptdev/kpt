// Copyright 2019 The kpt Authors
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

package common

import (
	"os"
	"path/filepath"
)

const markdownExtension = ".md"

func ReadFiles(source string, recursive bool) ([]string, error) {
	filePaths := make([]string, 0)
	if recursive {
		err := filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if filepath.Ext(info.Name()) == markdownExtension {
				filePaths = append(filePaths, path)
			}
			return nil
		})
		if err != nil {
			return filePaths, err
		}
	} else {
		if filepath.Ext(source) == markdownExtension {
			filePaths = append(filePaths, source)
		} else {
			files, err := os.ReadDir(source)
			if err != nil {
				return filePaths, err
			}
			for _, info := range files {
				if filepath.Ext(info.Name()) == markdownExtension {
					path := filepath.Join(source, info.Name())
					filePaths = append(filePaths, path)
				}
			}
		}
	}
	return filePaths, nil
}
