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

package engine

import (
	"fmt"
	"testing"
)

// Relevant: https://github.com/golang/go/issues/20126

func TestSafeJoin(t *testing.T) {
	grid := []struct {
		base      string
		relative  string
		want      string
		wantError bool
	}{
		{
			base:     "/tmp",
			relative: "foo",
			want:     "/tmp/foo",
		},
		{
			base:     "/tmp/subdir/",
			relative: "foo",
			want:     "/tmp/subdir/foo",
		},
		{
			base:     "tmp",
			relative: "foo",
			want:     "tmp/foo",
		},
		{
			base:      "/tmp/subdir/",
			relative:  "/foo",
			wantError: true,
		},
		{
			base:      "/tmp/",
			relative:  "/tmp/foo",
			wantError: true,
		},
		{
			base:     "tmp/",
			relative: "tmp/foo",
			want:     "tmp/tmp/foo",
		},
		{
			base:      "tmp/",
			relative:  "../foo",
			wantError: true,
		},
		{
			base:      "tmp/",
			relative:  "a/../foo",
			wantError: true,
		},
		{
			base:      "tmp/",
			relative:  "a/../../foo",
			wantError: true,
		},
		{
			base:      "tmp/",
			relative:  "a/../../tmp/foo",
			wantError: true,
		},
	}

	for _, g := range grid {
		t.Run(fmt.Sprintf("%#v", g), func(t *testing.T) {
			got, err := filepathSafeJoin(g.base, g.relative)
			if g.wantError {
				if err == nil {
					t.Errorf("got %q and nil error, want error", got)
				}
			} else {
				if g.want != got {
					t.Errorf("unexpected value; got %q, want %q", got, g.want)
				}
			}
		})
	}
}
