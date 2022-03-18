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

package porch

import (
	"fmt"
	"strings"
)

type PackageName struct {
	Original   string
	Repository string
	Package    string
	Revision   string
}

func ParsePackageName(name string) (PackageName, error) {
	parts := strings.Split(name, ":")
	if len(parts) != 3 {
		return PackageName{}, fmt.Errorf("invalid package name: %q", name)
	}
	return PackageName{
		Original:   name,
		Repository: parts[0],
		Package:    parts[1],
		Revision:   parts[2],
	}, nil
}

func ParsePartialPackageName(name string) (PackageName, int) {
	result := PackageName{
		Original: name,
	}

	parts := strings.Split(name, ":")
	count := len(parts)

	if count > 0 {
		result.Repository = parts[0]
	}
	if count > 1 {
		result.Package = parts[1]
	}
	if count > 2 {
		result.Revision = parts[2]
	}
	return result, count
}

func (name PackageName) Identifier() string {
	return name.String()
}

func (name PackageName) String() string {
	return fmt.Sprintf("%s:%s:%s", name.Repository, name.Package, name.Revision)
}

func LastSegment(path string) string {
	path = strings.TrimRight(path, "/")
	return path[strings.LastIndex(path, "/")+1:]
}
