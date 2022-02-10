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

package content

import (
	"errors"
	"fmt"

	"github.com/GoogleContainerTools/kpt/pkg/content/extensions"
	"github.com/GoogleContainerTools/kpt/pkg/location"
)

var (
	ErrNotCommittable = errors.New("not committable")
)

func Commit(content Content) (location.Reference, location.ReferenceLock, error) {
	switch content := content.(type) {
	case extensions.ChangeCommitter:
		return content.CommitChanges()
	default:
		return nil, nil, fmt.Errorf("invalid %T operation: %w", content, ErrNotCommittable)
	}
}
