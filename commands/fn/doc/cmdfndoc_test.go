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
	"testing"

	"github.com/GoogleContainerTools/kpt/commands/fn/doc"
	"github.com/GoogleContainerTools/kpt/pkg/printer/fake"
	"sigs.k8s.io/kustomize/kyaml/testutil"
)

// TestDesc_Execute tests happy path for Describe command.
func TestFnDoc(t *testing.T) {
	type testcase struct {
		image     string
		expectErr string
	}
	testcases := []testcase{
		{
			image: "gcr.io/kpt-fn/set-namespace:v0.1.1",
		},
		{
			image:     "gcr.io/kpt-fn/set-namespace:v0.1.0",
			expectErr: "please ensure the container has an entrypoint and it supports --help flag",
		},
		{
			image:     "",
			expectErr: "image must be specified",
		},
	}

	for _, tc := range testcases {
		b := &bytes.Buffer{}
		runner := doc.NewRunner(fake.CtxWithPrinter(b, b), "kpt")
		runner.Image = tc.image
		err := runner.Command.Execute()
		if tc.expectErr == "" {
			testutil.AssertNoError(t, err)
		} else {
			testutil.AssertErrorContains(t, err, tc.expectErr)
		}
	}
}
