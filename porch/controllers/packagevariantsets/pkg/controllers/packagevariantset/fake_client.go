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

package packagevariantset

import (
	"context"
	"fmt"

	pkgvarapi "github.com/GoogleContainerTools/kpt/porch/controllers/packagevariants/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type fakeClient struct {
	objects []client.Object
	created []client.Object
	updated []client.Object
	deleted []client.Object
	client.Client
}

var _ client.Client = &fakeClient{}

func (f *fakeClient) Create(_ context.Context, obj client.Object, _ ...client.CreateOption) error {
	fmt.Println("Creating", obj.GetName())
	f.created = append(f.created, obj)
	return nil
}

func (f *fakeClient) Delete(_ context.Context, obj client.Object, _ ...client.DeleteOption) error {
	fmt.Println("Deleting", obj.GetName())
	f.deleted = append(f.deleted, obj)
	return nil
}

func (f *fakeClient) Update(_ context.Context, obj client.Object, _ ...client.UpdateOption) error {
	fmt.Println("Updating", obj.GetName())
	f.updated = append(f.updated, obj)
	return nil
}

func (f *fakeClient) List(_ context.Context, obj client.ObjectList, _ ...client.ListOption) error {
	podList := `apiVersion: v1
kind: PodList
metadata:
  name: my-pod-list
items:
- apiVersion: v1
  kind: Pod
  metadata:
    name: my-pod-1
    labels:
      foo: bar
      abc: def
- apiVersion: v1
  kind: Pod
  metadata:
    name: my-pod-2
    labels:
      abc: def
      efg: hij`

	pvList := `apiVersion: config.porch.kpt.dev
kind: PackageVariantList
metadata:
  name: my-pv-list
items:
- apiVersion: config.porch.kpt.dev
  kind: PackageVariant
  metadata:
    name: my-pvs-dnrepo1-dnpkg1
  spec:
    upstream:
      repo: up
      package: up
      revision: up
    downstream:
      repo: dnrepo1
      package: dnpkg1
- apiVersion: config.porch.kpt.dev
  kind: PackageVariant
  metadata:
    name: my-pvs-dnrepo2-dnpkg2
  spec:
    upstream:
      repo: up
      package: up
      revision: up
    downstream:
      repo: dnrepo2
      package: dnpkg2`

	var err error
	switch v := obj.(type) {
	case *unstructured.UnstructuredList:
		err = yaml.Unmarshal([]byte(podList), v)
		for _, o := range v.Items {
			f.objects = append(f.objects, o.DeepCopy())
		}
	case *pkgvarapi.PackageVariantList:
		err = yaml.Unmarshal([]byte(pvList), v)
		for _, o := range v.Items {
			f.objects = append(f.objects, o.DeepCopy())
		}
	default:
		return fmt.Errorf("unsupported type")
	}
	return err
}
