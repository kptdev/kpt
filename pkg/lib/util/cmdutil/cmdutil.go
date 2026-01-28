// Copyright 2019 The kpt Authors
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
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/kptdev/kpt/pkg/live"
	"github.com/kptdev/kpt/pkg/printer"
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
)

const (
	StackTraceOnErrors = "COBRA_STACK_TRACE_ON_ERRORS"
	trueString         = "true"
	Stdout             = "stdout"
	Unwrap             = "unwrap"
)

// FixDocs replaces instances of old with new in the docs for c
func FixDocs(oldVal, newVal string, c *cobra.Command) {
	c.Use = strings.ReplaceAll(c.Use, oldVal, newVal)
	c.Short = strings.ReplaceAll(c.Short, oldVal, newVal)
	c.Long = strings.ReplaceAll(c.Long, oldVal, newVal)
	c.Example = strings.ReplaceAll(c.Example, oldVal, newVal)
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
		err := os.MkdirAll(outDir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create output directory %q: %q", outDir, err.Error())
		}
		outputs = []kio.Writer{&kio.LocalPackageWriter{PackagePath: outDir}}
	} else {
		outputs = []kio.Writer{&kio.ByteWriter{
			Writer: w,
			ClearAnnotations: []string{kioutil.IndexAnnotation, kioutil.PathAnnotation,
				kioutil.LegacyIndexAnnotation, kioutil.LegacyPathAnnotation}}, // nolint:staticcheck
		}
	}

	return kio.Pipeline{
		Inputs:  []kio.Reader{&kio.ByteReader{Reader: r, PreserveSeqIndent: true, WrapBareSeqNode: true}},
		Outputs: outputs}.Execute()
}

// CheckDirectoryNotPresent returns error if the directory already exists
func CheckDirectoryNotPresent(outDir string) error {
	_, err := os.Stat(outDir)
	if err == nil || os.IsExist(err) {
		return fmt.Errorf("directory %q already exists, please delete the directory and retry", outDir)
	}
	if !os.IsNotExist(err) {
		return err
	}
	return nil
}

func GetKeywordsFromFlag(cmd *cobra.Command) []string {
	flagVal := cmd.Flag("keywords").Value.String()
	flagVal = strings.TrimPrefix(flagVal, "[")
	flagVal = strings.TrimSuffix(flagVal, "]")
	splitted := strings.Split(flagVal, ",")
	var trimmed []string
	for _, val := range splitted {
		if strings.TrimSpace(val) == "" {
			continue
		}
		trimmed = append(trimmed, strings.TrimSpace(val))
	}
	return trimmed
}

// InstallResourceGroupCRD will install the ResourceGroup CRD into the cluster.
// The function will block until the CRD is either installed and established, or
// an error was encountered.
// If the CRD could not be installed, an error of the type
// ResourceGroupCRDInstallError will be returned.
func InstallResourceGroupCRD(ctx context.Context, f util.Factory) error {
	pr := printer.FromContextOrDie(ctx)
	pr.Printf("installing inventory ResourceGroup CRD.\n")
	err := (&live.ResourceGroupInstaller{
		Factory: f,
	}).InstallRG(ctx)
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

// ResourceGroupCRDNotLatestError is an error type that will be used when a
// cluster has a ResourceGroup CRD that doesn't match the
// latest declaration.
type ResourceGroupCRDNotLatestError struct {
	Err error
}

func (e *ResourceGroupCRDNotLatestError) Error() string {
	return fmt.Sprintf(
		"Type ResourceGroup CRD needs update. Please make sure you have the permission "+
			"to update CRD then run `kpt live install-resource-group`.\n %v", e.Err)
}
