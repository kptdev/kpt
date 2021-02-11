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

package cmdutil

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-errors/errors"
	"github.com/spf13/cobra"
)

const (
	StackTraceOnErrors = "COBRA_STACK_TRACE_ON_ERRORS"
	trueString         = "true"
)

// FixDocs replaces instances of old with new in the docs for c
func FixDocs(old, new string, c *cobra.Command) {
	c.Use = strings.ReplaceAll(c.Use, old, new)
	c.Short = strings.ReplaceAll(c.Short, old, new)
	c.Long = strings.ReplaceAll(c.Long, old, new)
	c.Example = strings.ReplaceAll(c.Example, old, new)
}

func PrintErrorStacktrace(err error) {
	e := os.Getenv(StackTraceOnErrors)
	if StackOnError || e == trueString || e == "1" {
		if err, ok := err.(*errors.Error); ok {
			fmt.Fprintf(os.Stderr, "%s", err.Stack())
		}
	}
}

// StackOnError if true, will print a stack trace on failure.
var StackOnError bool

// K8sSchemaSource defines where we should look for the kubernetes openAPI
// schema
var K8sSchemaSource string

// K8sSchemaPath defines the path to the openAPI schema if we are reading from
// a file
var K8sSchemaPath string

func ResolveAbsAndRelPaths(path string) (string, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", err
	}

	var relPath string
	var absPath string
	if filepath.IsAbs(path) {
		// If the provided path is absolute, we find the relative path by
		// comparing it to the current working directory.
		relPath, err = filepath.Rel(cwd, path)
		if err != nil {
			return "", "", err
		}
		absPath = filepath.Clean(path)
	} else {
		// If the provided path is relative, we find the absolute path by
		// combining the current working directory with the relative path.
		relPath = filepath.Clean(path)
		absPath = filepath.Join(cwd, path)
	}

	return relPath, absPath, nil
}

// DockerCmdAvailable runs `docker ps` to check that the docker command is
// available, and returns an error with installation instructions if it is not
func DockerCmdAvailable() error {
	cmd := exec.Command("docker", "ps")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf(stderr.String() +
			`Docker is required to run this command. 
To install docker, follow the instructions at https://docs.docker.com/get-docker/`)
	}
	return nil
}
