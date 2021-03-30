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
	"os/exec"
)

func runCommand(pwd, name string, arg []string) (string, error) {
	cmd := exec.Command(name, arg...)
	cmd.Dir = pwd
	o, err := cmd.CombinedOutput()
	return string(o), err
}

func copyDir(src, dst string) error {
	_, err := runCommand("", "cp", []string{"-r", src, dst})
	return err
}

func gitInit(d string) error {
	o, err := runCommand(d, "git", []string{"init"})
	if err != nil {
		return fmt.Errorf("git init error: %w, output: %s", err, o)
	}
	return nil
}

func gitAddAll(d string) error {
	o, err := runCommand(d, "git", []string{"add", "--all"})
	if err != nil {
		return fmt.Errorf("git commit error: %w, output: %s", err, o)
	}
	return nil
}

func gitCommit(d, msg string) error {
	o, err := runCommand(d, "git", []string{"config", "user.name", "none"})
	if err != nil {
		return fmt.Errorf("git config error: %w, output: %s", err, o)
	}
	o, err = runCommand(d, "git", []string{"config", "user.email", "none"})
	if err != nil {
		return fmt.Errorf("git config error: %w, output: %s", err, o)
	}
	o, err = runCommand(d, "git", []string{"commit", "-m", msg, "--allow-empty"})
	if err != nil {
		return fmt.Errorf("git commit error: %w, output: %s", err, o)
	}
	return nil
}

func gitDiff(d, commit1, commit2 string) (string, error) {
	o, err := runCommand(d, "git", []string{"diff", commit1, commit2})
	if err != nil {
		return "", fmt.Errorf("git diff error: %w, output: %s", err, o)
	}
	return o, nil
}
