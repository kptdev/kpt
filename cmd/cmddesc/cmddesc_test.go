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

package cmddesc_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/assert"
	"kpt.dev/cmddesc"
	"lib.kpt.dev/kptfile"
	"lib.kpt.dev/testutil"
)

// TestDesc_Execute tests happy path for Describe command.
func TestDesc_Execute(t *testing.T) {
	d, err := ioutil.TempDir("", "kptdesc")
	testutil.AssertNoError(t, err)

	defer func() {
		_ = os.RemoveAll(d)
	}()

	// write the KptFile
	err = ioutil.WriteFile(filepath.Join(d, kptfile.KptFileName), []byte(`
apiVersion: kpt.dev/v1alpha1
kind: Kptfile
metadata:
  name: cockroachdb_perf
packageMetadata: {}
upstream:
  git:
    commit: 9b6aeba0f9c2f8c44c712848b6f147f15ca3344f
    directory: cloud/kubernetes/performance
    ref: master
    repo: https://github.com/cockroachdb/cockroach
  type: git
`), 0600)
	testutil.AssertNoError(t, err)

	b := &bytes.Buffer{}
	cmd := cmddesc.Cmd()
	cmd.C.SetArgs([]string{d})
	cmd.C.SetOut(b)
	err = cmd.C.Execute()
	testutil.AssertNoError(t, err)

	exp := fmt.Sprintf(`+------------------+------------------+------------------------------------------+------------------------------+---------+---------+
| LOCAL DIRECTORY  |       NAME       |            SOURCE REPOSITORY             |           SUBPATH            | VERSION | COMMIT  |
+------------------+------------------+------------------------------------------+------------------------------+---------+---------+
| %s | cockroachdb_perf | https://github.com/cockroachdb/cockroach | cloud/kubernetes/performance | master  | 9b6aeba |
+------------------+------------------+------------------------------------------+------------------------------+---------+---------+
`, filepath.Base(d))
	assert.Equal(t, exp, b.String())

}

// TestCmd_defaultPkg tests describe command execution with no directory
// specified.
func TestCmd_defaultPkg(t *testing.T) {
	b := &bytes.Buffer{}
	cmd := cmddesc.Cmd()
	cmd.C.SetOut(b)
	err := cmd.C.Execute()
	testutil.AssertNoError(t, err)

	exp := `+-----------------+------+-------------------+---------+---------+--------+
| LOCAL DIRECTORY | NAME | SOURCE REPOSITORY | SUBPATH | VERSION | COMMIT |
+-----------------+------+-------------------+---------+---------+--------+
+-----------------+------+-------------------+---------+---------+--------+
`
	assert.Equal(t, exp, b.String())
}
