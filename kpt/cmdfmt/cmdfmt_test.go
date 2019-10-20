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

package cmdfmt_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"kpt.dev/kpt/cmdfmt"
	"lib.kpt.dev/kio/filters/testyaml"
)

// TestCmd_files verifies the fmt command formats the files
func TestCmd_files(t *testing.T) {
	f1, err := ioutil.TempFile("", "cmdfmt*.yaml")
	if !assert.NoError(t, err) {
		return
	}
	defer os.RemoveAll(f1.Name())
	err = ioutil.WriteFile(f1.Name(), testyaml.UnformattedYaml1, 0600)
	if !assert.NoError(t, err) {
		return
	}

	f2, err := ioutil.TempFile("", "cmdfmt*.yaml")
	if !assert.NoError(t, err) {
		return
	}
	defer os.RemoveAll(f2.Name())
	err = ioutil.WriteFile(f2.Name(), testyaml.UnformattedYaml2, 0600)
	if !assert.NoError(t, err) {
		return
	}

	// fmt the files
	r := cmdfmt.Cmd()
	r.C.SetArgs([]string{f1.Name(), f2.Name()})
	err = r.C.Execute()
	if !assert.NoError(t, err) {
		return
	}

	// verify the contents
	b, err := ioutil.ReadFile(f1.Name())
	if !assert.NoError(t, err) {
		return
	}
	if !assert.Equal(t, string(testyaml.FormattedYaml1), string(b)) {
		return
	}

	b, err = ioutil.ReadFile(f2.Name())
	if !assert.NoError(t, err) {
		return
	}
	if !assert.Equal(t, string(testyaml.FormattedYaml2), string(b)) {
		return
	}
}

func TestCmd_stdin(t *testing.T) {
	out := &bytes.Buffer{}
	r := cmdfmt.Cmd()
	r.C.SetOut(out)
	r.C.SetIn(bytes.NewReader(testyaml.UnformattedYaml1))

	// fmt the input
	err := r.C.Execute()
	assert.NoError(t, err)

	// verify the output
	assert.Equal(t, string(testyaml.FormattedYaml1), out.String())
}

// TestCmd_filesAndstdin verifies that if both files and stdin input are provided, only
// the files are formatted and the input is ignored
func TestCmd_filesAndstdin(t *testing.T) {
	f1, err := ioutil.TempFile("", "cmdfmt*.yaml")
	if !assert.NoError(t, err) {
		return
	}
	err = ioutil.WriteFile(f1.Name(), testyaml.UnformattedYaml1, 0600)
	if !assert.NoError(t, err) {
		return
	}

	f2, err := ioutil.TempFile("", "cmdfmt*.yaml")
	if !assert.NoError(t, err) {
		return
	}
	err = ioutil.WriteFile(f2.Name(), testyaml.UnformattedYaml2, 0600)
	if !assert.NoError(t, err) {
		return
	}

	out := &bytes.Buffer{}
	in := &bytes.Buffer{}
	r := cmdfmt.Cmd()
	r.C.SetOut(out)
	r.C.SetIn(in)

	// fmt the files
	r = cmdfmt.Cmd()
	r.C.SetArgs([]string{f1.Name(), f2.Name()})
	err = r.C.Execute()
	if !assert.NoError(t, err) {
		return
	}

	// verify the output
	b, err := ioutil.ReadFile(f1.Name())
	if !assert.NoError(t, err) {
		return
	}
	if !assert.Equal(t, string(testyaml.FormattedYaml1), string(b)) {
		return
	}

	b, err = ioutil.ReadFile(f2.Name())
	if !assert.NoError(t, err) {
		return
	}
	if !assert.Equal(t, string(testyaml.FormattedYaml2), string(b)) {
		return
	}
	err = r.C.Execute()
	if !assert.NoError(t, err) {
		return
	}

	if !assert.Equal(t, "", out.String()) {
		return
	}
}

// TestCmd_files verifies the fmt command formats the files
func TestCmd_failFiles(t *testing.T) {
	// fmt the files
	r := cmdfmt.Cmd()
	r.C.SetArgs([]string{"notrealfile"})
	err := r.C.Execute()
	assert.EqualError(t, err, "lstat notrealfile: no such file or directory")
}

// TestCmd_files verifies the fmt command formats the files
func TestCmd_failFileContents(t *testing.T) {
	out := &bytes.Buffer{}
	r := cmdfmt.Cmd()
	r.C.SetOut(out)
	r.C.SetIn(strings.NewReader(`{`))

	// fmt the input
	err := r.C.Execute()

	// expect an error
	assert.EqualError(t, err, "yaml: line 1: did not find expected node content")
}
