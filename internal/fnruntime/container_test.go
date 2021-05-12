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

package fnruntime_test

import (
	"bytes"
	"context"
	"os/exec"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/fnruntime"
	"github.com/GoogleContainerTools/kpt/internal/printer"
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
		ctx := context.Background()
		t.Run(tt.name, func(t *testing.T) {
			outBuff, errBuff := &bytes.Buffer{}, &bytes.Buffer{}
			instance := fnruntime.ContainerFn{
				Ctx:   printer.WithContext(ctx, printer.New(outBuff, errBuff)),
				Image: tt.image,
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

func TestFnImagePull(t *testing.T) {
	image := "gcr.io/kpt-fn/search-replace:v0.1"

	// Initial test setup (our environment must not already include the image)
	cmdInspect := exec.Command("docker", "image", "inspect", image)
	errInspect := cmdInspect.Run()
	// docker image inspect exits successfully if an image is found
	// If image is already present remove the image before running test
	if errInspect == nil {
		cmdRm := exec.Command("docker", "image", "rm", image)
		errRm := cmdRm.Run()
		// Image should remove successfully
		assert.NoError(t, errRm)
	}
	// Test setup complete

	// Exec function (this should pull the missing function image)
	ctx := context.Background()
	outBuff, errBuff := &bytes.Buffer{}, &bytes.Buffer{}
	instance := fnruntime.ContainerFn{
		Ctx:   printer.WithContext(ctx, printer.New(outBuff, errBuff)),
		Image: image,
	}
	// Function execution should be a no-op
	input := bytes.NewBufferString("")
	output := &bytes.Buffer{}
	err := instance.Run(input, output)
	assert.NoError(t, err)

	// Inspect fails with an exit code of 1 if no image is present
	// We expect this command to succeed (the image exists)
	cmdImageInspect := exec.Command("docker", "image", "inspect", image)
	errImageInspect := cmdImageInspect.Run()
	assert.NoError(t, errImageInspect)
}
