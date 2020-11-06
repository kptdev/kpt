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
	"strings"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/cmdinit"
	"github.com/GoogleContainerTools/kpt/internal/gitutil"
	"github.com/GoogleContainerTools/kpt/internal/testutil"
	"github.com/GoogleContainerTools/kpt/internal/util/man"
	"github.com/stretchr/testify/assert"
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

	b, err = ioutil.ReadFile(filepath.Join(d, "my-pkg", man.ManFilename))
	assert.NoError(t, err)
	assert.Equal(t, strings.ReplaceAll(`# my-pkg

## Description
my description

## Usage

### Fetch the package
'kpt pkg get REPO_URI[.git]/PKG_PATH[@VERSION] my-pkg'
Details: https://googlecontainertools.github.io/kpt/reference/pkg/get/

### View package content
'kpt cfg tree my-pkg'
Details: https://googlecontainertools.github.io/kpt/reference/cfg/tree/

### List setters
'kpt cfg list-setters my-pkg'
Details: https://googlecontainertools.github.io/kpt/reference/cfg/list-setters/

### Set a value
'kpt cfg set my-pkg NAME VALUE'
Details: https://googlecontainertools.github.io/kpt/reference/cfg/set/

### Apply the package
'''
kpt live init my-pkg
kpt live apply my-pkg --reconcile-timeout=2m --output=table
'''
Details: https://googlecontainertools.github.io/kpt/reference/live/
`, "'", "`"), string(b))
}

func TestCmd_currentDir(t *testing.T) {
	d, err := ioutil.TempDir("", "kpt")
	assert.NoError(t, err)
	assert.NoError(t, os.Mkdir(filepath.Join(d, "my-pkg"), 0700))

	packageDir := filepath.Join(d, "my-pkg")
	currentDir, err := os.Getwd()
	assert.NoError(t, err)
	err = func() error {
		nestedErr := os.Chdir(packageDir)
		if nestedErr != nil {
			return nestedErr
		}
		defer func() {
			deferErr := os.Chdir(currentDir)
			if deferErr != nil {
				panic(deferErr)
			}
		}()

		r := cmdinit.NewRunner("kpt")
		r.Command.SetArgs([]string{".", "--description", "my description", "--tag", "app.kpt.dev/cockroachdb"})
		return r.Command.Execute()
	}()
	assert.NoError(t, err)

	// verify the contents
	b, err := ioutil.ReadFile(filepath.Join(packageDir, "Kptfile"))
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

func TestGitUtil_DefaultRef(t *testing.T) {
	// set up git repo with both main and master branches
	g, _, clean := testutil.SetupDefaultRepoAndWorkspace(t, testutil.Dataset1)
	defer clean()

	// check if master is picked as default if both main and master branches exist
	defaultRef, err := gitutil.DefaultRef("file://" + g.RepoDirectory)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	if !assert.Equal(t, "master", defaultRef) {
		t.FailNow()
	}
	if !assert.Equal(t, "master", defaultRef) {
		t.FailNow()
	}

	err = g.CheckoutBranch("main", false)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	// delete master branch and check if main is selected as default
	err = g.DeleteBranch("master")
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	defaultRef, err = gitutil.DefaultRef("file://" + g.RepoDirectory)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	if !assert.Equal(t, "main", defaultRef) {
		t.FailNow()
	}

	err = g.CheckoutBranch("master", true)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	// delete main branch and check if master is selected as default
	err = g.DeleteBranch("main")
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	defaultRef, err = gitutil.DefaultRef("file://" + g.RepoDirectory)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	if !assert.Equal(t, "master", defaultRef) {
		t.FailNow()
	}
}
