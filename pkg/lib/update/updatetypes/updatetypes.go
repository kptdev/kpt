// Copyright 2026 The kpt Authors
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

// Package updatetypes holds the exposed types for updates in kpt
package updatetypes

// Updater updates a local package
type Updater interface {
	Update(options Options) error
}

type Options struct {
	// RelPackagePath is the relative path of a subpackage to the root. If the
	// package is root, the value here will be ".".
	RelPackagePath string

	// LocalPath is the absolute path to the package on the local fork.
	LocalPath string

	// OriginPath is the absolute path to the package in the on-disk clone
	// of the origin ref of the repo.
	OriginPath string

	// UpdatedPath is the absolute path to the package in the on-disk clone
	// of the updated ref of the repo.
	UpdatedPath string

	// IsRoot is true if the package is the root, i.e. the clones of
	// updated and origin were fetched based on the information in the
	// Kptfile from this package.
	IsRoot bool
}
