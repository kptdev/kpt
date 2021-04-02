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

package runner

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// EvalTestCaseConfig contains the config only for imperative
// function run
type EvalTestCaseConfig struct {
	// ExecPath is a path to the executable file that will be run as function
	// Mutually exclusive with Image
	ExecPath string `json:"execPath,omitempty" yaml:"execPath,omitempty"`
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
	FnConfig string `json:"fnConfig,omitempty" yaml:"fnConfig,omitempty"`
}

// TestCaseConfig contains the config information for the test case
type TestCaseConfig struct {
	// ExitCode is the expected exit code from the kpt commands. Default: 0
	ExitCode int `json:"exitCode,omitempty" yaml:"exitCode,omitempty"`
	// RunCount is how many times the test will run. Each iteration will be
	// immediately run after previous one. Default: 1
	RunCount int `json:"runCount,omitempty" yaml:"runCount,omitempty"`
	// Skip means should this test case be skipped. Default: false
	Skip bool `json:"skip,omitempty" yaml:"skip,omitempty"`
	// Debug means will the debug behavior be enabled. Default: false
	// Debug behavior:
	//  1. Keep the temporary directory used to run the test cases
	//    after test.
	Debug bool `json:"debug,omitempty" yaml:"debug,omitempty"`
	// TestType is the type of the test case. Possible value: ['render', 'eval']
	// Default: 'eval'
	TestType string `json:"testType,omitempty" yaml:"testType,omitempty"`
	// EvalConfig is the configs for eval tests
	EvalConfig *EvalTestCaseConfig `json:",inline" yaml:",inline"`
}

func newTestCaseConfig(path string) (TestCaseConfig, error) {
	configPath := filepath.Join(path, expectedDir, expectedConfigFile)
	b, err := ioutil.ReadFile(configPath)
	if os.IsNotExist(err) {
		// return default config
		return TestCaseConfig{
			RunCount: 1,
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
	if config.RunCount == 0 {
		config.RunCount = 1
	}
	if config.TestType == "" {
		// by default we test pipeline
		config.TestType = CommandFnRender
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
