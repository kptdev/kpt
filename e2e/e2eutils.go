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
	"io/ioutil"
	"os"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func SetupGitRepo(t *testing.T) (*testutil.TestGitRepo, string, func()) {
	local, err := ioutil.TempDir("", "")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	if !assert.NoError(t, os.Chdir(local)) {
		defer os.RemoveAll(local)
		t.FailNow()
	}

	g, _, c := testutil.SetupDefaultRepoAndWorkspace(t, testutil.Dataset1)
	upstream := g.RepoDirectory

	clean := func() {
		_ = os.RemoveAll(local)
		c()
	}

	// remove default data
	testutil.RemoveData(t, g)

	return g, upstream, clean
}

// Exec runs a cobra command and fails if the command fails
func Exec(t *testing.T, cmd *cobra.Command) {
	if !assert.NoError(t, cmd.Execute()) {
		t.FailNow()
	}
}
