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

package fn

import (
	"errors"
	"testing"

	v1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/google/go-cmp/cmp"
)

func TestNotFound(t *testing.T) {
	var err error

	fn := &v1.Function{Image: "foo"}
	err = &NotFoundError{Function: *fn}

	var notFoundErr *NotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Fatalf("expected NotFoundError to satisfy errors.As")
	}

	if diff := cmp.Diff(notFoundErr.Function, *fn); diff != "" {
		t.Fatalf("Unexpected result (-want, +got): %s", diff)
	}

	if diff := cmp.Diff(notFoundErr.Error(), "function \"foo\" not found"); diff != "" {
		t.Fatalf("Unexpected result (-want, +got): %s", diff)
	}
}
