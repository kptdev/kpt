package cmdrender

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/printer/fake"
	"github.com/GoogleContainerTools/kpt/internal/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestCmd_flagAndArgParsing_Symlink(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	defer os.RemoveAll(dir)
	defer testutil.Chdir(t, dir)()

	err = os.MkdirAll(filepath.Join(dir, "path", "to", "pkg", "dir"), 0700)
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
