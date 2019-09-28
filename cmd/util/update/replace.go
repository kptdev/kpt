// Copyright 2019 Google LLC
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

package update

import "kpt.dev/util/get"

// Updater updates a package to a new upstream version.
//
// If the package at pkgPath differs from the upstream ref it was fetch from, then Update will
// delete the local package.  This will wipe all local changes.
type ReplaceUpdater struct{}

func (u ReplaceUpdater) Update(options UpdateOptions) error {
	options.KptFile.Upstream.Git.Ref = options.ToRef
	options.KptFile.Upstream.Git.Repo = options.ToRepo
	return get.Command{Destination: options.PackagePath, Clean: true, Git: options.KptFile.Upstream.Git}.Run()
}
