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

package render

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/testutil"
	"github.com/GoogleContainerTools/kpt/pkg/printer/fake"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestCmd_flagAndArgParsing_Symlink(t *testing.T) {
	dir := t.TempDir()
	defer testutil.Chdir(t, dir)()

	err := os.MkdirAll(filepath.Join(dir, "path", "to", "pkg", "dir"), 0700)
	assert.NoError(t, err)
	err = os.Symlink(filepath.Join("path", "to", "pkg", "dir"), "foo")
	assert.NoError(t, err)

	// verify the branch ref is set to the correct value
	r := NewRunner(fake.CtxWithDefaultPrinter(), "kpt")
	r.Command.RunE = NoOpRunE
	r.Command.SetArgs([]string{"foo"})
	err = r.Command.Execute()
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join("path", "to", "pkg", "dir"), r.pkgPath)
}

// NoOpRunE is a noop function to replace the run function of a command.  Useful for testing argument parsing.
var NoOpRunE = func(cmd *cobra.Command, args []string) error { return nil }
