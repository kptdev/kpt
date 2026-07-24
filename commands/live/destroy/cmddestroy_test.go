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

package destroy

import (
	"path/filepath"
	"testing"
	"time"

	kptfilev1 "github.com/kptdev/kpt/api/kptfile/v1"
	"github.com/kptdev/kpt/internal/testutil"
	"github.com/kptdev/kpt/pkg/kptfile/kptfileutil"
	"github.com/kptdev/kpt/pkg/printer/fake"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/inventory"
)

func TestCmd(t *testing.T) {
	validInventory := &kptfilev1.Inventory{
		Namespace:   "my-ns",
		Name:        "my-name",
		InventoryID: "my-inv-id",
	}

	// failOnDestroy is used for error cases where preRunE should reject the
	// input before the destroy runner is reached.
	failOnDestroy := func(t *testing.T, _ *Runner, _ inventory.Info) {
		t.Fatal("destroy runner should not be called for error cases")
	}

	testCases := map[string]struct {
		args                []string
		namespace           string
		inventory           *kptfilev1.Inventory
		destroyCallbackFunc func(*testing.T, *Runner, inventory.Info)
		expectedErrorMsg    string
	}{
		"destroy rejects invalid inventory policy": {
			args:                []string{"--inventory-policy", "noSuchPolicy"},
			namespace:           "testns",
			destroyCallbackFunc: failOnDestroy,
			expectedErrorMsg:    "inventory policy must be one of strict, adopt",
		},
		"destroy rejects invalid status policy": {
			args:                []string{"--status-policy", "noSuchPolicy"},
			namespace:           "testns",
			destroyCallbackFunc: failOnDestroy,
			expectedErrorMsg:    "status policy must be one of none, all",
		},
		"destroy rejects invalid output format": {
			args:                []string{"--output", "foo"},
			namespace:           "testns",
			destroyCallbackFunc: failOnDestroy,
			expectedErrorMsg:    "unknown output type \"foo\"",
		},
		"destroy rejects invalid delete-propagation-policy": {
			args:                []string{"--delete-propagation-policy", "noSuchPolicy"},
			namespace:           "testns",
			destroyCallbackFunc: failOnDestroy,
			expectedErrorMsg:    "prune propagation policy must be one of Background, Foreground, Orphan",
		},
		"destroy accepts Foreground delete-propagation-policy": {
			args: []string{
				"--delete-propagation-policy", "Foreground",
			},
			inventory: validInventory,
			namespace: "testns",
			destroyCallbackFunc: func(t *testing.T, r *Runner, _ inventory.Info) {
				assert.Equal(t, metav1.DeletePropagationForeground, r.deletePropPolicy)
			},
		},
		"destroy parses delete-timeout duration": {
			args: []string{
				"--delete-timeout", "2m",
			},
			inventory: validInventory,
			namespace: "testns",
			destroyCallbackFunc: func(t *testing.T, r *Runner, _ inventory.Info) {
				assert.Equal(t, 2*time.Minute, r.deleteTimeout)
			},
		},
		"destroy reads inventory from Kptfile": {
			args: []string{
				"--inventory-policy", "adopt",
				"--output", "events",
			},
			inventory: validInventory,
			namespace: "testns",
			destroyCallbackFunc: func(t *testing.T, _ *Runner, inv inventory.Info) {
				assert.Equal(t, "my-ns", inv.Namespace())
				assert.Equal(t, "my-name", inv.Name())
				assert.Equal(t, "my-inv-id", inv.ID())
			},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			tf := cmdtesting.NewTestFactory().WithNamespace(tc.namespace)
			defer tf.Cleanup()
			ioStreams, _, _, _ := genericclioptions.NewTestIOStreams() //nolint:dogsled

			w, clean := testutil.SetupWorkspace(t)
			defer clean()
			kf := kptfileutil.DefaultKptfile(filepath.Base(w.WorkspaceDirectory))
			kf.Inventory = tc.inventory
			testutil.AddKptfileToWorkspace(t, w, kf)

			revert := testutil.Chdir(t, w.WorkspaceDirectory)
			defer revert()

			runner := NewRunner(fake.CtxWithDefaultPrinter(), tf, ioStreams)
			runner.Command.SetArgs(tc.args)
			// Stub out the destroy execution to isolate flag parsing and
			// validation from actual cluster interaction.
			runner.destroyRunner = func(r *Runner, inv inventory.Info, _ common.DryRunStrategy) error {
				tc.destroyCallbackFunc(t, r, inv)
				return nil
			}
			err := runner.Command.Execute()

			if tc.expectedErrorMsg != "" {
				if !assert.Error(t, err) {
					t.FailNow()
				}
				assert.Contains(t, err.Error(), tc.expectedErrorMsg)
				return
			}
			assert.NoError(t, err)
		})
	}
}
