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
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	KindClusterName = "live-e2e-test"
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
	isAvailable := r.CheckKindClusterAvailable(t)
	if r.Config.RequiresCleanCluster {
		t.Log("Test requires clean cluster")
		if isAvailable {
			t.Log("Removing existing cluster")
			r.RemoveKindCluster(t)
		}
		t.Log("Creating new cluster")
		r.CreateKindCluster(t)
	} else {
		if !isAvailable {
			t.Log("Creating new cluster")
			r.CreateKindCluster(t)
		} else if r.CheckForNamespace(t, testName) {
			t.Log("Namespace already exist, creating new cluster")
			r.RemoveKindCluster(t)
			r.CreateKindCluster(t)
		}
	}

	if r.Config.PreinstallResourceGroup {
		r.InstallResourceGroup(t)
	}

	r.CreateNamespace(t, testName)
	defer r.RemoveNamespace(t, testName)

	stdout, stderr, err := r.RunApply()
	r.VerifyExitCode(t, err)
	r.VerifyStdout(t, stdout)
	r.VerifyStderr(t, stderr)
	if len(r.Config.Inventory) != 0 {
		r.VerifyInventory(t, testName, testName)
	}
}

func (r *Runner) RunApply() (string, string, error) {
	args := append([]string{"live", "apply"}, r.Config.KptArgs...)
	cmd := exec.Command("kpt", args...)
	cmd.Dir = filepath.Join(r.Path, "resources")

	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

func (r *Runner) InstallResourceGroup(t *testing.T) {
	cmd := exec.Command("kpt", "live", "install-resource-group")
	if err := cmd.Run(); err != nil {
		t.Fatalf("error installing ResourceGroup CRD: %v", err)
	}
}

func (r *Runner) CheckIfResourceGroupInstalled(t *testing.T) bool {
	cmd := exec.Command("kubectl", "get", "resourcegroups.kpt.dev")
	output, err := cmd.CombinedOutput()
	if strings.Contains(string(output), "the server doesn't have a resource type") {
		return false
	}
	if err != nil {
		t.Fatalf("error checking for ResourceGroup CRD: %v", err)
	}
	return true
}

func (r *Runner) CheckKindClusterAvailable(t *testing.T) bool {
	cmd := exec.Command("kind", "get", "clusters")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to check for kind cluster: %v", err)
	}

	sc := bufio.NewScanner(bytes.NewReader(output))
	for sc.Scan() {
		if strings.TrimSpace(sc.Text()) == KindClusterName {
			return true
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("error parsing output from 'kind get cluster': %v", err)
	}
	return false
}

func (r *Runner) CreateKindCluster(t *testing.T) {
	cmd := exec.Command("kind", "create", "cluster", fmt.Sprintf("--name=%s", KindClusterName))
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create new kind cluster: %v", err)
	}
}

func (r *Runner) RemoveKindCluster(t *testing.T) {
	cmd := exec.Command("kind", "delete", "cluster", fmt.Sprintf("--name=%s", KindClusterName))
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to remove existing cluster: %v", err)
	}
}

func (r *Runner) CheckForNamespace(t *testing.T, namespace string) bool {
	cmd := exec.Command("kubectl", "get", "ns", namespace, "--no-headers", "--output=name")
	output, err := cmd.CombinedOutput()
	if strings.Contains(string(output), "NotFound") {
		return false
	}
	if err != nil {
		t.Fatalf("error listing namespaces with kubectl: %v", err)
	}
	sc := bufio.NewScanner(bytes.NewReader(output))
	for sc.Scan() {
		if strings.TrimSpace(sc.Text()) == fmt.Sprintf("namespace/%s", namespace) {
			return true
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("error parsing output from 'kubectl get ns': %v", err)
	}
	return false
}

func (r *Runner) CreateNamespace(t *testing.T, namespace string) {
	cmd := exec.Command("kubectl", "create", "ns", namespace)
	if err := cmd.Run(); err != nil {
		t.Fatalf("error creating namespace %s: %v", namespace, err)
	}
}

func (r *Runner) RemoveNamespace(t *testing.T, namespace string) {
	cmd := exec.Command("kubectl", "delete", "ns", namespace, "--wait=false")
	if err := cmd.Run(); err != nil {
		t.Logf("error deleting namespace %s: %v", namespace, err)
	}
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
	assert.Equal(t, strings.TrimSpace(r.Config.StdOut), strings.TrimSpace(stdout))
}

func (r *Runner) VerifyStderr(t *testing.T, stderr string) {
	assert.Equal(t, strings.TrimSpace(r.Config.StdErr), strings.TrimSpace(stderr))
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
