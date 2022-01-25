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

type Name struct {
	RepositoryName string
	ImageName      string
	Version        string
}

func ParseName(name string) (Name, error) {
	firstIndex := strings.Index(name, ":")
	lastIndex := strings.LastIndex(name, ":")
	if firstIndex == lastIndex {
		return Name{}, fmt.Errorf("invalid name - insufficient colons")
	}
	return Name{
		RepositoryName: name[:firstIndex],
		ImageName:      name[firstIndex+1 : lastIndex],
		Version:        name[lastIndex+1:],
	}, nil
}
