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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/GoogleContainerTools/kpt/internal/fnruntime"
	"github.com/GoogleContainerTools/kpt/internal/util/httputil"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
)

const (
	StackTraceOnErrors                      = "COBRA_STACK_TRACE_ON_ERRORS"
	trueString                              = "true"
	Stdout                                  = "stdout"
	Unwrap                                  = "unwrap"
	dockerVersionTimeout      time.Duration = 5 * time.Second
	FunctionsCatalogURL                     = "https://catalog.kpt.dev/catalog-v2.json"
	minSupportedDockerVersion string        = "v20.10.0"
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

// DockerCmdAvailable runs `docker version` to check that the docker command is
// available and is a supported version. Returns an error with installation
// instructions if it is not
func DockerCmdAvailable() error {
	suggestedText := `docker must be running to use this command
To install docker, follow the instructions at https://docs.docker.com/get-docker/.
`
	cmdOut := &bytes.Buffer{}

	ctx, cancel := context.WithTimeout(context.Background(), dockerVersionTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "version", "--format", "{{.Client.Version}}")
	cmd.Stdout = cmdOut
	err := cmd.Run()
	if err != nil || cmdOut.String() == "" {
		return fmt.Errorf("%s", suggestedText)
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

func ValidateImagePullPolicyValue(v string) error {
	v = strings.ToLower(v)
	if v != strings.ToLower(string(fnruntime.AlwaysPull)) &&
		v != strings.ToLower(string(fnruntime.IfNotPresentPull)) &&
		v != strings.ToLower(string(fnruntime.NeverPull)) {
		return fmt.Errorf("image pull policy must be one of %s, %s and %s", fnruntime.AlwaysPull, fnruntime.IfNotPresentPull, fnruntime.NeverPull)
	}
	return nil
}

func StringToImagePullPolicy(v string) fnruntime.ImagePullPolicy {
	switch strings.ToLower(v) {
	case strings.ToLower(string(fnruntime.NeverPull)):
		return fnruntime.NeverPull
	case strings.ToLower(string(fnruntime.IfNotPresentPull)):
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

// FetchFunctionImages returns the list of latest function images from catalog.kpt.dev
func FetchFunctionImages() []string {
	content, err := httputil.FetchContent(FunctionsCatalogURL)
	if err != nil {
		return nil
	}

	return listImages(content)
}

// fnName -> v<major>.<minor> -> catalogEntry
type catalogV2 map[string]map[string]struct {
	LatestPatchVersion string
	Examples           interface{}
}

// listImages returns the list of latest images from the input catalog content
func listImages(content string) []string {
	var result []string
	var jsonData catalogV2
	err := json.Unmarshal([]byte(content), &jsonData)
	if err != nil {
		return result
	}
	for fnName, fnInfo := range jsonData {
		var latestVersion string
		for _, catalogEntry := range fnInfo {
			version := catalogEntry.LatestPatchVersion
			if semver.Compare(version, latestVersion) == 1 {
				latestVersion = version
			}
		}
		result = append(result, fmt.Sprintf("%s:%s", fnName, latestVersion))
	}
	return result
}
