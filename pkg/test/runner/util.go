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
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func runCommand(cmd *exec.Cmd) (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func getCommand(pwd, name string, arg []string) *exec.Cmd {
	cmd := exec.Command(name, arg...)
	cmd.Dir = pwd
	return cmd
}

func copyDir(src, dst string) error {
	_, _, err := runCommand(getCommand("", "cp", []string{"-r", src, dst}))
	return err
}

func gitInit(d string) error {
	stdout, stderr, err := runCommand(getCommand(d, "git", []string{"init"}))
	if err != nil {
		return fmt.Errorf("git init error: %w, output: %s, stderr: %s", err, stdout, stderr)
	}
	return nil
}

func gitAddAll(d string) error {
	stdout, stderr, err := runCommand(getCommand(d, "git", []string{"add", "--all"}))
	if err != nil {
		return fmt.Errorf("git commit error: %w, output: %s, stderr: %s", err, stdout, stderr)
	}
	return nil
}

func gitCommit(d, msg string) error {
	stdout, stderr, err := runCommand(getCommand(d, "git", []string{"config", "user.name", "none"}))
	if err != nil {
		return fmt.Errorf("git config error: %w, output: %s, stderr: %s", err, stdout, stderr)
	}
	stdout, stderr, err = runCommand(getCommand(d, "git", []string{"config", "user.email", "none"}))
	if err != nil {
		return fmt.Errorf("git config error: %w, output: %s, stderr: %s", err, stdout, stderr)
	}
	stdout, stderr, err = runCommand(getCommand(d, "git", []string{"commit", "-m", msg, "--allow-empty"}))
	if err != nil {
		return fmt.Errorf("git commit error: %w, output: %s, stderr: %s", err, stdout, stderr)
	}
	return nil
}

func gitDiff(d, commit1, commit2 string) (string, error) {
	stdout, stderr, err := runCommand(getCommand(d, "git", []string{"diff", commit1, commit2}))
	if err != nil {
		return "", fmt.Errorf("git diff error: %w, output: %s, stderr: %s", err, stdout, stderr)
	}
	return stdout, nil
}

func getCommitHash(d string) (string, error) {
	stdout, stderr, err := runCommand(getCommand(d, "git", []string{"log", "-n", "1", "--pretty=format:%h"}))
	if err != nil {
		return "", fmt.Errorf("git log error: %w, output: %s, stderr: %s", err, stdout, stderr)
	}
	return stdout, nil
}

func diffStrings(actual, expected string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "kpt-e2e-diff-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)
	actualPath := filepath.Join(tmpDir, "actual")
	expectedPath := filepath.Join(tmpDir, "expected")
	if err := os.WriteFile(actualPath, []byte(actual), 0644); err != nil {
		return "", fmt.Errorf("failed to write file %s", actualPath)
	}
	if err := os.WriteFile(expectedPath, []byte(expected), 0644); err != nil {
		return "", fmt.Errorf("failed to write file %s", expectedPath)
	}
	// diff is expected to exit with 1 so we ignore the error here
	output, _, _ := runCommand(getCommand(tmpDir, "diff", []string{"-u", expectedPath, actualPath}))
	return output, nil
}
