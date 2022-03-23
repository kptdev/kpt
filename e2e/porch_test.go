// Copyright 2022 Google LLC
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

//go:build porch

package e2e

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/kpt/pkg/test/porch"
	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

const (
	testGitNamespace     = "test-git-namespace"
	gitRepositoryAddress = "http://git-server." + testGitNamespace + ".svc.cluster.local:8080"
)

var (
	update = flag.Bool("update", false, "Update golden files")
)

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

func TestPorch(t *testing.T) {
	abs, err := filepath.Abs(filepath.Join(".", "testdata", "porch"))
	if err != nil {
		t.Fatalf("Failed to get absolute path to testdata directory: %v", err)
	}
	runTests(t, abs)
}

func runTests(t *testing.T, path string) {
	git := startGitServer(t, path)
	testCases := scanTestCases(t, path)

	for _, tc := range testCases {
		t.Run(tc.TestCase, func(t *testing.T) {
			runTestCase(t, git, tc)
		})
	}
}

func runTestCase(t *testing.T, git string, tc porch.TestCaseConfig) {
	porch.KubectlCreateNamespace(t, tc.TestCase)
	t.Cleanup(func() {
		porch.KubectlDeleteNamespace(t, tc.TestCase)
	})

	if tc.Repository != "" {
		porch.RegisterRepository(t, git, tc.TestCase, tc.Repository)
	}

	for i := range tc.Commands {
		command := &tc.Commands[i]
		cmd := exec.Command("kpt", command.Args...)

		var stdout, stderr bytes.Buffer
		if command.Stdin != "" {
			cmd.Stdin = strings.NewReader(command.Stdin)
		}
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()

		if command.Yaml {
			reorderYamlStdout(t, &stdout)
		}

		if *update {
			updateCommand(command, err, stdout.String(), stderr.String())
		}

		if got, want := exitCode(err), command.ExitCode; got != want {
			t.Errorf("unexpected exit code from 'kpt %s'; got %d, want %d", strings.Join(command.Args, " "), got, want)
		}
		if got, want := stdout.String(), command.Stdout; got != want {
			t.Errorf("unexpected stdout content from 'kpt %s'; (-want, +got) %s", strings.Join(command.Args, " "), cmp.Diff(want, got))
		}
		if got, want := stderr.String(), command.Stderr; got != want {
			t.Errorf("unexpected stderr content from 'kpt %s'; (-want, +got) %s", strings.Join(command.Args, " "), cmp.Diff(want, got))
		}
	}

	if *update {
		porch.WriteTestCaseConfig(t, &tc)
	}
}

func reorderYamlStdout(t *testing.T, buf *bytes.Buffer) {
	var data interface{}
	if err := yaml.Unmarshal(buf.Bytes(), &data); err != nil {
		// not yaml.
		return
	}

	var stable bytes.Buffer
	encoder := yaml.NewEncoder(&stable)
	encoder.SetIndent(2)
	if err := encoder.Encode(data); err != nil {
		t.Fatalf("Failed to re-encode yaml output: %v", err)
	}
	buf.Reset()
	if _, err := buf.Write(stable.Bytes()); err != nil {
		t.Fatalf("Failed to update reordered yaml output: %v", err)
	}
}

func startGitServer(t *testing.T, path string) string {
	gitServerImage := porch.GetGitServerImageName(t)
	t.Logf("Git Image: %s", gitServerImage)

	configFile := filepath.Join(path, "git-server.yaml")
	configBytes, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read git server config file %q: %v", configFile, err)
	}
	config := string(configBytes)
	config = strings.ReplaceAll(config, "GIT_SERVER_IMAGE", gitServerImage)

	t.Cleanup(func() {
		porch.KubectlDeleteNamespace(t, testGitNamespace)
	})

	porch.KubectlApply(t, config)
	porch.KubectlWaitForDeployment(t, testGitNamespace, "git-server")
	porch.KubectlWaitForService(t, testGitNamespace, "git-server")
	// TODO: kpt git address parsing is strange; the address constructed here is invalid
	// but required nonetheless to satisfy kpt parsing.
	porch.KubectlWaitForGitDNS(t, fmt.Sprintf("%s.git/@main", gitRepositoryAddress))

	return gitRepositoryAddress
}

func scanTestCases(t *testing.T, root string) []porch.TestCaseConfig {
	testCases := []porch.TestCaseConfig{}

	if err := filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		if path == root {
			return nil
		}

		tc := porch.ReadTestCaseConfig(t, info.Name(), path)
		testCases = append(testCases, tc)

		return nil
	}); err != nil {
		t.Fatalf("Failed to scan test cases: %v", err)
	}

	return testCases
}

func updateCommand(command *porch.Command, exit error, stdout, stderr string) {
	command.ExitCode = exitCode(exit)
	command.Stdout = stdout
	command.Stderr = stderr
}

func exitCode(exit error) int {
	var ee *exec.ExitError
	if errors.As(exit, &ee) {
		return ee.ExitCode()
	}
	return 0
}
