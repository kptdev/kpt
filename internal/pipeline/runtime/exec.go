// Copyright 2021 Google LLC
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
	"context"
	"io"
	"os"
	"os/exec"
	"time"
)

const (
	// default timeout for ExecFn is 5 minutes
	defaultTimeout = 300 * time.Second
)

// ExecFn runs a exec command
type ExecFn struct {
	// Path is the path to the executable to run
	Path string `yaml:"path,omitempty"`

	// Args are the arguments to the executable
	Args []string `yaml:"args,omitempty"`

	Timeout time.Duration
}

// Run runs the command
func (c *ExecFn) Run(reader io.Reader, writer io.Writer) error {
	timeout := defaultTimeout
	if c.Timeout != 0 {
		timeout = c.Timeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, c.Path, c.Args...)
	cmd.Stdin = reader
	cmd.Stdout = writer
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
