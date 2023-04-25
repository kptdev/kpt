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
	"bufio"
	"bytes"
	"context"
	goerrors "errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	fnresult "github.com/GoogleContainerTools/kpt/pkg/api/fnresult/v1"
	"github.com/GoogleContainerTools/kpt/pkg/printer"
	"golang.org/x/mod/semver"
	"sigs.k8s.io/kustomize/kyaml/fn/runtime/runtimeutil"
)

// We may create multiple instance of ContainerFn, but we only want to check
// if container runtime is available once.
var checkContainerRuntimeOnce sync.Once

// containerNetworkName is a type for network name used in container
type containerNetworkName string

const (
	networkNameNone           containerNetworkName = "none"
	networkNameHost           containerNetworkName = "host"
	defaultLongTimeout                             = 5 * time.Minute
	versionCommandTimeout                          = 5 * time.Second
	minSupportedDockerVersion string               = "v20.10.0"

	dockerBin  string = "docker"
	podmanBin  string = "podman"
	nerdctlBin string = "nerdctl"

	ContainerRuntimeEnv = "KPT_FN_RUNTIME"

	Docker  ContainerRuntime = "docker"
	Podman  ContainerRuntime = "podman"
	Nerdctl ContainerRuntime = "nerdctl"
)

type ContainerRuntime string

// ContainerFnPermission contains the permission of container
// function such as network access.
type ContainerFnPermission struct {
	AllowNetwork bool
	AllowMount   bool
}

// ContainerFn implements a KRMFn which run a containerized
// KRM function
type ContainerFn struct {
	Ctx context.Context

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

func (r ContainerRuntime) GetBin() string {
	switch r {
	case Podman:
		return podmanBin
	case Nerdctl:
		return nerdctlBin
	default:
		return dockerBin
	}
}

// Run runs the container function using docker runtime.
// It reads the input from the given reader and writes the output
// to the provided writer.
func (f *ContainerFn) Run(reader io.Reader, writer io.Writer) error {
	// If the env var is empty, stringToContainerRuntime defaults it to docker.
	runtime, err := StringToContainerRuntime(os.Getenv(ContainerRuntimeEnv))
	if err != nil {
		return err
	}

	checkContainerRuntimeOnce.Do(func() {
		err = ContainerRuntimeAvailable(runtime)
	})
	if err != nil {
		return err
	}

	switch runtime {
	case Podman:
		return f.runCLI(reader, writer, podmanBin, filterPodmanCLIOutput)
	case Nerdctl:
		return f.runCLI(reader, writer, nerdctlBin, filterNerdctlCLIOutput)
	default:
		return f.runCLI(reader, writer, dockerBin, filterDockerCLIOutput)
	}
}

func (f *ContainerFn) runCLI(reader io.Reader, writer io.Writer, bin string, filterCLIOutputFn func(io.Reader) string) error {
	errSink := bytes.Buffer{}
	cmd, cancel := f.getCmd(bin)
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
				Stderr:         filterCLIOutputFn(&errSink),
				TruncateOutput: printer.TruncateOutput,
			}
		}
		return fmt.Errorf("unexpected function error: %w", err)
	}

	if errSink.Len() > 0 {
		f.FnResult.Stderr = filterCLIOutputFn(&errSink)
	}
	return nil
}

