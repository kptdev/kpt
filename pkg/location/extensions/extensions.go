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

package extensions

import (
	"context"
	"fmt"
)

type Reference interface {
	fmt.Stringer
	Type() string
	Validate() error
}

type ReferenceLock interface {
	Reference
}

// Revisable is present on Reference types that
// support location.GetRevision and location.WithRevision.
type Revisable interface {
	Reference
	GetRevision() (string, bool)
	WithRevision(revision string) (Reference, error)
}

// DefaultRevisionProvider is present on Reference types that
// support location.DefaultRevision.
type DefaultRevisionProvider interface {
	DefaultRevision(ctx context.Context) (string, error)
}

type LockGetter interface {
	GetLock() (string, bool)
}

// DefaultDirectoryNameGetter is present on Reference types that
// suggest a default local folder name
type DefaultDirectoryNameGetter interface {
	// GetDefaultDirectoryName implements the location.DefaultDirectoryName() method
	GetDefaultDirectoryName() (string, bool)
}

// RelPather is present on Reference types that
// will return a relative path if one reference is a sub-package
// location in another. The comparison is strict, meaning all criteria
// other than the directory component (like repo, ref, image, tag, etc.) must be equal.
type RelPather interface {
	Rel(targref Reference) (string, error)
}
