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

// some of these codes are copied from
// - https://github.com/kubernetes-sigs/kustomize/blob/master/kyaml/fn/runtime/container/container.go
// - https://github.com/kubernetes-sigs/kustomize/blob/master/kyaml/fn/runtime/exec/exec.go

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"time"
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
	AllowNetwork  bool
	AsCurrentUser bool
}

// ContainerFn implements a KRMFn which run a containerized
// KRM function
type ContainerFn struct {
	// Image is the container image to run
	Image string
	// Container function will be killed after this timeour.
	// The default value is 5 minutes.
	Timeout time.Duration
	Perm    ContainerFnPermission
}

// Run implements KRMFn. It will run the container function with
// stdin from r and write the output to w
func (f *ContainerFn) Run(r io.Reader, w io.Writer) error {
	// run the container using docker.  this is simpler than using the docker
	// libraries, and ensures things like auth work the same as if the container
	// was run from the cli.
	path := "docker"

	network := networkNameNone
	if f.Perm.AllowNetwork {
		network = networkNameHost
	}
	UIDGID, err := getUIDGID(f.Perm.AsCurrentUser)
	if err != nil {
		return fmt.Errorf("failed to get current UID and GID: %w", err)
	}
	args := []string{"run",
		"--rm",                                              // delete the container afterward
		"-i", "-a", "STDIN", "-a", "STDOUT", "-a", "STDERR", // attach stdin, stdout, stderr
		"--network", string(network),
		"--user", UIDGID,
		"--security-opt=no-new-privileges", // don't allow the user to escalate privileges
		// note: don't make fs readonly because things like heredoc rely on writing tmp files
	}
	args = append(args, f.Image)

	timeout := defaultTimeout
	if f.Timeout != 0 {
		timeout = f.Timeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Stdin = r
	cmd.Stdout = w
	cmd.Stderr = os.Stderr
	return cmd.Run()
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

// TODO: Create ContainerFn instance from a pipeline.Function
