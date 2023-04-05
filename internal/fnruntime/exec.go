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

package fnruntime

import (
	"bytes"
	"context"
	goerrors "errors"
	"fmt"
	"io"
	"os/exec"
	"time"

	fnresult "github.com/GoogleContainerTools/kpt/pkg/api/fnresult/v1"
	"github.com/GoogleContainerTools/kpt/pkg/printer"
)

type ExecFn struct {
	// Path is the os specific path to the executable
	// file. It can be relative or absolute.
	Path string
	// Args are the arguments to the executable
	Args []string

	Env map[string]string
	// Container function will be killed after this timeour.
	// The default value is 5 minutes.
	Timeout time.Duration
	// FnResult is used to store the information about the result from
	// the function.
	FnResult *fnresult.Result
}

// Run runs the executable file which reads the input from r and
// writes the output to w.
func (f *ExecFn) Run(r io.Reader, w io.Writer) error {
	// setup exec run timeout
	timeout := defaultLongTimeout
	if f.Timeout != 0 {
		timeout = f.Timeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, f.Path, f.Args...)

	errSink := bytes.Buffer{}
	cmd.Stdin = r
	cmd.Stdout = w
	cmd.Stderr = &errSink

	for k, v := range f.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%v=%v", k, v))
	}

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if goerrors.As(err, &exitErr) {
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
