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

package location

// DirectoryNameDefaulter is present on Reference types that
// suggest a default local folder name
type DirectoryNameDefaulter interface {
	// GetDefaultDirectoryName implements the location.DefaultDirectoryName() method
	GetDefaultDirectoryName() (string, bool)
}

// DefaultDirectoryName returns the suggested local directory name to
// create when a package from a remove reference is cloned or pulled.
// Returns an empty string and false if the Reference type does not have
// anything path-like to suggest from.
func DefaultDirectoryName(ref Reference) (string, bool) {
	if ref, ok := ref.(DirectoryNameDefaulter); ok {
		return ref.GetDefaultDirectoryName()
	}
	return "", false
}
