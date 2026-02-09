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

package doc_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/kptdev/kpt/commands/fn/doc"
	"github.com/kptdev/kpt/internal/fnruntime"
	"github.com/kptdev/kpt/pkg/printer/fake"
	"sigs.k8s.io/kustomize/kyaml/testutil"
)

// TestDesc_Execute tests happy path for Describe command.
func TestFnDoc(t *testing.T) {
	// Skip test if Docker is not available
	runtime, err := fnruntime.StringToContainerRuntime(os.Getenv(fnruntime.ContainerRuntimeEnv))
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	if err := fnruntime.ContainerRuntimeAvailable(runtime); err != nil {
		t.Skipf("Skipping test: container runtime not available: %v", err)
	}
	type testcase struct {
		image     string
		expectErr string
	}
	testcases := []testcase{
		{
			image: "ghcr.io/kptdev/krm-functions-catalog/upsert-resource:latest",
		},
		{
			image:     "ghcr.io/kptdev/krm-functions-catalog/upsert-resource:v0.0.1",
			expectErr: "please ensure the container has an entrypoint and it supports --help flag",
		},
		{
			image:     "",
			expectErr: "image must be specified",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.image, func(t *testing.T) {
			b := &bytes.Buffer{}
			runner := doc.NewRunner(fake.CtxWithPrinter(b, b), "kpt")
			runner.Image = tc.image
			err := runner.Command.Execute()
			if tc.expectErr == "" {
				// Skip if container runtime fails to pull/run the image
				// This can happen in CI due to rate limits or network issues
				if err != nil && strings.Contains(err.Error(), "exit status 125") {
					t.Skipf("Skipping test: container runtime failed to run image: %v", err)
				}
				testutil.AssertNoError(t, err)
			} else {
				testutil.AssertErrorContains(t, err, tc.expectErr)
			}
		})
	}
}
