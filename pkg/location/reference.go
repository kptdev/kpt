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

import (
	"fmt"

	"github.com/GoogleContainerTools/kpt/pkg/location/extensions"
)

type Reference = extensions.Reference
type ReferenceLock = extensions.ReferenceLock

// GetRevision returns the string value used to identify
// which branch or tag is referenced.
// Typical revision values are often a semantic name like 'draft', 'main', 'prod', or a
// string representation of a version. The specifics of how the revision is
// mapped to storage depends on the type of reference.
func GetRevision(ref Reference) (string, bool) {
	if ref, ok := ref.(extensions.Revisable); ok {
		return ref.GetRevision()
	}
	return "", false
}

// SetRevision returns a new Reference where the property that
// identifies the branch, tag, or label has been replaced with value given.
// Typical revision values are often a semantic name like 'draft', 'main', 'prod', or a
// string representation of a version. The specifics of how the revision is
// mapped to storage depends on the type of reference.
func SetRevision(ref Reference, revision string) (Reference, error) {
	if ref, ok := ref.(extensions.Revisable); ok {
		return ref.SetRevision(revision)
	}
	return nil, fmt.Errorf("changing revision not supported for reference: %v", ref)
}

// DefaultRevision returns the suggested revision to use
// for a reference location when one is not provided. This may
// be a default tag name like "latest", or a default branch name
// like "main". Some locations may attempt to obtain the default
// revision by communicating with the remote provider.
func DefaultRevision(ref Reference, opts ...Option) (string, error) {
	if ref, ok := ref.(extensions.DefaultRevisionProvider); ok {
		opt := makeOptions(opts...)
		return ref.DefaultRevision(opt.ctx)
	}
	return "", fmt.Errorf("not supported")
}

func Lock(ref ReferenceLock) (string, bool) {
	if ref, ok := ref.(extensions.LockGetter); ok {
		return ref.GetLock()
	}
	return "", false
}

// Rel will return a relative path if one reference is a sub-package
// location in another. The usage is similar to filepath.Rel. The
// comparison is strict, meaning all criteria other than the directory
// component (like repo, ref, image, tag, etc.) must be equal.
func Rel(baseref Reference, targref Reference) (string, error) {
	if baseref, ok := baseref.(extensions.RelPather); ok {
		return baseref.Rel(targref)
	}
	return "", fmt.Errorf("not supported")
}

// DefaultDirectoryName returns the suggested local directory name to
// create when a package from a remove reference is cloned or pulled.
// Returns an empty string and false if the Reference type does not have
// anything path-like to suggest from.
func DefaultDirectoryName(ref Reference) (string, bool) {
	if ref, ok := ref.(extensions.DefaultDirectoryNameGetter); ok {
		return ref.GetDefaultDirectoryName()
	}
	return "", false
}
