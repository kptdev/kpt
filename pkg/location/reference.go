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

type Reference interface {
	fmt.Stringer
	Type() string
	Validate() error
}

type ReferenceLock interface {
	Reference
}

func Identifier(ref Reference) (string, bool) {
	if ref, ok := ref.(extensions.IdentifierGetter); ok {
		return ref.GetIdentifier()
	}
	return "", false
}

func Lock(ref ReferenceLock) (string, bool) {
	if ref, ok := ref.(extensions.LockGetter); ok {
		return ref.GetLock()
	}
	return "", false
}

// DefaultIdentifier returns the suggested identifier to use
// for a reference location when one is not provided. This may
// be a default tag name like "latest", or a default branch name
// like "main". Some locations may attempt to obtain the default
// identifier by communicating with the remote provider.
func DefaultIdentifier(ref Reference, opts ...Option) (string, error) {
	if ref, ok := ref.(extensions.DefaultIdentifierGetter); ok {
		opt := makeOptions(opts...)
		return ref.GetDefaultIdentifier(opt.ctx)
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
