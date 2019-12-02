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

package cmdinit_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"kpt.dev/kpt/cmdinit"
)

// TestCmd verifies the directory is initialized
func TestCmd(t *testing.T) {
	d, err := ioutil.TempDir("", "kpt")
	assert.NoError(t, err)
	assert.NoError(t, os.Mkdir(filepath.Join(d, "my-pkg"), 0700))

	r := cmdinit.NewRunner("kpt")
	r.Command.SetArgs([]string{filepath.Join(d, "my-pkg"), "--description", "my description", "--tag", "app.kpt.dev/cockroachdb"})
	err = r.Command.Execute()
	assert.NoError(t, err)

	// verify the contents
	b, err := ioutil.ReadFile(filepath.Join(d, "my-pkg", "Kptfile"))
	assert.NoError(t, err)
	assert.Equal(t, `apiVersion: kpt.dev/v1alpha1
kind: Kptfile
metadata:
  name: my-pkg
packageMetadata:
  tags:
  - app.kpt.dev/cockroachdb
  shortDescription: my description
`, string(b))

	b, err = ioutil.ReadFile(filepath.Join(d, "my-pkg", "MAN.md"))
	assert.NoError(t, err)
	assert.Equal(t, `my-pkg
==================================================

# NAME

  my-pkg

# SYNOPSIS

  kubectl apply --recursive -f my-pkg

# Description

my description

# SEE ALSO

`, string(b))
}

// TestCmd_failExists verifies the command throws and error if the directory exists
func TestCmd_failNotExists(t *testing.T) {
	d, err := ioutil.TempDir("", "kpt")
	assert.NoError(t, err)

	r := cmdinit.NewRunner("kpt")
	r.Command.SetArgs([]string{filepath.Join(d, "my-pkg"), "--description", "my description", "--tag", "app.kpt.dev/cockroachdb"})
	err = r.Command.Execute()
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "does not exist")
	}
}
