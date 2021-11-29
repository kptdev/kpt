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
	"bufio"
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
	fnresult "github.com/GoogleContainerTools/kpt/pkg/api/fnresult/v1"
	"sigs.k8s.io/kustomize/kyaml/fn/runtime/runtimeutil"
)

// containerNetworkName is a type for network name used in container
type containerNetworkName string

const (
	networkNameNone    containerNetworkName = "none"
	networkNameHost    containerNetworkName = "host"
	defaultLongTimeout time.Duration        = 5 * time.Minute
	dockerBin          string               = "docker"

	AlwaysPull       ImagePullPolicy = "Always"
	IfNotPresentPull ImagePullPolicy = "IfNotPresent"
	NeverPull        ImagePullPolicy = "Never"
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
	// FnResult is used to store the information about the result from
	// the function.
	FnResult *fnresult.Result
}

// Run runs the container function using docker runtime.
// It reads the input from the given reader and writes the output
// to the provided writer.
func (f *ContainerFn) Run(reader io.Reader, writer io.Writer) error {
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
				Stderr:         filterDockerCLIOutput(&errSink),
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
	switch f.ImagePullPolicy {
	case NeverPull:
		args = append(args, "--pull", "never")
	case AlwaysPull:
		args = append(args, "--pull", "pull")
	case IfNotPresentPull:
		args = append(args, "--pull", "missing")
	default:
		args = append(args, "--pull", "missing")
	}
	for _, storageMount := range f.StorageMounts {
		args = append(args, "--mount", storageMount.String())
	}
	args = append(args,
		NewContainerEnvFromStringSlice(f.Env).GetDockerFlags()...)
	args = append(args, f.Image)
	// setup container run timeout
	timeout := defaultLongTimeout
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

// AddDefaultImagePathPrefix adds default gcr.io/kpt-fn/ path prefix to image if only image name is specified
func AddDefaultImagePathPrefix(image string) string {
	if !strings.Contains(image, "/") {
		return fmt.Sprintf("gcr.io/kpt-fn/%s", image)
	}
	return image
}

// ContainerImageError is an error type which will be returned when
// the container run time cannot verify docker image.
type ContainerImageError struct {
	Image  string
	Output string
}

func (e *ContainerImageError) Error() string {
	//nolint:lll
	return fmt.Sprintf(
		"Error: Function image %q doesn't exist remotely. If you are developing new functions locally, you can choose to set the image pull policy to ifNotPresent or never.\n%v",
		e.Image, e.Output)
}

/*
	"Unable to find image 'gcr.io/kpt-fn/starlark:v0.3' locally"
    "v0.3: Pulling from kpt-fn/starlark"
    "4e9f2cdf4387: Already exists"
    "aafbf7df3ddf: Pulling fs layer"
    "aafbf7df3ddf: Verifying Checksum"
    "aafbf7df3ddf: Download complete"
    "aafbf7df3ddf: Pull complete"
    "Digest: sha256:c347e28606fa1a608e8e02e03541a5a46e4a0152005df4a11e44f6c4ab1edd9a"
    "Status: Downloaded newer image for gcr.io/kpt-fn/starlark:v0.3"
*/

// filterDockerCLIOutput filters out docker CLI messages
// from the given buffer.
func filterDockerCLIOutput(in io.Reader) string {
	out := strings.Builder{}

	s := bufio.NewScanner(in)

	for s.Scan() {
		txt := s.Text()
		if !isdockerCLIoutput(txt) {
			out.WriteString(txt)
			out.WriteString("\n")
		}
	}
	return out.String()
}

// isdockerCLIoutput is helper method to determine if
// the given string is a docker CLI output message.
func isdockerCLIoutput(s string) bool {
	if strings.Contains(s, "Already exists") ||
		strings.Contains(s, "Pulling fs layer") ||
		strings.Contains(s, "Verifying Checksum") ||
		strings.Contains(s, "Download complete") ||
		strings.Contains(s, "Pulling from") ||
		strings.Contains(s, "Pull complete") ||
		strings.Contains(s, "Digest: sha256") ||
		strings.Contains(s, "Status: Downloaded newer image") ||
		strings.Contains(s, "Unable to find image") {
		return true
	}
	return false
}
