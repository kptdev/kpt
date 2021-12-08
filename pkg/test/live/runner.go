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

package live

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// Runner uses the provided Config to run a test.
type Runner struct {
	// Config provides the configuration for how this test should be
	// executed.
	Config TestCaseConfig

	// Path provides the path to the test files.
	Path string
}

// Run executes the test.
func (r *Runner) Run(t *testing.T) {
	testName := filepath.Base(r.Path)
	r.RunPreApply(t)

	stdout, stderr, err := r.RunApply(t)
	r.VerifyExitCode(t, err)
	r.VerifyStdout(t, stdout)
	r.VerifyStderr(t, stderr)
	if len(r.Config.Inventory) != 0 {
		r.VerifyInventory(t, testName, testName)
	}
}

func (r *Runner) RunPreApply(t *testing.T) {
	preApplyDir := filepath.Join(r.Path, "pre-apply")
	fi, err := os.Stat(preApplyDir)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("error checking for pre-apply dir: %v", err)
	}
	if os.IsNotExist(err) || !fi.IsDir() {
		return
	}
	t.Log("Applying resources in pre-apply directory")
	cmd := exec.Command("kubectl", "apply", "-f", preApplyDir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("error applying pre-apply dir: %v", err)
	}
}

func (r *Runner) RunApply(t *testing.T) (string, string, error) {
	args := append([]string{"live", "apply"}, r.Config.KptArgs...)
	t.Logf("Running command: kpt %s", strings.Join(args, " "))
	cmd := exec.Command("kpt", args...)
	cmd.Dir = filepath.Join(r.Path, "resources")

	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

func (r *Runner) VerifyExitCode(t *testing.T, err error) {
	exitCode := 0
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		exitCode = exitErr.ExitCode()
	}
	if want, got := r.Config.ExitCode, exitCode; want != got {
		t.Errorf("expected exit code %d, but got %d", want, got)
	}
}

func (r *Runner) VerifyStdout(t *testing.T, stdout string) {
	assert.Equal(t, strings.TrimSpace(r.Config.StdOut), strings.TrimSpace(substituteTimestamps(stdout)))
}

func (r *Runner) VerifyStderr(t *testing.T, stderr string) {
	assert.Equal(t, strings.TrimSpace(r.Config.StdErr), strings.TrimSpace(substituteTimestamps(stderr)))
}

func (r *Runner) VerifyInventory(t *testing.T, name, namespace string) {
	rgExec := exec.Command("kubectl", "get", "resourcegroups.kpt.dev",
		"-n", namespace, name, "-oyaml")
	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	rgExec.Stdout = &outBuf
	rgExec.Stderr = &errBuf
	err := rgExec.Run()
	if strings.Contains(errBuf.String(), "NotFound") {
		t.Errorf("inventory with namespace %s and name %s not found",
			namespace, name)
		return
	}
	if err != nil {
		t.Fatalf("error looking up resource group: %v", err)
	}
	var rg map[string]interface{}
	err = yaml.Unmarshal(outBuf.Bytes(), &rg)
	if err != nil {
		t.Fatalf("error unmarshalling inventory object: %v", err)
	}

	var inventory []InventoryEntry
	if rg["spec"] != nil {
		spec := rg["spec"].(map[string]interface{})
		if spec["resources"] != nil {
			resources := spec["resources"].([]interface{})
			for i := range resources {
				r := resources[i].(map[string]interface{})
				inventory = append(inventory, InventoryEntry{
					Group:     r["group"].(string),
					Kind:      r["kind"].(string),
					Name:      r["name"].(string),
					Namespace: r["namespace"].(string),
				})
			}
		}
	}

	expectedInventory := r.Config.Inventory
	sort.Slice(expectedInventory, inventorySortFunc(expectedInventory))
	sort.Slice(inventory, inventorySortFunc(inventory))

	assert.Equal(t, expectedInventory, inventory)

}

func inventorySortFunc(inv []InventoryEntry) func(i, j int) bool {
	return func(i, j int) bool {
		iInv := inv[i]
		jInv := inv[j]

		if iInv.Group != jInv.Group {
			return iInv.Group < jInv.Group
		}
		if iInv.Kind != jInv.Kind {
			return iInv.Kind < jInv.Kind
		}
		if iInv.Name != jInv.Name {
			return iInv.Name < jInv.Name
		}
		return iInv.Namespace < jInv.Namespace
	}
}

var timestampRegexp = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z`)

func substituteTimestamps(text string) string {
	return timestampRegexp.ReplaceAllString(text, "<TIMESTAMP>")
}
