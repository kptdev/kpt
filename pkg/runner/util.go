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
