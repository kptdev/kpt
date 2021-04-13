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

// Package types defines the basic types used by the kpt codebase.
package types

// UniquePath represents absolute unique OS-defined path to the package directory on the filesystem.
type UniquePath string

// String returns the absolute path in string format.
func (u UniquePath) String() string {
	return string(u)
}

// Empty returns true if the UniquePath is empty
func (u UniquePath) Empty() bool {
	return len(u) == 0
}

// DisplayPath represents Slash-separated path to the package directory on the filesytem relative
// to current working directory.
// This is not guaranteed to be unique (e.g. in presence of symlinks) and should only
// be used for display purposes and is subject to change.
type DisplayPath string
