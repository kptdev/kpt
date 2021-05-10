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

package fnruntime

import (
	"bytes"
	"context"
	goerrors "errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/printer"
	"github.com/GoogleContainerTools/kpt/internal/types"
	"sigs.k8s.io/kustomize/kyaml/fn/runtime/runtimeutil"
)

// containerNetworkName is a type for network name used in container
type containerNetworkName string

const (
	networkNameNone containerNetworkName = "none"
	networkNameHost containerNetworkName = "host"
	defaultTimeout  time.Duration        = 5 * time.Minute
	dockerBin       string               = "docker"
)

// ContainerFnPermission contains the permission of container
// function such as network access.
type ContainerFnPermission struct {
	AllowNetwork bool
	AllowMount   bool
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
	// UIDGID is the os User ID and Group ID that will be
	// used to run the container in format userId:groupId.
	// If it's empty, "nobody" will be used.
	UIDGID string
	// StorageMounts are the storage or directories to mount
	// into the container
	StorageMounts []runtimeutil.StorageMount
	// Env is a slice of env string that will be exposed to container
	Env []string
}

// Run runs the container function using docker runtime.
// It reads the input from the given reader and writes the output
// to the provided writer.
func (f *ContainerFn) Run(reader io.Reader, writer io.Writer) error {
	// check and pull image before running to avoid polluting CLI
	// output
	err := f.prepareImage()
	if err != nil {
		return fmt.Errorf("failed to prepare image: %w", err)
	}

	errSink := bytes.Buffer{}
	cmd, cancel := f.getDockerCmd()
	defer cancel()
	cmd.Stdin = reader
	cmd.Stdout = writer
	cmd.Stderr = &errSink

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if goerrors.As(err, &exitErr) {
			return &errors.FnExecError{
				OriginalErr:    exitErr,
				ExitCode:       exitErr.ExitCode(),
				Stderr:         errSink.String(),
				TruncateOutput: printer.TruncateOutput,
			}
		}
		return fmt.Errorf("unexpected function error: %w", err)
	}

	return nil
}

func (f *ContainerFn) getDockerCmd() (*exec.Cmd, context.CancelFunc) {
	network := networkNameNone
	if f.Perm.AllowNetwork {
		network = networkNameHost
	}
	uidgid := "nobody"
	if f.UIDGID != "" {
		uidgid = f.UIDGID
	}

	args := []string{
		"run", "--rm", "-i",
		"-a", "STDIN", "-a", "STDOUT", "-a", "STDERR",
		"--network", string(network),
		// TODO: this env is only used in TS SDK to print the errors
		// to stderr. We don't need this once we support structured
		// results.
		"-e", "LOG_TO_STDERR=true",
		"--user", uidgid,
		"--security-opt=no-new-privileges",
	}
	for _, storageMount := range f.StorageMounts {
		args = append(args, "--mount", storageMount.String())
	}
	args = append(args,
		runtimeutil.NewContainerEnvFromStringSlice(f.Env).GetDockerFlags()...)
	args = append(args, f.Image)
	// setup container run timeout
	timeout := defaultTimeout
	if f.Timeout != 0 {
		timeout = f.Timeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	return exec.CommandContext(ctx, dockerBin, args...), cancel
}

// prepareImage will check local images and pull it if it doesn't
// exist.
func (f *ContainerFn) prepareImage() error {
	// check image existence
	args := []string{"image", "ls", f.Image}
	cmd := exec.Command(dockerBin, args...)
	var output []byte
	var err error
	if output, err = cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to check local function image %q: %w", f.Image, err)
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
	cmd = exec.CommandContext(ctx, dockerBin, args...)
	if _, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("function image %q doesn't exist", f.Image)
	}
	return nil
}
