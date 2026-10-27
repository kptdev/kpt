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

type ExecFn struct {
	// Path is the os specific path to the executable
	// file. It can be relative or absolute.
	Path string
	// Args are the arguments to the executable
	Args []string

	Env map[string]string
	// Exec function will be killed after this timeout.
	// The default value is 5 minutes, or the time limit of the context passed
	// to RunContext if it has one.
	Timeout time.Duration
	// FnResult is used to store the information about the result from
	// the function.
	FnResult *fnresultv1.Result
}

// Run runs the executable file which reads the input from r and
// writes the output to w.
func (f *ExecFn) Run(r io.Reader, w io.Writer) error {
	return f.RunContext(context.Background(), r, w)
}

// RunContext is like Run, but the function is also killed when the time
// limit of ctx has passed.
func (f *ExecFn) RunContext(ctx context.Context, r io.Reader, w io.Writer) error {
	// setup exec run timeout
	ctx, cancel := fnContext(ctx, f.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, f.Path, f.Args...)

	errSink := bytes.Buffer{}
	cmd.Stdin = r
	cmd.Stdout = w
	cmd.Stderr = &errSink

	for k, v := range f.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%v=%v", k, v))
	}

	// The function can exit successfully and leave a child process with its
	// output open. Then the end of ctx kills only the child, and err is nil.
	err := runCmd(ctx, cmd)
	if ctxErr := contextError(ctx, err, errSink.String); ctxErr != nil {
		return ctxErr
	}
	if err != nil {
		if exitErr, ok := goerrors.AsType[*exec.ExitError](err); ok {
			return &fnresultv1.ExecError{
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
