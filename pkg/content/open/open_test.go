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
	"testing"

	"github.com/GoogleContainerTools/kpt/pkg/content"
	"github.com/GoogleContainerTools/kpt/pkg/location"
	"github.com/stretchr/testify/assert"
)

func TestWithDifferentContentProvider(t *testing.T) {
	// arrange
	ref := location.Dir{Directory: "/x"}

	// act
	res, err := Content(
		ref,
		WithProviders(ContentProviderFunc(func(ctx context.Context, ref location.Reference) (content.Content, location.Reference, location.ReferenceLock, error) {
			if ref, ok := ref.(location.Dir); ok {
				return &customDirContent{}, ref, nil, nil
			}
			return nil, nil, nil, ErrUnknownReference
		})))

	// assert
	assert.Nil(t, err)
	assert.IsType(t, &customDirContent{}, res.Content)
}

type customDirContent struct {
}

func (customDirContent) Close() error {
	return nil
}
