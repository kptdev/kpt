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
	"os"
	"os/exec"
	"strings"

	"lib.kpt.dev/kio"

	"lib.kpt.dev/yaml"
)

// GrepFilter filters Resources using a container image.
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

	checkInput func(string)
}

// GrepFilter implements kio.GrepFilter
func (c *ContainerFilter) Filter(input []*yaml.RNode) ([]*yaml.RNode, error) {
	// get the command to filter the Resources
	cmd, err := c.getCommand()
	if err != nil {
		return nil, err
	}

	in := &bytes.Buffer{}
	out := &bytes.Buffer{}

	// write the input
	err = kio.ByteWriter{
		WrappingApiVersion: kio.ResourceListApiVersion,
		WrappingKind:       kio.ResourceListKind,
		Writer:             in, KeepReaderAnnotations: true, FunctionConfig: c.Config}.Write(input)
	if err != nil {
		return nil, err
	}

	// capture the command stdout for the return value
	r := &kio.ByteReader{Reader: out}

	// do the filtering
	if c.checkInput != nil {
		c.checkInput(in.String())
	}
	cmd.Stdin = in
	cmd.Stdout = out
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return r.Read()
}

// getArgs returns the command + args to run to spawn the container
func (c *ContainerFilter) getArgs() []string {
	// run the container using docker.  this is simpler than using the docker
	// libraries, and ensures things like auth work the same as if the container
	// was run from the cli.
	args := []string{"docker", "run",
		"--rm",                                              // delete the container afterward
		"-i", "-a", "STDIN", "-a", "STDOUT", "-a", "STDERR", // attach stdin, stdout, stderr

		// added security options
		"--network", "none", // disable the network
		"--user", "nobody", // run as nobody
		// don't make fs readonly because things like heredoc rely on writing tmp files
		"--security-opt=no-new-privileges", // don't allow the user to escalate privileges
	}

	// export the local environment vars to the container
	for _, pair := range os.Environ() {
		args = append(args, "-e", strings.Split(pair, "=")[0])
	}
	return append(args, c.Image)

}

// getCommand returns a command which will apply the GrepFilter using the container image
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
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	// set stderr for err messaging
	return cmd, nil
}
