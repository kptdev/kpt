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

// Git Repository Adapter
//
// This component implements integration of git into package orchestration.
// A local clone of a registered git repository is created in a cache, and
// periodically refreshed.
// All package operations happen on the local copy; on completion of an
// operation (create, update, delete package revision or its resources)
// any new changes (in one or more commits) are pushed to the remote
// repository.
//
// # Branching Strategy
//
// Porch doesn't create tracking branches for remotes. This indirection
// would add a layer of complexity where branches can become out of sync
// and in need of reconciliation and conflict resolution. Instead, Porch
// analyzes the remote references (refs/remotes/origin/branch...) to
// discover packges. These refs are never directly updated by Porch other
// than by push or fetch to/from remote.
// Any intermediate commits Porch makes are either in 'detached HEAD'
// mode, or using temporary branches (these will become relevant if/when
// Porch implements repository garbage collection).
//
// Porch uses the default convention for naming remote branches
// (refs/remotes/origin/branch...) in order to make direct introspection
// of the repositories aligned with traditional git repositories.
package git
