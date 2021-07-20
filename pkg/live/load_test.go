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
	"os"
	"sort"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/testutil/pkgbuilder"
	kptfile "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
	"k8s.io/kubectl/pkg/util/slice"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

func TestLoad_LocalDisk(t *testing.T) {
	testCases := map[string]struct {
		pkg            *pkgbuilder.RootPkg
		namespace      string
		expectedObjs   []object.ObjMetadata
		expectedInv    kptfile.Inventory
		expectedErrMsg string
	}{
		"no Kptfile in root package": {
			pkg: pkgbuilder.NewRootPkg().
				WithFile("deployment.yaml", deploymentA),
			namespace: "foo",
			expectedObjs: []object.ObjMetadata{
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment",
					Namespace: "foo",
				},
			},
		},
		"missing namespace for namespace scoped resources are defaulted": {
			pkg: pkgbuilder.NewRootPkg().
				WithFile("cm.yaml", configMap),
			namespace: "foo",
			expectedObjs: []object.ObjMetadata{
				{
					GroupKind: schema.GroupKind{
						Kind: "ConfigMap",
					},
					Name:      "cm",
					Namespace: "foo",
				},
			},
		},
		"function config is excluded": {
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithPipeline(
							pkgbuilder.NewFunction("gcr.io/kpt-dev/func:latest").
								WithConfigPath("cm.yaml"),
						),
				).
				WithFile("cm.yaml", configMap).
				WithSubPackages(
					pkgbuilder.NewSubPkg("subpkg").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithPipeline(
									pkgbuilder.NewFunction("gcr.io/kpt-dev/func").
										WithConfigPath("deployment.yaml"),
								),
						).
						WithFile("deployment.yaml", deploymentA),
				),
			namespace:    "foo",
			expectedObjs: []object.ObjMetadata{},
		},
		"inventory info is taken from the root Kptfile": {
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithInventory(pkgbuilder.Inventory{
							Name:      "foo",
							Namespace: "bar",
							ID:        "foo-bar",
						}),
				).
				WithFile("cm.yaml", configMap).
				WithSubPackages(
					pkgbuilder.NewSubPkg("subpkg").
						WithKptfile().
						WithFile("deployment.yaml", deploymentA),
				),
			namespace: "foo",
			expectedObjs: []object.ObjMetadata{
				{
					GroupKind: schema.GroupKind{
						Kind: "ConfigMap",
					},
					Name:      "cm",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment",
					Namespace: "foo",
				},
			},
			expectedInv: kptfile.Inventory{
				Name:        "foo",
				Namespace:   "bar",
				InventoryID: "foo-bar",
			},
		},
		"Inventory information in subpackages are ignored": {
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithInventory(pkgbuilder.Inventory{
							Name:      "foo",
							Namespace: "bar",
							ID:        "foo-bar",
						}),
				).
				WithSubPackages(
					pkgbuilder.NewSubPkg("subpkg").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithInventory(pkgbuilder.Inventory{
									Name:      "subpkg",
									Namespace: "subpkg",
									ID:        "subpkg",
								}),
						),
				),
			namespace:    "foo",
			expectedObjs: []object.ObjMetadata{},
			expectedInv: kptfile.Inventory{
				Name:        "foo",
				Namespace:   "bar",
				InventoryID: "foo-bar",
			},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			tf := cmdtesting.NewTestFactory().WithNamespace(tc.namespace)
			defer tf.Cleanup()

			dir := tc.pkg.ExpandPkg(t, nil)
			defer func() {
				_ = os.RemoveAll(dir)
			}()

			var buf bytes.Buffer
			objs, inv, err := Load(tf, dir, &buf)

			if tc.expectedErrMsg != "" {
				if !assert.Error(t, err) {
					t.FailNow()
				}
				assert.Contains(t, err.Error(), tc.expectedErrMsg)
				return
			}
			assert.NoError(t, err)

			objMetas := object.UnstructuredsToObjMetasOrDie(objs)
			sort.Slice(objMetas, func(i, j int) bool {
				return objMetas[i].String() < objMetas[j].String()
			})
			assert.Equal(t, tc.expectedObjs, objMetas)

			assert.Equal(t, tc.expectedInv, inv)
		})
	}
}

