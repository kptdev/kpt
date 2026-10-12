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

//go:build unix

package runtime

import (
	"bytes"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	fnresultv1 "github.com/kptdev/kpt/api/fnresult/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// heartbeatLoop adds a line to the file given as the first argument of the
// script every 50ms. It ends when the directory of the file is removed, which
// the test does when it finishes, so that no process is left behind.
const heartbeatLoop = `while echo alive >> "$1" 2>/dev/null; do sleep 0.05; done`

const (
	// functionThatWaitsForChild starts a child process, which ignores SIGTERM
	// and does not use the output of the function, and waits for it.
	functionThatWaitsForChild = `(trap '' TERM; ` + heartbeatLoop + `) >/dev/null 2>&1 &
wait`
	// functionThatLeavesChild starts a child process, which ignores SIGTERM and
	// has the output of the function open, and exits.
	functionThatLeavesChild = `(trap '' TERM; ` + heartbeatLoop + `) &`
	// functionThatRuns does not start a child process.
	functionThatRuns = heartbeatLoop
)

func newHeartbeatFn(t *testing.T, script string, timeout time.Duration) (fn *ExecFn, heartbeat string) {
	t.Helper()
	shBin, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("skipping exec function test: no sh binary found")
	}
	heartbeat = filepath.Join(t.TempDir(), "heartbeat")
	return &ExecFn{
		Path:     shBin,
		Args:     []string{"-c", script, "sh", heartbeat},
		Timeout:  timeout,
		FnResult: &fnresultv1.Result{},
	}, heartbeat
}

func fileSize(t *testing.T, path string) int64 {
	t.Helper()
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return 0
	}
	require.NoError(t, err)
	return info.Size()
}

// assertHeartbeatStopped checks that the process with the heartbeatLoop has
// run and is not running any more.
func assertHeartbeatStopped(t *testing.T, heartbeat string) {
	t.Helper()
	time.Sleep(100 * time.Millisecond)
	size := fileSize(t, heartbeat)
	assert.NotZero(t, size, "the process did not run")
	time.Sleep(300 * time.Millisecond)
	assert.Equal(t, size, fileSize(t, heartbeat), "the process is still running")
}

func TestExecFn_TimeoutStopsChildProcesses(t *testing.T) {
	testCases := map[string]struct {
		script      string
		expectError bool
	}{
		"function that waits for its child": {
			script:      functionThatWaitsForChild,
			expectError: true,
		},
		// the function is not running any more when the timeout passes
		"function that has exited and left a child with its output open": {
			script: functionThatLeavesChild,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			fn, heartbeat := newHeartbeatFn(t, tc.script, time.Second)

			start := time.Now()
			err := fn.Run(&bytes.Buffer{}, &bytes.Buffer{})

			if tc.expectError {
				var execErr *ExecError
				require.ErrorAs(t, err, &execErr)
				assert.Equal(t, -1, execErr.ExitCode)
			}
			// kpt must not have waited for the child to finish
			assert.Less(t, time.Since(start), 10*time.Second)
			assertHeartbeatStopped(t, heartbeat)
		})
	}
}

func TestExecFn_ForwardsSignal(t *testing.T) {
	if signal.Ignored(syscall.SIGINT) {
		t.Skip("skipping signal test: SIGINT is ignored")
	}
	// the signal must not end the test
	raised := make(chan syscall.Signal, 1)
	origRaiseSignal := raiseSignal
	raiseSignal = func(sig syscall.Signal) { raised <- sig }
	t.Cleanup(func() { raiseSignal = origRaiseSignal })

	fn, heartbeat := newHeartbeatFn(t, functionThatRuns, time.Minute)
	done := make(chan error, 1)
	go func() {
		done <- fn.Run(&bytes.Buffer{}, &bytes.Buffer{})
	}()
	require.Eventually(t, func() bool { return fileSize(t, heartbeat) > 0 },
		10*time.Second, 10*time.Millisecond, "the function did not start")

	require.NoError(t, syscall.Kill(syscall.Getpid(), syscall.SIGINT))

	// the function has received the signal, which ends it
	select {
	case err := <-done:
		assert.Error(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("the function did not receive the signal")
	}
	assertHeartbeatStopped(t, heartbeat)
	// kpt has received the signal again, now without a function to forward it to
	select {
	case sig := <-raised:
		assert.Equal(t, syscall.SIGINT, sig)
	case <-time.After(10 * time.Second):
		t.Fatal("the signal was not sent to kpt itself")
	}
}
