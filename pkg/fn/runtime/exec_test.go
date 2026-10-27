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
	"context"
	"os/exec"
	"testing"
	"time"

	fnresultv1 "github.com/kptdev/kpt/api/fnresult/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecFn_Timeout(t *testing.T) {
	shBin, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("skipping exec timeout test: no sh binary found")
	}

	testCases := map[string]struct {
		script string
		// timeout is the time limit of the function
		timeout time.Duration
		// runTimeout is the time limit of the whole run
		runTimeout       time.Duration
		expectedTimeout  time.Duration
		expectRunTimeout bool
		expectedStderr   string
	}{
		"function that outlives the timeout fails": {
			script:          "echo stuck >&2; exec sleep 30",
			timeout:         100 * time.Millisecond,
			expectedTimeout: 100 * time.Millisecond,
			expectedStderr:  "stuck\n",
		},
		"child process of the function that keeps the output open is not waited for": {
			script:          "sleep 30; true",
			timeout:         100 * time.Millisecond,
			expectedTimeout: 100 * time.Millisecond,
		},
		"function that outlives the timeout of the whole run fails": {
			script:           "echo stuck >&2; exec sleep 30",
			runTimeout:       100 * time.Millisecond,
			expectedTimeout:  100 * time.Millisecond,
			expectRunTimeout: true,
			expectedStderr:   "stuck\n",
		},
		"shorter timeout of the function comes before the timeout of the whole run": {
			script:          "exec sleep 30",
			timeout:         100 * time.Millisecond,
			runTimeout:      30 * time.Second,
			expectedTimeout: 100 * time.Millisecond,
		},
		"shorter timeout of the whole run comes before the timeout of the function": {
			script:           "exec sleep 30",
			timeout:          30 * time.Second,
			runTimeout:       100 * time.Millisecond,
			expectedTimeout:  100 * time.Millisecond,
			expectRunTimeout: true,
		},
		"function that finishes within the timeout of the whole run succeeds": {
			script:     "true",
			runTimeout: 30 * time.Second,
		},
		"function that finishes within the timeout succeeds": {
			script:  "true",
			timeout: 30 * time.Second,
		},
		"zero timeout falls back to the default": {
			script:  "true",
			timeout: 0,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			ctx, cancel := WithRunTimeout(context.Background(), tc.runTimeout)
			defer cancel()
			fn := &ExecFn{
				Path:     shBin,
				Args:     []string{"-c", tc.script},
				Timeout:  tc.timeout,
				FnResult: &fnresultv1.Result{},
			}

			start := time.Now()
			err := fn.RunContext(ctx, &bytes.Buffer{}, &bytes.Buffer{})
			if tc.expectedTimeout == 0 {
				assert.NoError(t, err)
				return
			}

			var execErr *fnresultv1.ExecError
			require.ErrorAs(t, err, &execErr)
			assert.Equal(t, tc.expectedTimeout, execErr.Timeout)
			assert.Equal(t, tc.expectRunTimeout, execErr.RunTimeout)
			assert.Equal(t, timeoutExitCode, execErr.ExitCode)
			assert.Equal(t, tc.expectedStderr, execErr.Stderr)
			// kpt must not have waited for the function to finish
			assert.Less(t, time.Since(start), 10*time.Second)
		})
	}
}