// getCmd assembles a command for docker, podman or nerdctl. The input binName
// is expected to be one of "docker", "podman" and "nerdctl".
func (f *ContainerFn) getCmd(binName string) (*exec.Cmd, context.CancelFunc) {
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
		"--network", string(network),
		"--user", uidgid,
		"--security-opt=no-new-privileges",
	}

	switch f.ImagePullPolicy {
	case NeverPull:
		args = append(args, "--pull", "never")
	case AlwaysPull:
		args = append(args, "--pull", "always")
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
	return exec.CommandContext(ctx, binName, args...), cancel
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

// ResolveToImageForCLI converts the function short path to the full image url.
// If the function is Catalog function, it adds "gcr.io/kpt-fn/".e.g. set-namespace:v0.1 --> gcr.io/kpt-fn/set-namespace:v0.1
func ResolveToImageForCLI(_ context.Context, image string) (string, error) {
	if !strings.Contains(image, "/") {
		return fmt.Sprintf("gcr.io/kpt-fn/%s", image), nil
	}
	return image, nil
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

// filterDockerCLIOutput filters out docker CLI messages
// from the given buffer.
func filterDockerCLIOutput(in io.Reader) string {
	s := bufio.NewScanner(in)
	var lines []string

	for s.Scan() {
		txt := s.Text()
		if !isdockerCLIoutput(txt) {
			lines = append(lines, txt)
		}
	}
	return strings.Join(lines, "\n")
}

// isdockerCLIoutput is helper method to determine if
// the given string is a docker CLI output message.
// Example docker output:
//
//	"Unable to find image 'gcr.io/kpt-fn/starlark:v0.3' locally"
//	"v0.3: Pulling from kpt-fn/starlark"
//	"4e9f2cdf4387: Already exists"
//	"aafbf7df3ddf: Pulling fs layer"
//	"aafbf7df3ddf: Verifying Checksum"
//	"aafbf7df3ddf: Download complete"
//	"6b759ab96cb2: Waiting"
//	"aafbf7df3ddf: Pull complete"
//	"Digest: sha256:c347e28606fa1a608e8e02e03541a5a46e4a0152005df4a11e44f6c4ab1edd9a"
//	"Status: Downloaded newer image for gcr.io/kpt-fn/starlark:v0.3"
func isdockerCLIoutput(s string) bool {
	if strings.Contains(s, ": Already exists") ||
		strings.Contains(s, ": Pulling fs layer") ||
		strings.Contains(s, ": Verifying Checksum") ||
		strings.Contains(s, ": Download complete") ||
		strings.Contains(s, ": Pulling from") ||
		strings.Contains(s, ": Waiting") ||
		strings.Contains(s, ": Pull complete") ||
		strings.Contains(s, "Digest: sha256") ||
		strings.Contains(s, "Status: Downloaded newer image") ||
		strings.Contains(s, "Unable to find image") {
		return true
	}
	return false
}

// filterPodmanCLIOutput filters out podman CLI messages
// from the given buffer.
func filterPodmanCLIOutput(in io.Reader) string {
	s := bufio.NewScanner(in)
	var lines []string

	for s.Scan() {
		txt := s.Text()
		if !isPodmanCLIoutput(txt) {
			lines = append(lines, txt)
		}
	}
	return strings.Join(lines, "\n")
}

var sha256Matcher = regexp.MustCompile(`^[A-Fa-f0-9]{64}$`)

// isPodmanCLIoutput is helper method to determine if
// the given string is a podman CLI output message.
// Example podman output:
//
//	"Trying to pull gcr.io/kpt-fn/starlark:v0.3..."
//	"Getting image source signatures"
//	"Copying blob sha256:aafbf7df3ddf625f4ababc8e55b4a09131651f9aac340b852b5f40b1a53deb65"
//	"Copying config sha256:17ce4f65660717ba0afbd143578dfd1c5b9822bd3ad3945c10d6878e057265f1"
//	"Writing manifest to image destination"
//	"Storing signatures"
//	"17ce4f65660717ba0afbd143578dfd1c5b9822bd3ad3945c10d6878e057265f1"
func isPodmanCLIoutput(s string) bool {
	if strings.Contains(s, "Trying to pull") ||
		strings.Contains(s, "Getting image source signatures") ||
		strings.Contains(s, "Copying blob sha256:") ||
		strings.Contains(s, "Copying config sha256:") ||
		strings.Contains(s, "Writing manifest to image destination") ||
		strings.Contains(s, "Storing signatures") ||
		sha256Matcher.MatchString(s) {
		return true
	}
	return false
}

// filterNerdctlCLIOutput filters out nerdctl CLI messages
// from the given buffer.
func filterNerdctlCLIOutput(in io.Reader) string {
	s := bufio.NewScanner(in)
	var lines []string

	for s.Scan() {
		txt := s.Text()
		if !isNerdctlCLIoutput(txt) {
			lines = append(lines, txt)
		}
	}
	return strings.Join(lines, "\n")
}

// isNerdctlCLIoutput is helper method to determine if
// the given string is a nerdctl CLI output message.
// Example nerdctl output:
// docker.io/library/hello-world:latest:                                             resolving      |--------------------------------------|
// docker.io/library/hello-world:latest:                                             resolved       |++++++++++++++++++++++++++++++++++++++|
// index-sha256:13e367d31ae85359f42d637adf6da428f76d75dc9afeb3c21faea0d976f5c651:    done           |++++++++++++++++++++++++++++++++++++++|
// manifest-sha256:f54a58bc1aac5ea1a25d796ae155dc228b3f0e11d046ae276b39c4bf2f13d8c4: done           |++++++++++++++++++++++++++++++++++++++|
// config-sha256:feb5d9fea6a5e9606aa995e879d862b825965ba48de054caab5ef356dc6b3412:   done           |++++++++++++++++++++++++++++++++++++++|
// layer-sha256:2db29710123e3e53a794f2694094b9b4338aa9ee5c40b930cb8063a1be392c54:    done           |++++++++++++++++++++++++++++++++++++++|
// elapsed: 2.4 s                                                                    total:  4.4 Ki (1.9 KiB/s)
func isNerdctlCLIoutput(s string) bool {
	if strings.Contains(s, "index-sha256:") ||
		strings.Contains(s, "Copying blob sha256:") ||
		strings.Contains(s, "manifest-sha256:") ||
		strings.Contains(s, "config-sha256:") ||
		strings.Contains(s, "layer-sha256:") ||
		strings.Contains(s, "elapsed:") ||
		strings.Contains(s, "++++++++++++++++++++++++++++++++++++++") ||
		strings.Contains(s, "--------------------------------------") ||
		sha256Matcher.MatchString(s) {
		return true
	}
	return false
}

func StringToContainerRuntime(v string) (ContainerRuntime, error) {
	switch strings.ToLower(v) {
	case string(Docker):
		return Docker, nil
	case string(Podman):
		return Podman, nil
	case string(Nerdctl):
		return Nerdctl, nil
	case "":
		return Docker, nil
	default:
		return "", fmt.Errorf("unsupported runtime: %q the runtime must be either %s or %s", v, Docker, Podman)
	}
}

func ContainerRuntimeAvailable(runtime ContainerRuntime) error {
	switch runtime {
	case Docker:
		return dockerCmdAvailable()
	case Podman:
		return podmanCmdAvailable()
	case Nerdctl:
		return nerdctlCmdAvailable()
	default:
		return dockerCmdAvailable()
	}
}

// dockerCmdAvailable runs `docker version` to check that the docker command is
// available and is a supported version. Returns an error with installation
// instructions if it is not
func dockerCmdAvailable() error {
	suggestedText := `docker must be running to use this command
To install docker, follow the instructions at https://docs.docker.com/get-docker/.
`
	cmdOut := &bytes.Buffer{}

	ctx, cancel := context.WithTimeout(context.Background(), versionCommandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, dockerBin, "version", "--format", "{{.Client.Version}}")
	cmd.Stdout = cmdOut
	err := cmd.Run()
	if err != nil || cmdOut.String() == "" {
		return fmt.Errorf("%v\n%s", err, suggestedText)
	}
	return isSupportedDockerVersion(strings.TrimSuffix(cmdOut.String(), "\n"))
}

// isSupportedDockerVersion returns an error if a given docker version is invalid
// or is less than minSupportedDockerVersion
func isSupportedDockerVersion(v string) error {
	suggestedText := fmt.Sprintf(`docker client version must be %s or greater`, minSupportedDockerVersion)
	// docker version output does not have a leading v which is required by semver, so we prefix it
	currentDockerVersion := fmt.Sprintf("v%s", v)
	if !semver.IsValid(currentDockerVersion) {
		return fmt.Errorf("%s: found invalid version %s", suggestedText, currentDockerVersion)
	}
	// if currentDockerVersion is less than minDockerClientVersion, compare returns +1
	if semver.Compare(minSupportedDockerVersion, currentDockerVersion) > 0 {
		return fmt.Errorf("%s: found %s", suggestedText, currentDockerVersion)
	}
	return nil
}

func podmanCmdAvailable() error {
	suggestedText := `podman must be installed.
To install podman, follow the instructions at https://podman.io/getting-started/installation.
`

	ctx, cancel := context.WithTimeout(context.Background(), versionCommandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, podmanBin, "version")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("%v\n%s", err, suggestedText)
	}
	return nil
}

func nerdctlCmdAvailable() error {
	suggestedText := `nerdctl must be installed.
To install nerdctl, follow the instructions at https://github.com/containerd/nerdctl#install.
`

	ctx, cancel := context.WithTimeout(context.Background(), versionCommandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, nerdctlBin, "version")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("%v\n%s", err, suggestedText)
	}
	return nil
}
