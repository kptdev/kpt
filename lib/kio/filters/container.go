// Copyright 2019 Google LLC
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

package filters

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"lib.kpt.dev/kio"

	"lib.kpt.dev/yaml"
)

// Filter filters Resources using a container image.
// The container must start a process that reads the list of
// input Resources from stdin, reads the Configuration from the env
// API_CONFIG, and writes the filtered Resources to stdout.
// If there is a error or validation failure, the process must exit
// non-zero.
// The full set of environment variables from the parent process
// are passed to the container.
type ContainerFilter struct {
	// Image is the container image to use to create a container.
	Image string `yaml:"image,omitempty"`

	// Config is the API configuration for the container and passed through the
	// API_CONFIG env var to the container.
	// Typically a Kubernetes style Resource Config.
	Config *yaml.RNode `yaml:"config,omitempty"`

	// args may be specified by tests to override how a container is spawned
	args []string
}

// Filter implements kio.Filter
func (c *ContainerFilter) Filter(input []*yaml.RNode) ([]*yaml.RNode, error) {
	// get the command to filter the Resources
	cmd, err := c.getCommand()
	if err != nil {
		return nil, err
	}

	// capture the command stdout for the return value
	out := &bytes.Buffer{}
	cmd.Stdout = out

	// write the input to the command stdin
	in := &bytes.Buffer{}
	cmd.Stdin = in
	if err := (kio.ByteWriter{Writer: in}).Write(input); err != nil {
		return nil, err
	}

	// do the filtering
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	// return the filtered Resources
	var output []*yaml.RNode
	if output, err = (kio.ByteReader{Reader: out}).Read(); err != nil {
		return nil, err
	}
	for _, n := range output {
		n.YNode().Style = yaml.FoldedStyle
	}
	return output, nil
}

// getArgs returns the command + args to run to spawn the container
func (c *ContainerFilter) getArgs() []string {
	// configure the environment to contain the configuration
	env := []string{"API_CONFIG"}
	for _, pair := range os.Environ() {
		env = append(env, strings.Split(pair, "=")[0])
	}

	// run the container using docker.  this is simpler than using the docker
	// libraries, and ensures things like auth work the same as if the container
	// was run from the cli.
	args := []string{"docker", "run",
		"--rm",              // delete the container afterward
		"-i",                // enable stdin
		"--network", "none", // disable the network for added security
	}

	// export the local environment vars to the container
	for _, e := range env {
		args = append(args, "-e", e)
	}
	return append(args, c.Image)

}

// getCommand returns a command which will apply the Filter using the container image
func (c *ContainerFilter) getCommand() (*exec.Cmd, error) {
	// encode the filter command API configuration
	cfg := &bytes.Buffer{}
	if err := func() error {
		e := yaml.NewEncoder(cfg)
		defer e.Close()
		// make it fit on a single line
		c.Config.YNode().Style = yaml.FlowStyle
		return e.Encode(c.Config.YNode())
	}(); err != nil {
		return nil, err
	}

	if len(c.args) == 0 {
		c.args = c.getArgs()
	}

	cmd := exec.Command(c.args[0], c.args[1:]...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("API_CONFIG=%s", cfg.String()))

	// set stderr for err messaging
	cmd.Stderr = os.Stderr
	return cmd, nil
}
