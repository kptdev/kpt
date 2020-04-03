// Copyright 2019 Google LLC
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

package e2e

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/kpt/run"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

func TestKptGetSet(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !assert.True(t, ok) {
		t.FailNow()
	}
	root := filepath.Dir(filepath.Dir(filename))

	d, err := ioutil.TempDir("", "")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	defer os.RemoveAll(d)
	if !assert.NoError(t, os.Chdir(d)) {
		t.FailNow()
	}

	cmd := run.GetMain()
	cmd.SetArgs([]string{"pkg", "get",
		filepath.Join(root, ".git", "package-examples", "helloworld-set"),
		"helloworld"})
	err = cmd.Execute()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	expected := &bytes.Buffer{}
	err = kio.Pipeline{
		Inputs: []kio.Reader{kio.LocalPackageReader{
			PackagePath: filepath.Join(
				root, "package-examples", "helloworld-set")}},
		Outputs: []kio.Writer{kio.ByteWriter{Writer: expected}},
	}.Execute()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	actual := &bytes.Buffer{}
	err = kio.Pipeline{
		Inputs: []kio.Reader{kio.LocalPackageReader{
			PackagePath: "helloworld"}},
		Outputs: []kio.Writer{kio.ByteWriter{Writer: actual}},
	}.Execute()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	if !assert.Equal(t, expected.String(), actual.String()) {
		t.FailNow()
	}

	cmd = run.GetMain()
	cmd.SetArgs([]string{"cfg", "set", "helloworld", "replicas", "7"})
	err = cmd.Execute()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	actual.Reset()
	err = kio.Pipeline{
		Inputs: []kio.Reader{kio.LocalPackageReader{
			PackagePath: "helloworld"}},
		Outputs: []kio.Writer{kio.ByteWriter{Writer: actual}},
	}.Execute()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	expectedString := strings.Replace(expected.String(),
		"replicas: 5", "replicas: 7", -1)

	if !assert.Equal(t, expectedString, actual.String()) {
		t.FailNow()
	}
}
