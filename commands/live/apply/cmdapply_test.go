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

package apply

import (
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/testutil"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"github.com/GoogleContainerTools/kpt/pkg/printer/fake"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/inventory"
)

func TestCmd(t *testing.T) {
	testCases := map[string]struct {
		args              []string
		namespace         string
		inventory         *kptfilev1.Inventory
		applyCallbackFunc func(*testing.T, *Runner, inventory.Info)
		expectedErrorMsg  string
	}{
		"invalid inventory policy": {
			args: []string{
				"--inventory-policy", "noSuchPolicy",
			},
			namespace: "testns",
			applyCallbackFunc: func(t *testing.T, _ *Runner, _ inventory.Info) {
				t.FailNow()
			},
			expectedErrorMsg: "inventory policy must be one of strict, adopt",
		},
		"invalid output format": {
			args: []string{
				"--output", "foo",
			},
			namespace: "testns",
			applyCallbackFunc: func(t *testing.T, _ *Runner, _ inventory.Info) {
				t.FailNow()
			},
			expectedErrorMsg: "unknown output type \"foo\"",
		},
		"fetches the correct inventory information from the Kptfile": {
			args: []string{
				"--inventory-policy", "adopt",
				"--output", "events",
			},
			inventory: &kptfilev1.Inventory{
				Namespace:   "my-ns",
				Name:        "my-name",
				InventoryID: "my-inv-id",
			},
			namespace: "testns",
			applyCallbackFunc: func(t *testing.T, _ *Runner, inv inventory.Info) {
				assert.Equal(t, "my-ns", inv.Namespace())
				assert.Equal(t, "my-name", inv.Name())
				assert.Equal(t, "my-inv-id", inv.ID())
			},
		},
		"install-resource-group flag defaults to true": {
			args: []string{},
			inventory: &kptfilev1.Inventory{
				Namespace:   "my-ns",
				Name:        "my-name",
				InventoryID: "my-inv-id",
			},
			namespace: "testns",
			applyCallbackFunc: func(t *testing.T, r *Runner, _ inventory.Info) {
				assert.True(t, r.installCRD)
			},
		},
		"install-resource-group flag defaults to false when dry-run": {
			args: []string{
				"--dry-run",
			},
			inventory: &kptfilev1.Inventory{
				Namespace:   "my-ns",
				Name:        "my-name",
				InventoryID: "my-inv-id",
			},
			namespace: "testns",
			applyCallbackFunc: func(t *testing.T, r *Runner, _ inventory.Info) {
				assert.True(t, r.dryRun)
				assert.False(t, r.installCRD)
			},
			expectedErrorMsg: "type ResourceGroup not found",
		},
		"install-resource-group flag remains true with dry-run if explicitly set": {
			args: []string{
				"--dry-run",
				"--install-resource-group",
			},
			inventory: &kptfilev1.Inventory{
				Namespace:   "my-ns",
				Name:        "my-name",
				InventoryID: "my-inv-id",
			},
			namespace: "testns",
			applyCallbackFunc: func(t *testing.T, r *Runner, _ inventory.Info) {
				assert.True(t, r.dryRun)
				assert.True(t, r.installCRD)
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

			runner := NewRunner(fake.CtxWithDefaultPrinter(), tf, ioStreams, false)
			runner.Command.SetArgs(tc.args)
			runner.applyRunner = func(_ *Runner, inv inventory.Info,
				_ []*unstructured.Unstructured, _ common.DryRunStrategy) error {
				tc.applyCallbackFunc(t, runner, inv)
				return nil
			}
			err := runner.Command.Execute()

			// Check if there should be an error
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