func TestLoad_StdIn(t *testing.T) {
	testCases := map[string]struct {
		pkg            *pkgbuilder.RootPkg
		namespace      string
		expectedObjs   []object.ObjMetadata
		expectedInv    kptfile.Inventory
		expectedErrMsg string
	}{
		"no Kptfile among the resources": {
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile(),
				).
				WithFile("deployment.yaml", deploymentA),
			namespace: "foo",
			expectedObjs: []object.ObjMetadata{
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment",
					Namespace: "foo",
				},
			},
		},
		"missing namespace for namespace scoped resources are defaulted": {
			pkg: pkgbuilder.NewRootPkg().
				WithFile("cm.yaml", configMap),
			namespace: "foo",
			expectedObjs: []object.ObjMetadata{
				{
					GroupKind: schema.GroupKind{
						Kind: "ConfigMap",
					},
					Name:      "cm",
					Namespace: "foo",
				},
			},
		},
		"inventory info is taken from the Kptfile": {
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithInventory(pkgbuilder.Inventory{
							Name:      "foo",
							Namespace: "bar",
							ID:        "foo-bar",
						}),
				).
				WithFile("cm.yaml", configMap).
				WithSubPackages(
					pkgbuilder.NewSubPkg("subpkg").
						WithKptfile().
						WithFile("deployment.yaml", deploymentA),
				),
			namespace: "foo",
			expectedObjs: []object.ObjMetadata{
				{
					GroupKind: schema.GroupKind{
						Kind: "ConfigMap",
					},
					Name:      "cm",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment",
					Namespace: "foo",
				},
			},
			expectedInv: kptfile.Inventory{
				Name:        "foo",
				Namespace:   "bar",
				InventoryID: "foo-bar",
			},
		},
		"Multiple Kptfiles with inventory is an error": {
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithInventory(pkgbuilder.Inventory{
							Name:      "foo",
							Namespace: "bar",
							ID:        "foo-bar",
						}),
				).
				WithSubPackages(
					pkgbuilder.NewSubPkg("subpkg").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithInventory(pkgbuilder.Inventory{
									Name:      "subpkg",
									Namespace: "subpkg",
									ID:        "subpkg",
								}),
						),
				),
			namespace:      "foo",
			expectedErrMsg: "multiple Kptfiles contains inventory information",
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			tf := cmdtesting.NewTestFactory().WithNamespace(tc.namespace)
			defer tf.Cleanup()

			dir := tc.pkg.ExpandPkg(t, nil)
			defer func() {
				_ = os.RemoveAll(dir)
			}()

			var buf bytes.Buffer
			err := (&kio.Pipeline{
				Inputs: []kio.Reader{
					kio.LocalPackageReader{
						PackagePath:           dir,
						OmitReaderAnnotations: true,
						MatchFilesGlob:        append([]string{kptfile.KptFileName}, kio.DefaultMatch...),
						IncludeSubpackages:    true,
					},
				},
				Outputs: []kio.Writer{
					kio.ByteWriter{
						Writer: &buf,
					},
				},
			}).Execute()
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			objs, inv, err := Load(tf, "-", &buf)

			if tc.expectedErrMsg != "" {
				if !assert.Error(t, err) {
					t.FailNow()
				}
				assert.Contains(t, err.Error(), tc.expectedErrMsg)
				return
			}
			assert.NoError(t, err)

			objMetas := object.UnstructuredsToObjMetasOrDie(objs)
			sort.Slice(objMetas, func(i, j int) bool {
				return objMetas[i].String() < objMetas[j].String()
			})
			assert.Equal(t, tc.expectedObjs, objMetas)

			assert.Equal(t, tc.expectedInv, inv)
		})
	}
}

func TestValidateInventory(t *testing.T) {
	testCases := map[string]struct {
		inventory           kptfile.Inventory
		expectErr           bool
		expectedErrorFields []string
	}{
		"complete inventory info validate": {
			inventory: kptfile.Inventory{
				Name:        "foo",
				Namespace:   "default",
				InventoryID: "foo-default",
			},
			expectErr: false,
		},
		"inventory info without name doesn't validate": {
			inventory: kptfile.Inventory{
				Namespace:   "default",
				InventoryID: "foo-default",
			},
			expectErr:           true,
			expectedErrorFields: []string{"name"},
		},
		"inventory without id or namespace doesn't validate": {
			inventory: kptfile.Inventory{
				Name: "foo",
			},
			expectErr:           true,
			expectedErrorFields: []string{"namespace", "inventoryID"},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			err := validateInventory(tc.inventory)

			if !tc.expectErr {
				assert.NoError(t, err)
				return
			}

			if !assert.Error(t, err) {
				t.FailNow()
			}

			invInfoValError, ok := err.(*InventoryInfoValidationError)
			if !ok {
				t.Errorf("expected error of type *InventoryInfoValidationError")
				t.FailNow()
			}
			assert.Equal(t, len(tc.expectedErrorFields), len(invInfoValError.Violations))
			fields := invInfoValError.Violations.Fields()
			for i := range tc.expectedErrorFields {
				if !slice.ContainsString(fields, tc.expectedErrorFields[i], nil) {
					t.Errorf("expected error for field %s, but didn't find it", tc.expectedErrorFields[i])
				}
			}
		})
	}
}
