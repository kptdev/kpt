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

package live

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/testutil"
	"github.com/GoogleContainerTools/kpt/internal/testutil/pkgbuilder"
	kptfile "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	rgfilev1alpha1 "github.com/GoogleContainerTools/kpt/pkg/api/resourcegroup/v1alpha1"
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
		expectedObjs   object.ObjMetadataSet
		expectedInv    kptfile.Inventory
		expectedErrMsg string
		rgFile         string
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
			expectedErrMsg: "no ResourceGroup object was provided within the stream or package",
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
			expectedErrMsg: "no ResourceGroup object was provided within the stream or package",
		},
		"function config is excluded": {
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithPipeline(
							pkgbuilder.NewFunction("gcr.io/kpt-dev/func:latest").
								WithConfigPath("cm.yaml"),
						),
				).WithRGFile(pkgbuilder.NewRGFile().WithInventory(pkgbuilder.Inventory{
				Name:      "foo",
				Namespace: "bar",
				ID:        "foo-bar"},
			)).
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
			namespace: "foo",
			expectedInv: kptfile.Inventory{
				Name:        "foo",
				Namespace:   "bar",
				InventoryID: "foo-bar",
			},
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
		"Inventory information taken from resourcegroup": {
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile(),
				).WithRGFile(pkgbuilder.NewRGFile().WithInventory(pkgbuilder.Inventory{
				Name:      "foo",
				Namespace: "bar",
				ID:        "foo-bar"},
			)),
			namespace:    "foo",
			expectedObjs: []object.ObjMetadata{},
			expectedInv: kptfile.Inventory{
				Name:        "foo",
				Namespace:   "bar",
				InventoryID: "foo-bar",
			},
			rgFile: "resourcegroup.yaml",
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

			objMetas := object.UnstructuredSetToObjMetadataSet(objs)
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
		expectedObjs   object.ObjMetadataSet
		expectedInv    kptfile.Inventory
		expectedErrMsg string
		rgFile         string
	}{
		"no inventory among the resources": {
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile(),
				).
				WithFile("deployment.yaml", deploymentA),
			expectedErrMsg: "no ResourceGroup object was provided within the stream or package",
		},
		"missing namespace for namespace scoped resources are defaulted": {
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithInventory(pkgbuilder.Inventory{
							Name:      "foo",
							Namespace: "bar",
							ID:        "foo-bar",
						}),
				).
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
			expectedInv: kptfile.Inventory{
				Name:        "foo",
				Namespace:   "bar",
				InventoryID: "foo-bar",
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
			expectedErrMsg: "multiple Kptfile inventories found in package",
		},
		"Inventory using STDIN resourcegroup file": {
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile(),
				).
				WithFile("cm.yaml", configMap).
				WithRGFile(pkgbuilder.NewRGFile().WithInventory(pkgbuilder.Inventory{
					Name:      "foo",
					Namespace: "bar",
					ID:        "foo-bar",
				},
				)),
			namespace: "foo",
			expectedInv: kptfile.Inventory{
				Name:        "foo",
				Namespace:   "bar",
				InventoryID: "foo-bar",
			},
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
		"Multiple inventories using STDIN resourcegroup and Kptfile is error": {
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
				WithRGFile(pkgbuilder.NewRGFile().WithInventory(pkgbuilder.Inventory{
					Name:      "foo",
					Namespace: "bar",
					ID:        "foo-bar",
				},
				)),
			expectedErrMsg: "inventory was found in both Kptfile and ResourceGroup object",
		},
		"Non-valid inventory using STDIN Kptfile is error": {
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithInventory(pkgbuilder.Inventory{
							Name: "foo",
						}),
				).
				WithFile("cm.yaml", configMap),
			expectedErrMsg: "the provided ResourceGroup is not valid",
		},
		"Non-valid inventory in resourcegroup is error": {
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile(),
				).
				WithFile("cm.yaml", configMap).
				WithRGFile(pkgbuilder.NewRGFile().WithInventory(pkgbuilder.Inventory{
					Name: "foo",
				},
				)),
			expectedErrMsg: "the provided ResourceGroup is not valid",
			rgFile:         rgfilev1alpha1.RGFileName,
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

			revert := testutil.Chdir(t, dir)
			defer revert()

			var buf bytes.Buffer
			err := (&kio.Pipeline{
				Inputs: []kio.Reader{
					kio.LocalPackageReader{
						PackagePath:           dir,
						OmitReaderAnnotations: true,
						MatchFilesGlob:        append([]string{kptfile.KptFileName}, kio.DefaultMatch...),
						IncludeSubpackages:    true,
						WrapBareSeqNode:       true,
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

			os.Remove(filepath.Join(dir, kptfile.KptFileName))
			os.Remove(filepath.Join(dir, tc.rgFile))

			objs, inv, err := Load(tf, "-", &buf)

			if tc.expectedErrMsg != "" {
				if !assert.Error(t, err) {
					t.FailNow()
				}
				assert.Contains(t, err.Error(), tc.expectedErrMsg)
				return
			}
			assert.NoError(t, err)

			objMetas := object.UnstructuredSetToObjMetadataSet(objs)
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
		"inventory namespace doesn't validate": {
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
