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

	AlwaysPull       ImagePullPolicy = "always"
	IfNotPresentPull ImagePullPolicy = "ifNotPresent"
	NeverPull        ImagePullPolicy = "never"
)

type ImagePullPolicy string

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
	// ImagePullPolicy controls the image pulling behavior.
	ImagePullPolicy ImagePullPolicy
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
		return err
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
			return &ExecError{
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
		"--user", uidgid,
		"--security-opt=no-new-privileges",
	}
	if f.ImagePullPolicy == NeverPull {
		args = append(args, "--pull", "never")
	}
	for _, storageMount := range f.StorageMounts {
		args = append(args, "--mount", storageMount.String())
	}
	args = append(args,
		NewContainerEnvFromStringSlice(f.Env).GetDockerFlags()...)
	args = append(args, f.Image)
	// setup container run timeout
	timeout := defaultTimeout
	if f.Timeout != 0 {
		timeout = f.Timeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	return exec.CommandContext(ctx, dockerBin, args...), cancel
}

// NewContainerEnvFromStringSlice returns a new ContainerEnv pointer with parsing
// input envStr. envStr example: ["foo=bar", "baz"]
// using this instead of runtimeutil.NewContainerEnvFromStringSlice() to avoid
// default envs LOG_TO_STDERR
func NewContainerEnvFromStringSlice(envStr []string) *runtimeutil.ContainerEnv {
	ce := &runtimeutil.ContainerEnv{
		EnvVars: make(map[string]string),
	}
	// default envs
	for _, e := range envStr {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 1 {
			ce.AddKey(e)
		} else {
			ce.AddKeyValue(parts[0], parts[1])
		}
	}
	return ce
}

// prepareImage will check local images and pull it if it doesn't
// exist.
func (f *ContainerFn) prepareImage() error {
	// If ImagePullPolicy is set to "never", we don't need to do anything here.
	if f.ImagePullPolicy == NeverPull {
		return nil
	}

	// check image existence
	foundImageInLocalCache := false
	args := []string{"image", "inspect", f.Image}
	cmd := exec.Command(dockerBin, args...)
	var output []byte
	var err error
	if _, err = cmd.CombinedOutput(); err == nil {
		// image exists locally
		foundImageInLocalCache = true
	}

	// If ImagePullPolicy is set to "ifNotPresent", we scan the local images
	// first. If there is a match, we just return. This can be useful for local
	// development to prevent the remote image to accidentally override the
	// local image when they use the same name and tag.
	if f.ImagePullPolicy == IfNotPresentPull && foundImageInLocalCache {
		return nil
	}

	// If ImagePullPolicy is set to always (which is the default), we will try
	// to pull the image regardless if the tag has been seen in the local cache.
	// This can help to ensure we have the latest release for "moving tags" like
	// v1 and v1.2. The performance cost is very minimal, since `docker pull`
	// checks the SHA first and only pull the missing docker layer(s).
	args = []string{"image", "pull", f.Image}
	// setup timeout
	timeout := defaultTimeout
	if f.Timeout != 0 {
		timeout = f.Timeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd = exec.CommandContext(ctx, dockerBin, args...)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return &ContainerImageError{
			Image:  f.Image,
			Output: string(output),
		}
	}
	return nil
}

// ContainerImageError is an error type which will be returned when
// the container run time cannot verify docker image.
type ContainerImageError struct {
	Image  string
	Output string
}

func (e *ContainerImageError) Error() string {
	return fmt.Sprintf(
		"Function image %q doesn't exist remotely. If you are developing new functions locally, you can choose to set the image pull policy to ifNotPresent or never.\n%v",
		e.Image, e.Output)
}
