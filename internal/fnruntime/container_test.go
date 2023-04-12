//go:build docker

// Copyright 2021 The kpt Authors
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

package fnruntime

import (
	"bytes"
	"context"
	"testing"

	fnresult "github.com/GoogleContainerTools/kpt/pkg/api/fnresult/v1"
	"github.com/GoogleContainerTools/kpt/pkg/printer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		ctx := context.Background()
		t.Run(tt.name, func(t *testing.T) {
			errBuff := &bytes.Buffer{}
			instance := ContainerFn{
				Ctx:   printer.WithContext(ctx, printer.New(nil, errBuff)),
				Image: tt.image,
				FnResult: &fnresult.Result{
					Image: tt.image,
				},
			}
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

func TestIsSupportedDockerVersion(t *testing.T) {
	tests := []struct {
		name   string
		inputV string
		errMsg string
	}{
		{
			name:   "greater than min version",
			inputV: "20.10.1",
		},
		{
			name:   "equal to min version",
			inputV: "20.10.0",
		},
		{
			name:   "less than min version",
			inputV: "20.9.1",
			errMsg: "docker client version must be v20.10.0 or greater: found v20.9.1",
		},
		{
			name:   "invalid semver",
			inputV: "20..12.1",
			errMsg: "docker client version must be v20.10.0 or greater: found invalid version v20..12.1",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			err := isSupportedDockerVersion(tt.inputV)
			if tt.errMsg != "" {
				require.NotNil(err)
				require.Contains(err.Error(), tt.errMsg)
			} else {
				require.NoError(err)
			}
		})
	}
}
