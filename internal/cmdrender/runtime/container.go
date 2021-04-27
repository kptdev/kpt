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
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/GoogleContainerTools/kpt/internal/printer"
	"github.com/GoogleContainerTools/kpt/internal/types"
)

// containerNetworkName is a type for network name used in container
type containerNetworkName string

const (
	networkNameNone containerNetworkName = "none"
	networkNameHost containerNetworkName = "host"
	defaultTimeout  time.Duration        = 5 * time.Minute
)

// ContainerFnPermission contains the permission of container
// function such as network access.
type ContainerFnPermission struct {
	AllowNetwork bool
}

// ContainerFn implements a KRMFn which run a containerized
// KRM function
type ContainerFn struct {
	Ctx  context.Context
	Path types.UniquePath
	// Image is the container image to run
	Image string
	// Container function will be killed after this timeour.
	// The default value is 5 minutes.
	Timeout time.Duration
	Perm    ContainerFnPermission
}

// Run runs the container function using docker runtime.
// It reads the input from the given reader and writes the output
// to the provided writer.
func (f *ContainerFn) Run(reader io.Reader, writer io.Writer) error {
	// check and pull image before running to avoid polluting CLI
	// output
	err := f.prepareImage()
	if err != nil {
		return fmt.Errorf("failed to check function existence: %w", err)
	}
	pr := printer.FromContextOrDie(f.Ctx)
	errSink := bytes.Buffer{}
	cmd, cancel := f.getDockerCmd()
	defer cancel()
	cmd.Stdin = reader
	cmd.Stdout = writer
	cmd.Stderr = &errSink

	printOpt := printer.NewOpt().WithIndentation(printer.FnIndentation)
	pr.Printf(printOpt, "[RUNNING] %q\n", f.Image)
	if err := cmd.Run(); err != nil {
		pr.Printf(printOpt, "[FAIL] %q\n", f.Image)
		var exitCode int
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			return fmt.Errorf("cannot get function exit code: %w", err)
		}
		ff := &printer.FnFailure{
			ExitCode: exitCode,
			Stderr:   errSink.String(),
		}
		failureOpt := printer.NewOpt().
			WithIndentation(printer.FnFailureIndentation).
			WithStderr(true)
		pr.PrintPrintable(failureOpt, ff)
		// TODO: write complete details to a file
		return fmt.Errorf("function %q failed", f.Image)
	}
	pr.Printf(printOpt, "[PASS] %q\n", f.Image)
	return nil
}

func (f *ContainerFn) getDockerCmd() (*exec.Cmd, context.CancelFunc) {
	// directly use docker executable to run the container
	path := "docker"

	network := networkNameNone
	if f.Perm.AllowNetwork {
		network = networkNameHost
	}

	args := []string{
		"run", "--rm", "-i",
		"-a", "STDIN", "-a", "STDOUT", "-a", "STDERR",
		"--network", string(network),
		// TODO: this env is only used in TS SDK to print the errors
		// to stderr. We don't need this once we support structured
		// results.
		"-e", "LOG_TO_STDERR=true",
		"--security-opt=no-new-privileges",
	}
	args = append(args, f.Image)
	// setup container run timeout
	timeout := defaultTimeout
	if f.Timeout != 0 {
		timeout = f.Timeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	return exec.CommandContext(ctx, path, args...), cancel
}

// prepareImage will check local images and pull it if it doesn't
// exist.
func (f *ContainerFn) prepareImage() error {
	// check image existence
	path := "docker"
	args := []string{"image", "ls", f.Image}
	cmd := exec.Command(path, args...)
	var output []byte
	var err error
	if output, err = cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to check function %q: %w", f.Image, err)
	}
	if strings.Contains(string(output), strings.Split(f.Image, ":")[0]) {
		// image exists locally
		return nil
	}
	args = []string{"image", "pull", f.Image}
	// setup timeout
	timeout := defaultTimeout
	if f.Timeout != 0 {
		timeout = f.Timeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd = exec.CommandContext(ctx, path, args...)
	if _, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("function %q doesn't exist: %w", f.Image, err)
	}
	return nil
}
