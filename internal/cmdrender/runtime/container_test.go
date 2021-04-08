// +build docker

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

package runtime_test

import (
	"bytes"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/cmdrender/runtime"
	"github.com/stretchr/testify/assert"
)

func TestContainerFn(t *testing.T) {
	var tests = []struct {
		input  string
		image  string
		output string
		name   string
		err    bool
	}{
		{
			name:  "simple busybox",
			image: "gcr.io/google-containers/busybox",
		},
		{
			name:  "non-existing image",
			image: "foobar",
			err:   true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			instance := runtime.ContainerFn{}
			instance.Image = tt.image
			input := bytes.NewBufferString(tt.input)
			output := &bytes.Buffer{}
			err := instance.Run(input, output)
			if tt.err && !assert.Error(t, err) {
				t.FailNow()
			}
			if !tt.err && !assert.NoError(t, err) {
				t.FailNow()
			}
			if !assert.Equal(t, tt.output, output.String()) {
				t.FailNow()
			}
		})
	}
}
