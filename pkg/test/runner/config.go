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

package runner

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/internal/types"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// EvalTestCaseConfig contains the config only for imperative
// function run
type EvalTestCaseConfig struct {
	// ExecPath is a path to the executable file that will be run as function
	// Mutually exclusive with Image.
	// The path should be separated by slash '/'
	ExecPath string `json:"execPath,omitempty" yaml:"execPath,omitempty"`
	// execUniquePath is an absolute, OS-specific path to exec file.
	execUniquePath types.UniquePath
	// Image is the image name for the function
	Image string `json:"image,omitempty" yaml:"image,omitempty"`
	// Args are the arguments that will be passed into function.
	// Args will be passed as 'key=value' format after the '--' in command.
	Args map[string]string `json:"args,omitempty" yaml:"args,omitempty"`
	// Network indicates is network accessible from the function container. Default: false
	Network bool `json:"network,omitempty" yaml:"network,omitempty"`
	// IncludeMetaResources enables including meta resources, like Kptfile,
	// in the function input. Default: false
	IncludeMetaResources bool `json:"includeMetaResources,omitempty" yaml:"includeMetaResources,omitempty"`
	// FnConfig is the path to the function config file.
	// The path should be separated by slash '/'
	FnConfig string `json:"fnConfig,omitempty" yaml:"fnConfig,omitempty"`
	// fnConfigUniquePath is an absolute, OS-specific path to function config file.
	fnConfigUniquePath types.UniquePath
}

// TestCaseConfig contains the config information for the test case
type TestCaseConfig struct {
	// ExitCode is the expected exit code from the kpt commands. Default: 0
	ExitCode int `json:"exitCode,omitempty" yaml:"exitCode,omitempty"`

	// StdErr is the expected standard error output and should be checked
	// when a nonzero exit code is expected. Default: ""
	StdErr string `json:"stdErr,omitempty" yaml:"stdErr,omitempty"`
	// StdErrRegEx is the regular expression to match standard error output and should be checked
	// when a nonzero exit code is expected. Default: ""
	StdErrRegEx string `json:"stdErrRegEx,omitempty" yaml:"stdErrRegEx,omitempty"`

	// StdOut is the expected standard output from running the command.
	// Default: ""
	StdOut string `json:"stdOut,omitempty" yaml:"stdOut,omitempty"`

	// Sequential means should this test case be run sequentially. Default: false
	Sequential bool `json:"sequential,omitempty" yaml:"sequential,omitempty"`

	// ImagePullPolicy controls the image pulling behavior. It can be set to one
	// of `Always`, `IfNotPresent` and `Never`. If unspecified, the default will
	// be the same as the CLI flag.
	ImagePullPolicy string `json:"imagePullPolicy,omitempty" yaml:"imagePullPolicy,omitempty"`

	// Runtimes controls if a test case should be skipped. If the current runtime doesn't match
	// any of the desired runtimes here, the test case will be skipped. Valid values are `docker`
	// and `podman`. If unspecified, it will match any runtime.
	Runtimes []string `json:"runtimes,omitempty" yaml:"runtimes,omitempty"`

	// AllowExec determines if `fn render` needs to be invoked with `--allow-exec` flag
	AllowExec bool `json:"allowExec,omitempty" yaml:"allowExec,omitempty"`

	// AllowExec determines if `fn render` needs to be invoked with `--allow-network` flag
	AllowNetwork bool `json:"allowNetwork,omitempty" yaml:"allowNetwork,omitempty"`

	// AllowWasm determines if `fn render` needs to be invoked with `--allow-alpha-wasm` flag
	AllowWasm bool `json:"allowWasm,omitempty" yaml:"allowWasm,omitempty"`

	// Skip means should this test case be skipped. Default: false
	Skip bool `json:"skip,omitempty" yaml:"skip,omitempty"`

	// Debug means will the debug behavior be enabled. Default: false
	// Debug behavior:
	//  1. Keep the temporary directory used to run the test cases
	//    after test.
	Debug bool `json:"debug,omitempty" yaml:"debug,omitempty"`

	// TestType is the type of the test case. Possible value: ['render', 'eval']
	// Default: 'render'
	TestType string `json:"testType,omitempty" yaml:"testType,omitempty"`

	// DisableOutputTruncate indicates should error output be truncated
	DisableOutputTruncate bool `json:"disableOutputTruncate,omitempty" yaml:"disableOutputTruncate,omitempty"`

	// EvalConfig is the configs for eval tests
	EvalConfig *EvalTestCaseConfig `json:",inline" yaml:",inline"`

	// Environment variables to be set for the test case.
	Env map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
}

func (c *TestCaseConfig) RunCount() int {
	return 2
}

func newTestCaseConfig(path string) (TestCaseConfig, error) {
	configPath := filepath.Join(path, expectedDir, expectedConfigFile)
	b, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		// return default config
		return TestCaseConfig{
			TestType: CommandFnRender,
		}, nil
	}
	if err != nil {
		return TestCaseConfig{}, fmt.Errorf("filed to read test config file: %w", err)
	}

	var config TestCaseConfig
	err = yaml.Unmarshal(b, &config)
	if err != nil {
		return config, fmt.Errorf("failed to unmarshal config file: %s\n: %w", string(b), err)
	}
	if config.TestType == "" {
		// by default we test pipeline
		config.TestType = CommandFnRender
	}
	if config.EvalConfig != nil {
		config.EvalConfig.fnConfigUniquePath, err = fromSlashPath(filepath.Join(path, expectedDir), config.EvalConfig.FnConfig)
		if err != nil {
			return config, fmt.Errorf("failed to get UniquePath from slash path %s: %w",
				config.EvalConfig.FnConfig, err)
		}
		config.EvalConfig.execUniquePath, err = fromSlashPath(filepath.Join(path, expectedDir), config.EvalConfig.ExecPath)
		if err != nil {
			return config, fmt.Errorf("failed to get UniquePath from slash path %s: %w",
				config.EvalConfig.ExecPath, err)
		}
	}
	return config, nil
}

// TestCase contains the information needed to run a test. Each test case
// run by this driver is described by a `TestCase`.
type TestCase struct {
	Path   string
	Config TestCaseConfig
}

// TestCases contains a list of TestCase.
type TestCases []TestCase

func isTestCase(path string, info os.FileInfo) bool {
	if !info.IsDir() {
		return false
	}

	expectedPath := filepath.Join(path, expectedDir)
	expectedInfo, err := os.Stat(expectedPath)
	if err != nil {
		return false
	}
	if !expectedInfo.IsDir() {
		return false
	}
	return true
}

// ScanTestCases will recursively scan the directory `path` and return
// a list of TestConfig found
func ScanTestCases(path string) (*TestCases, error) {
	var cases TestCases
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !isTestCase(path, info) {
			return nil
		}

		config, err := newTestCaseConfig(path)
		if err != nil {
			return err
		}

		cases = append(cases, TestCase{
			Path:   path,
			Config: config,
		})

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan test cases in %s", path)
	}
	return &cases, nil
}

// fromSlashPath returns a UniquePath according to the input slash 'path'.
// 'base' should be an OS-specific base path which will be joined with 'path'
// if 'path' is not absolute.
func fromSlashPath(base, path string) (types.UniquePath, error) {
	if path == "" {
		return types.UniquePath(""), nil
	}
	path = filepath.FromSlash(path)
	if filepath.IsAbs(path) {
		return types.UniquePath(path), nil
	}
	p, err := filepath.Abs(filepath.Join(base, path))
	if err != nil {
		return "", err
	}
	return types.UniquePath(p), nil
}
