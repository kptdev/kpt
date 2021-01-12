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
	"fmt"
	"io"
	"os/user"
)

// ContainerNetworkName is a type for network name used in container
type containerNetworkName string

const (
	networkNameNone containerNetworkName = "none"
	networkNameHost containerNetworkName = "host"
)

type containerFunctionPermission struct {
	AllowNetwork  bool
	AsCurrentUser bool
}

// ContainerRunner implements a KRMFn which run a containerized
// KRM function
type ContainerRunner struct {
	// Image is the container image to run
	Image string

	Exec ExecRunner

	containerFunctionPermission
}

// Run implements KRMFn. It will run the container function with
// stdin from r and write the output to w
func (f *ContainerRunner) Run(r io.Reader, w io.Writer) error {
	err := f.setupExec()
	if err != nil {
		return fmt.Errorf("error when setup exec: %w", err)
	}
	return f.Exec.Run(r, w)
}

func (f *ContainerRunner) setupExec() error {
	// don't init 2x
	if f.Exec.Path != "" {
		return nil
	}

	path, args, err := f.getCommand()
	if err != nil {
		return fmt.Errorf("error when get command and args: %w", err)
	}
	f.Exec.Path = path
	f.Exec.Args = args
	return nil
}

// getCommand returns the command + args to run to spawn the container
func (f *ContainerRunner) getCommand() (string, []string, error) {
	// run the container using docker.  this is simpler than using the docker
	// libraries, and ensures things like auth work the same as if the container
	// was run from the cli.
	network := networkNameNone
	if f.AllowNetwork {
		network = networkNameHost
	}
	UIDGID, err := getUIDGID(f.AsCurrentUser)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get current UID and GID: %w", err)
	}
	args := []string{"run",
		"--rm",                                              // delete the container afterward
		"-i", "-a", "STDIN", "-a", "STDOUT", "-a", "STDERR", // attach stdin, stdout, stderr
		"--network", string(network),
		"--user", UIDGID,
		"--security-opt=no-new-privileges", // don't allow the user to escalate privileges
		// note: don't make fs readonly because things like heredoc rely on writing tmp files
	}
	a := append(args, f.Image)
	return "docker", a, nil
}

// getUIDGID will return "nobody" if asCurrentUser is false. Otherwise
// return "uid:gid" according to the return from `user.Current()` function.
func getUIDGID(asCurrentUser bool) (string, error) {
	if !asCurrentUser {
		return "nobody", nil
	}

	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%s", u.Uid, u.Gid), nil
}

// TODO: Create ContainerRunner instance from a pipeline.Function
