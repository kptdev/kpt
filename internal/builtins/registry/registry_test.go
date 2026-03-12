// Copyright 2026 The kpt Authors
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

package registry

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

type fakeBuiltin struct {
	name string
}

func (f *fakeBuiltin) ImageName() string                  { return f.name }
func (f *fakeBuiltin) Run(_ io.Reader, _ io.Writer) error { return nil }

func TestNormalizeImage(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ghcr.io/kptdev/starlark:v0.5.0", "ghcr.io/kptdev/starlark"},
		{"ghcr.io/kptdev/starlark@sha256:abc123", "ghcr.io/kptdev/starlark"},
		{"ghcr.io/kptdev/starlark:v0.5.0@sha256:abc123", "ghcr.io/kptdev/starlark"},
		{"ghcr.io/kptdev/starlark", "ghcr.io/kptdev/starlark"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, normalizeImage(tc.input))
		})
	}
}

func TestRegisterAndLookup(t *testing.T) {
	registry = map[string]BuiltinFunction{}

	fn := &fakeBuiltin{name: "ghcr.io/kptdev/test:v1.0"}
	Register(fn)

	result := Lookup("ghcr.io/kptdev/test:v2.0")
	assert.NotNil(t, result)
	assert.Equal(t, fn, result)

	result = Lookup("ghcr.io/kptdev/unknown")
	assert.Nil(t, result)
}

func TestList(t *testing.T) {
	registry = map[string]BuiltinFunction{}

	Register(&fakeBuiltin{name: "ghcr.io/kptdev/fn1:v1"})
	Register(&fakeBuiltin{name: "ghcr.io/kptdev/fn2:v1"})

	list := List()
	assert.Len(t, list, 2)
}
