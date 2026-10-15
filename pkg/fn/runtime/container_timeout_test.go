// Copyright 2026 The kpt Authors
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

package runtime

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	fnresultv1 "github.com/kptdev/kpt/api/fnresult/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainerFn_Timeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping container timeout test on Windows")
	}

	// A stand-in for the container CLI, so that the test does not need a
	// container runtime. It prints a CLI message and a function message, and
	// then does not finish.
	fakeCLI := filepath.Join(t.TempDir(), "fake-container-cli")
	script := `#!/bin/sh
echo "Unable to find image 'example.com/fn:v1' locally" >&2
echo "function is stuck" >&2
exec sleep 30
`
	require.NoError(t, os.WriteFile(fakeCLI, []byte(script), 0700))

	fn := &ContainerFn{
		Image:    "example.com/fn:v1",
		Timeout:  100 * time.Millisecond,
		FnResult: &fnresultv1.Result{},
	}

	start := time.Now()
	err := fn.runCLI(&bytes.Buffer{}, &bytes.Buffer{}, fakeCLI, filterDockerCLIOutput)

	var execErr *ExecError
	require.ErrorAs(t, err, &execErr)
	assert.Equal(t, 100*time.Millisecond, execErr.Timeout)
	assert.Equal(t, timeoutExitCode, execErr.ExitCode)
	// the CLI message is filtered out, the message of the function is kept
	assert.Equal(t, "function is stuck", execErr.Stderr)
	assert.Less(t, time.Since(start), 10*time.Second)
}
