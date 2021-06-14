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
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/GoogleContainerTools/kpt/internal/fnruntime"
	"github.com/GoogleContainerTools/kpt/internal/printer"
	"github.com/GoogleContainerTools/kpt/pkg/live"
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
)

const (
	StackTraceOnErrors                 = "COBRA_STACK_TRACE_ON_ERRORS"
	trueString                         = "true"
	Stdout                             = "stdout"
	Unwrap                             = "unwrap"
	dockerVersionTimeout time.Duration = 5 * time.Second
)

// FixDocs replaces instances of old with new in the docs for c
func FixDocs(old, new string, c *cobra.Command) {
	c.Use = strings.ReplaceAll(c.Use, old, new)
	c.Short = strings.ReplaceAll(c.Short, old, new)
	c.Long = strings.ReplaceAll(c.Long, old, new)
	c.Example = strings.ReplaceAll(c.Example, old, new)
}

func PrintErrorStacktrace() bool {
	e := os.Getenv(StackTraceOnErrors)
	if StackOnError || e == trueString || e == "1" {
		return true
	}
	return false
}

// StackOnError if true, will print a stack trace on failure.
var StackOnError bool

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
	suggestedText := `docker must be running to use this command
To install docker, follow the instructions at https://docs.docker.com/get-docker/.
`
	buffer := &bytes.Buffer{}

	ctx, cancel := context.WithTimeout(context.Background(), dockerVersionTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "version")
	cmd.Stderr = buffer
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("%s", suggestedText)
	}
	return nil
}

func ValidateImagePullPolicyValue(v string) error {
	if v != string(fnruntime.AlwaysPull) && v != string(fnruntime.IfNotPresentPull) && v != string(fnruntime.NeverPull) {
		return fmt.Errorf("image pull policy must be one of %s, %s and %s", fnruntime.AlwaysPull, fnruntime.IfNotPresentPull, fnruntime.NeverPull)
	}
	return nil
}

func StringToImagePullPolicy(v string) fnruntime.ImagePullPolicy {
	switch v {
	case string(fnruntime.NeverPull):
		return fnruntime.NeverPull
	case string(fnruntime.IfNotPresentPull):
		return fnruntime.IfNotPresentPull
	default:
		return fnruntime.AlwaysPull
	}
}

// WriteFnOutput writes the output resources of function commands to provided destination
func WriteFnOutput(dest, content string, fromStdin bool, w io.Writer) error {
	r := strings.NewReader(content)
	switch dest {
	case Stdout:
		// if user specified dest is "stdout" directly write the content as it is already wrapped
		_, err := w.Write([]byte(content))
		return err
	case Unwrap:
		// if user specified dest is "unwrap", write the unwrapped content to the provided writer
		return WriteToOutput(r, w, "")
	case "":
		if fromStdin {
			// if user didn't specify dest, and if input is from STDIN, write the wrapped content provided writer
			// this is same as "stdout" input above
			_, err := w.Write([]byte(content))
			return err
		}
	default:
		// this means user specified a directory as dest, write the content to dest directory
		return WriteToOutput(r, nil, dest)
	}
	return nil
}

// WriteToOutput reads the input from r and writes the output to either w or outDir
func WriteToOutput(r io.Reader, w io.Writer, outDir string) error {
	var outputs []kio.Writer
	if outDir != "" {
		err := os.MkdirAll(outDir, 0700)
		if err != nil {
			return fmt.Errorf("failed to create output directory %q: %q", outDir, err.Error())
		}
		outputs = []kio.Writer{&kio.LocalPackageWriter{PackagePath: outDir}}
	} else {
		outputs = []kio.Writer{&kio.ByteWriter{
			Writer:           w,
			ClearAnnotations: []string{kioutil.IndexAnnotation, kioutil.PathAnnotation}},
		}
	}

	return kio.Pipeline{
		Inputs:  []kio.Reader{&kio.ByteReader{Reader: r}},
		Outputs: outputs}.Execute()
}

// InputFromStdin returns true if the input is from stdin
func InputFromStdin() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		panic(err)
	}
	if fi.Mode()&os.ModeNamedPipe == 0 {
		return false
	}
	return true
}

// InstallResourceGroupCRD will install the ResourceGroup CRD into the cluster.
// The function will block until the CRD is either installed and established, or
// an error was encountered.
// If the CRD could not be installed, an error of the type
// ResourceGroupCRDInstallError will be returned.
func InstallResourceGroupCRD(ctx context.Context, f util.Factory) error {
	pr := printer.FromContextOrDie(ctx)
	pr.Printf("installing inventory ResourceGroup CRD.\n")
	err := live.InstallResourceGroupCRD(f)
	if err != nil {
		return &ResourceGroupCRDInstallError{
			Err: err,
		}
	}
	return nil
}

// ResourceGroupCRDInstallError is an error that will be returned if the
// ResourceGroup CRD can't be applied to the cluster.
type ResourceGroupCRDInstallError struct {
	Err error
}

func (*ResourceGroupCRDInstallError) Error() string {
	return "error installing ResourceGroup crd"
}

func (i *ResourceGroupCRDInstallError) Unwrap() error {
	return i.Err
}

// VerifyResourceGroupCRD verifies that the ResourceGroupCRD exists in
// the cluster. If it doesn't an error of type NoResourceGroupCRDError
// was returned.
func VerifyResourceGroupCRD(f util.Factory) error {
	if !live.ResourceGroupCRDApplied(f) {
		return &NoResourceGroupCRDError{}
	}
	return nil
}

// NoResourceGroupCRDError is an error type that will be used when a
// cluster doesn't have the ResourceGroup CRD installed.
type NoResourceGroupCRDError struct{}

func (*NoResourceGroupCRDError) Error() string {
	return "type ResourceGroup not found"
}

// ResolveInputSource resolves the source from which the input must be accepted
func ResolveInputSource(c *cobra.Command, args []string) ([]string, io.Reader, error) {
	if InputFromStdin() {
		if len(args) > 0 {
			return args, nil, fmt.Errorf("input can either be passed from stdin or directory but not both")
		}
		return []string{}, c.InOrStdin(), nil
	}

	if len(args) == 0 {
		// default to the current working directory
		cwd, err := os.Getwd()
		if err != nil {
			return args, nil, err
		}
		args = append(args, cwd)
	}

	return args, nil, nil
}
