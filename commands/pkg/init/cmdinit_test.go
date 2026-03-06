// Copyright 2019 The kpt Authors
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

package init_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	initialization "github.com/kptdev/kpt/commands/pkg/init"
	"github.com/kptdev/kpt/internal/builtins"
	"github.com/kptdev/kpt/internal/testutil"
	"github.com/kptdev/kpt/internal/util/man"
	builtintypes "github.com/kptdev/kpt/pkg/lib/builtins/builtintypes"
	"github.com/kptdev/kpt/pkg/printer/fake"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.ConfigureTestKptCache(m))
}

// TestCmd verifies the directory is initialized
func TestCmd(t *testing.T) {
	d := t.TempDir()
	assert.NoError(t, os.Mkdir(filepath.Join(d, "my-pkg"), 0700))

	r := initialization.NewRunner(fake.CtxWithDefaultPrinter(), "kpt")
	r.Command.SetArgs([]string{filepath.Join(d, "my-pkg"), "--description", "my description"})
	err := r.Command.Execute()
	assert.NoError(t, err)

	// verify the contents
	b, err := os.ReadFile(filepath.Join(d, "my-pkg", "Kptfile"))
	assert.NoError(t, err)
	assert.Equal(t, `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: my-pkg
  annotations:
    config.kubernetes.io/local-config: "true"
info:
  description: my description
`, string(b))

	b, err = os.ReadFile(filepath.Join(d, "my-pkg", man.ManFilename))
	assert.NoError(t, err)
	assert.Equal(t, strings.ReplaceAll(`# my-pkg

## Description
my description

## Usage

### Fetch the package
'kpt pkg get REPO_URI[.git]/PKG_PATH[@VERSION] my-pkg'
Details: https://kpt.dev/reference/cli/pkg/get/

### View package content
'kpt pkg tree my-pkg'
Details: https://kpt.dev/reference/cli/pkg/tree/

### Apply the package
'''
kpt live init my-pkg
kpt live apply my-pkg --reconcile-timeout=2m --output=table
'''
Details: https://kpt.dev/reference/cli/live/
`, "'", "`"), string(b))

	b, err = os.ReadFile(filepath.Join(d, "my-pkg", builtintypes.PkgContextFile))
	assert.NoError(t, err)
	assert.Equal(t, b, []byte(builtins.AbstractPkgContext()))
}

func TestCmd_currentDir(t *testing.T) {
	d := t.TempDir()
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

		r := initialization.NewRunner(fake.CtxWithDefaultPrinter(), "kpt")
		r.Command.SetArgs([]string{".", "--description", "my description"})
		return r.Command.Execute()
	}()
	assert.NoError(t, err)

	// verify the contents
	b, err := os.ReadFile(filepath.Join(packageDir, "Kptfile"))
	assert.NoError(t, err)
	assert.Equal(t, `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: my-pkg
  annotations:
    config.kubernetes.io/local-config: "true"
info:
  description: my description
`, string(b))
}

func TestCmd_DefaultToCurrentDir(t *testing.T) {
	d := t.TempDir()
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

		r := initialization.NewRunner(fake.CtxWithDefaultPrinter(), "kpt")
		r.Command.SetArgs([]string{"--description", "my description"})
		return r.Command.Execute()
	}()
	assert.NoError(t, err)

	// verify the contents
	b, err := os.ReadFile(filepath.Join(packageDir, "Kptfile"))
	assert.NoError(t, err)
	assert.Equal(t, `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: my-pkg
  annotations:
    config.kubernetes.io/local-config: "true"
info:
  description: my description
`, string(b))
}

// TestCmd_AutoCreateDir verifies the command creates the directory if it doesn't exist
func TestCmd_AutoCreateDir(t *testing.T) {
	d := t.TempDir()
	r := initialization.NewRunner(fake.CtxWithDefaultPrinter(), "kpt")
	r.Command.SetArgs([]string{filepath.Join(d, "my-pkg"), "--description", "my description"})
	err := r.Command.Execute()
	assert.NoError(t, err)

	// verify the directory was created
	_, err = os.Stat(filepath.Join(d, "my-pkg"))
	assert.NoError(t, err)

	// verify the contents
	b, err := os.ReadFile(filepath.Join(d, "my-pkg", "Kptfile"))
	assert.NoError(t, err)
	assert.Equal(t, `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: my-pkg
  annotations:
    config.kubernetes.io/local-config: "true"
info:
  description: my description
`, string(b))
}
