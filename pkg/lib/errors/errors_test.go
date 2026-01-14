// Copyright 2025-2026 The kpt and Nephio Authors
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

package errors

import (
	"errors"
	"strings"
	"testing"

	"github.com/kptdev/kpt/pkg/lib/types"
)

func TestErrorFormatting(t *testing.T) {
	baseErr := errors.New("base error")

	e := &Error{
		Op:    "pkg.get",
		Path:  types.UniquePath("/workspace/my-pkg"),
		Fn:    "my-fn",
		Repo:  "github.com/example/repo",
		Class: InvalidParam,
		Err:   baseErr,
	}

	got := e.Error()
	wantSubstrings := []string{
		"pkg.get",
		"pkg /workspace/my-pkg",
		"fn my-fn",
		"repo github.com/example/repo",
		"invalid parameter value",
		"base error",
	}

	for _, substr := range wantSubstrings {
		if !strings.Contains(got, substr) {
			t.Errorf("Expected error string to contain %q, got: %q", substr, got)
		}
	}
}
