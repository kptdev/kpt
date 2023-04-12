// Copyright 2022 The kpt Authors
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

package porch

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

type Command struct {
	// Args is a list of args for the kpt CLI.
	Args []string `yaml:"args,omitempty"`
	// StdIn contents will be passed as the command's standard input, if not empty.
	Stdin string `yaml:"stdin,omitempty"`
	// StdOut is the standard output expected from the command.
	Stdout string `yaml:"stdout,omitempty"`
	// StdErr is the standard error output expected from the command.
	Stderr string `yaml:"stderr,omitempty"`
	// ExitCode is the expected exit code frm the command.
	ExitCode int `yaml:"exitCode,omitempty"`
	// Yaml indicates that stdout is yaml and the test will reformat it for stable ordering
	Yaml bool `yaml:"yaml,omitempty"`
}

type TestCaseConfig struct {
	// TestCase is the name of the test case.
	TestCase string `yaml:"-"`
	// ConfigFile stores the name of the config file from which the config was loaded.
	// Used when generating or updating golden files.
	ConfigFile string `yaml:"-"`
	// Repository is the name of the k8s Repository resource to register the default Git repo.
	Repository string `yaml:"repository,omitempty"`
	// Commands is a list of kpt commands to be executed by the test.
	Commands []Command `yaml:"commands,omitempty"`
	// Skip the test? If the value is not empty, it will be used as a message with which to skip the test.
	Skip string `yaml:"skip,omitempty"`
}

func ReadTestCaseConfig(t *testing.T, name, path string) TestCaseConfig {
	configPath := filepath.Join(path, "config.yaml")
	b, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read test config file %q: %v", configPath, err)
	}

	var tc TestCaseConfig
	if err := yaml.Unmarshal(b, &tc); err != nil {
		t.Fatalf("Failed to unmarshal test config %q: %v", configPath, err)
	}

	tc.TestCase = name
	tc.ConfigFile = configPath
	return tc
}

func WriteTestCaseConfig(t *testing.T, tc *TestCaseConfig) {
	var out bytes.Buffer
	e := yaml.NewEncoder(&out)
	e.SetIndent(2)
	if err := e.Encode(tc); err != nil {
		t.Fatalf("Failed to marshal test case config for %s: %v", tc.TestCase, err)
	}
	if err := os.WriteFile(tc.ConfigFile, out.Bytes(), 0644); err != nil {
		t.Errorf("Failed to save test case config for %s into %q: %v", tc.TestCase, tc.ConfigFile, err)
	}
}
