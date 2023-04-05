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

package git

import "strings"

// Determines whether a package specified by a path is in a directory given.
func packageInDirectory(pkg, dir string) bool {
	if dir == "" {
		return true
	}
	if strings.HasPrefix(pkg, dir) {
		if len(pkg) == len(dir) {
			return true
		}
		if pkg[len(dir)] == '/' {
			return true
		}
	}
	return false
}
