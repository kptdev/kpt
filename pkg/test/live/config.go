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

package live

import (
	"os"
	"path/filepath"
	"testing"

	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type TestCaseConfig struct {
	// ExitCode is the expected exit code from the kpt commands. Default: 0
	ExitCode int `yaml:"exitCode,omitempty"`

	// StdErr is the expected standard error output. Default: ""
	StdErr string `yaml:"stdErr,omitempty"`

	// StdOut is the expected standard output from running the command.
	// Default: ""
	StdOut string `yaml:"stdOut,omitempty"`

	// OptionalStdOut is a list of lines that are optional in the standard
	// output from running the command.
	// Default: nil
	OptionalStdOut []string `yaml:"optionalStdOut,omitempty"`

	// Inventory is the expected list of resource present in the inventory.
	Inventory []InventoryEntry `yaml:"inventory,omitempty"`

	// NoResourceGroup defines whether the RG CRD should be present in the cluster
	// when the test starts.
	NoResourceGroup bool `yaml:"noResourceGroup,omitempty"`

	// Parallel defines whether the test can be run in parallel with other
	// tests. The primary requirement here is that the test doesn't create,
	// update, or delete any cluster-scoped resources.
	Parallel bool `yaml:"parallel,omitempty"`

	// KptArgs is a list of args that will be provided to the kpt command
	// when running the test.
	KptArgs []string `yaml:"kptArgs,omitempty"`
}

// InventoryEntry defines an entry in an inventory list.
type InventoryEntry struct {
	Group     string `yaml:"group,omitempty"`
	Kind      string `yaml:"kind,omitempty"`
	Name      string `yaml:"name,omitempty"`
	Namespace string `yaml:"namespace,omitempty"`
}

func ReadTestCaseConfig(t *testing.T, path string) TestCaseConfig {
	configPath := filepath.Join(path, "config.yaml")
	b, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("unable to read test config at %s", configPath)
	}

	var config TestCaseConfig
	err = yaml.Unmarshal(b, &config)
	if err != nil {
		t.Fatalf("unable to unmarshal test config file %s: %v", configPath, err)
	}
	return config
}
