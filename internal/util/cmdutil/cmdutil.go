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
	"strings"
	"time"

	"github.com/GoogleContainerTools/kpt/internal/fnruntime"
	"github.com/GoogleContainerTools/kpt/internal/util/function"
	"github.com/GoogleContainerTools/kpt/internal/util/httputil"
	"github.com/GoogleContainerTools/kpt/internal/util/porch"
	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
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
	minSupportedDockerVersion string        = "v20.10.0"
	FunctionsCatalogURL                     = "https://catalog.kpt.dev/catalog-v2.json"
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

// SuggestFunctions looks for functions from kpt curated catalog list as well as the Porch
// orchestrator to suggest functions.
func SuggestFunctions(cmd *cobra.Command) []string {
	matchers := []function.Matcher{
		function.TypeMatcher{FnType: cmd.Flag("type").Value.String()},
		function.KeywordsMatcher{Keywords: GetKeywordsFromFlag(cmd)},
	}
	functions := DiscoverFunctions(cmd)
	matched := function.MatchFunctions(functions, matchers...)
	return function.GetNames(matched)
}

// SuggestKeywords looks for all the unique keywords from Porch functions. This keywords
// can later help users to select functions.
func SuggestKeywords(cmd *cobra.Command) []string {
	functions := DiscoverFunctions(cmd)
	matched := function.MatchFunctions(functions, function.TypeMatcher{FnType: cmd.Flag("type").Value.String()})
	return porch.UnifyKeywords(matched)
}

func DiscoverFunctions(cmd *cobra.Command) []v1alpha1.Function {
	porchFns := porch.FunctionListGetter{}.Get(cmd.Context())
	catalogV2Fns := fetchCatalogFunctions()
	return append(porchFns, catalogV2Fns...)
}

// fetchCatalogFunctions returns the list of latest function images from catalog.kpt.dev.
func fetchCatalogFunctions() []v1alpha1.Function {
	content, err := httputil.FetchContent(FunctionsCatalogURL)
	if err != nil {
		return nil
	}
	return parseFunctions(content)
}

// fnName -> v<major>.<minor> -> catalogEntry
type catalogV2 map[string]map[string]struct {
	LatestPatchVersion string
	Examples           interface{}
	Types              []string
	Keywords           []string
}

// listImages returns the list of latest images from the input catalog content
func parseFunctions(content string) []v1alpha1.Function {
	var jsonData catalogV2
	err := json.Unmarshal([]byte(content), &jsonData)
	if err != nil {
		return nil
	}
	var functions []v1alpha1.Function
	for fnName, fnInfo := range jsonData {
		var latestVersion string
		var keywords []string
		var fnTypes []v1alpha1.FunctionType
		for _, catalogEntry := range fnInfo {
			version := catalogEntry.LatestPatchVersion
			if semver.Compare(version, latestVersion) == 1 {
				latestVersion = version
				keywords = catalogEntry.Keywords
				for _, tp := range catalogEntry.Types {
					switch tp {
					case "validator":
						fnTypes = append(fnTypes, v1alpha1.FunctionTypeValidator)
					case "mutator":
						fnTypes = append(fnTypes, v1alpha1.FunctionTypeMutator)
					}
				}
			}
		}
		fnName := fmt.Sprintf("%s:%s", fnName, latestVersion)
		functions = append(functions, function.CatalogFunction(fnName, keywords, fnTypes))
	}
	return functions
}
