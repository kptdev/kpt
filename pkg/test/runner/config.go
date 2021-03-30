package runner

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// TestCaseConfig contains the config information for the test case
type TestCaseConfig struct {
	ExitCode int  `json:"exitCode,omitempty" yaml:"exitCode,omitempty"`
	Network  bool `json:"network,omitempty" yaml:"network,omitempty"`
	RunCount int  `json:"runCount,omitempty" yaml:"runCount,omitempty"`
	Skip     bool `json:"skip,omitempty" yaml:"skip,omitempty"`
	Debug    bool `json:"debug,omitempty" yaml:"debug,omitempty"`
}

func newTestCaseConfig(path string) (TestCaseConfig, error) {
	configPath := filepath.Join(path, expectedDir, expectedConfigFile)
	b, err := ioutil.ReadFile(configPath)
	if os.IsNotExist(err) {
		// return default config
		return TestCaseConfig{
			ExitCode: 0,
			Network:  false,
			RunCount: 1,
			Skip:     false,
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
