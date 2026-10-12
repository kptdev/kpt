// Copyright 2021,2026 The kpt Authors
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
	goerrors "errors"
	"fmt"
	"io"
	"os/exec"
	"time"

	fnresultv1 "github.com/kptdev/kpt/api/fnresult/v1"
	"github.com/kptdev/kpt/pkg/printer"
)

// execWaitDelay is how long kpt waits for the output of an exec function
// to be closed after the function has exited or has been killed.
const execWaitDelay = time.Second

type ExecFn struct {
	// Ctx is the context of the run. If it has a time limit, the function is
	// killed when that limit has passed.
	Ctx context.Context

	// Path is the os specific path to the executable
	// file. It can be relative or absolute.
	Path string
	// Args are the arguments to the executable
	Args []string

	Env map[string]string
	// Exec function will be killed after this timeout.
	// The default value is 5 minutes, or the time limit of Ctx if it has one.
	Timeout time.Duration
	// FnResult is used to store the information about the result from
	// the function.
	FnResult *fnresultv1.Result
}

// Run runs the executable file which reads the input from r and
// writes the output to w.
func (f *ExecFn) Run(r io.Reader, w io.Writer) error {
	// setup exec run timeout
	ctx, cancel := fnContext(f.Ctx, f.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, f.Path, f.Args...)
	// do not wait past the timeout for a child process of the function that
	// still has the output open
	cmd.WaitDelay = execWaitDelay

	errSink := bytes.Buffer{}
	cmd.Stdin = r
	cmd.Stdout = w
	cmd.Stderr = &errSink

	for k, v := range f.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%v=%v", k, v))
	}

	if err := cmd.Run(); err != nil {
		if goerrors.Is(ctx.Err(), context.DeadlineExceeded) {
			return newTimeoutExecError(ctx, err, errSink.String())
		}
		if exitErr, ok := goerrors.AsType[*exec.ExitError](err); ok {
			return &ExecError{
				OriginalErr:    exitErr,
				ExitCode:       exitErr.ExitCode(),
				Stderr:         errSink.String(),
				TruncateOutput: printer.TruncateOutput,
			}
		}
		return fmt.Errorf("unexpected function error: %w", err)
	}

	if errSink.Len() > 0 {
		f.FnResult.Stderr = errSink.String()
	}

	return nil
}
