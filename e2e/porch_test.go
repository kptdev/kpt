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

//go:build porch

package e2e

import (
	"bufio"
	"bytes"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GoogleContainerTools/kpt/pkg/test/porch"
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	updateGoldenFiles = "UPDATE_GOLDEN_FILES"
	testGitNamespace  = "test-git-namespace"
)

func TestPorch(t *testing.T) {
	abs, err := filepath.Abs(filepath.Join(".", "testdata", "porch"))
	if err != nil {
		t.Fatalf("Failed to get absolute path to testdata directory: %v", err)
	}
	runTests(t, abs)
}

func runUtilityCommand(t *testing.T, command string, args ...string) error {
	cmd := exec.Command(command, args...)
	t.Logf("running utility command %s %s", command, strings.Join(args, " "))
	return cmd.Run()
}

func runTests(t *testing.T, path string) {
	gitServerURL := startGitServer(t, path)
	testCases := scanTestCases(t, path)

	// remove any tmp files from previous test runs
	err := runUtilityCommand(t, "rm", "-rf", "/tmp/porch-e2e")
	if err != nil {
		t.Fatalf("Failed to clean up older run: %v", err)
	}
	err = runUtilityCommand(t, "mkdir", "/tmp/porch-e2e")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	for _, tc := range testCases {
		t.Run(tc.TestCase, func(t *testing.T) {
			if tc.Skip != "" {
				t.Skipf("Skipping test: %s", tc.Skip)
			}
			repoURL := gitServerURL + "/" + strings.ReplaceAll(tc.TestCase, "/", "-")
			runTestCase(t, repoURL, tc)
		})
	}
}

func runTestCase(t *testing.T, repoURL string, tc porch.TestCaseConfig) {
	porch.KubectlCreateNamespace(t, tc.TestCase)
	t.Cleanup(func() {
		porch.KubectlDeleteNamespace(t, tc.TestCase)
	})

	if tc.Repository != "" {
		porch.RegisterRepository(t, repoURL, tc.TestCase, tc.Repository)
	}

	for i := range tc.Commands {
		time.Sleep(1 * time.Second)
		command := &tc.Commands[i]
		cmd := exec.Command("kpt", command.Args...)

		var stdout, stderr bytes.Buffer
		if command.Stdin != "" {
			cmd.Stdin = strings.NewReader(command.Stdin)
		}
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		t.Logf("running command %v", strings.Join(cmd.Args, " "))
		err := cmd.Run()

		if command.Yaml {
			reorderYamlStdout(t, &stdout)
		}

		cleanupStderr(t, &stderr)

		if os.Getenv(updateGoldenFiles) != "" {
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

		// hack here; but if the command registered a repo, give a few extra seconds for the repo to reach readiness
		for _, arg := range command.Args {
			if arg == "register" {
				time.Sleep(5 * time.Second)
			}
		}
	}

	if os.Getenv(updateGoldenFiles) != "" {
		porch.WriteTestCaseConfig(t, &tc)
	}
}

// remove PASS lines from kpt fn eval, which includes a duration and will vary
func cleanupStderr(t *testing.T, buf *bytes.Buffer) {
	scanner := bufio.NewScanner(buf)
	var newBuf bytes.Buffer
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "[PASS]") {
			newBuf.Write([]byte(line))
			newBuf.Write([]byte("\n"))
		}
	}

	buf.Reset()
	if _, err := buf.Write(newBuf.Bytes()); err != nil {
		t.Fatalf("Failed to update cleaned up stderr: %v", err)
	}
}

func reorderYamlStdout(t *testing.T, buf *bytes.Buffer) {
	if buf.Len() == 0 {
		return
	}

	// strip out the resourceVersion:, creationTimestamp:
	// because that will change with every run
	scanner := bufio.NewScanner(buf)
	var newBuf bytes.Buffer
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "resourceVersion:") &&
			!strings.Contains(line, "creationTimestamp:") {
			newBuf.Write([]byte(line))
			newBuf.Write([]byte("\n"))
		}
	}

	var data interface{}
	if err := yaml.Unmarshal(newBuf.Bytes(), &data); err != nil {
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
	gitServerURL := "http://git-server." + testGitNamespace + ".svc.cluster.local:8080"

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
	porch.KubectlWaitForGitDNS(t, gitServerURL)

	return gitServerURL
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
