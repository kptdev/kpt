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

package open

import (
	"context"

	"github.com/GoogleContainerTools/kpt/pkg/content"
	"github.com/GoogleContainerTools/kpt/pkg/location"
)

// ContentProvider enables opening custom location.Reference types. They
// are given to the various open functions as an open.WithProviders(...) option.
//
// If the provider does not recognize the location.Reference type, then
// it MUST return open.ErrUnknownReference to allow the opening to continue
// normally.
type ContentProvider interface {
	Content(ctx context.Context, ref location.Reference) (content.Content, location.Reference, location.ReferenceLock, error)
}

// ContentProviderFunc enables a ContentProvider to be implemented as a func.
type ContentProviderFunc func(ctx context.Context, ref location.Reference) (content.Content, location.Reference, location.ReferenceLock, error)

func (f ContentProviderFunc) Content(ctx context.Context, ref location.Reference) (content.Content, location.Reference, location.ReferenceLock, error) {
	return f(ctx, ref)
}
