// Copyright 2022 The kpt Authors
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
	"fmt"
	"path/filepath"
	"strings"
)

// Relevant: https://github.com/golang/go/issues/20126

func filepathSafeJoin(dir string, relative string) (string, error) {
	p := filepath.Join(dir, relative)
	p = filepath.Clean(p)

	rel, err := filepath.Rel(dir, p)
	if err != nil {
		return "", fmt.Errorf("invalid relative path %q", relative)
	}
	if rel != relative || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || strings.HasPrefix(rel, "."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid relative path %q", relative)
	}
	return p, nil
}
