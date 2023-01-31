// Copyright 2022 Google LLC
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
	output []string
	client.Client
}

var _ client.Client = &fakeClient{}

func (f *fakeClient) Create(_ context.Context, obj client.Object, _ ...client.CreateOption) error {
	f.output = append(f.output, fmt.Sprintf("creating object: %s", obj.GetName()))
	return nil
}

func (f *fakeClient) Delete(_ context.Context, obj client.Object, _ ...client.DeleteOption) error {
	f.output = append(f.output, fmt.Sprintf("deleting object: %s", obj.GetName()))
	return nil
}

func (f *fakeClient) List(_ context.Context, obj client.ObjectList, _ ...client.ListOption) error {
	f.output = append(f.output, fmt.Sprintf("listing objects"))
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
    name: my-pv-1
  spec:
    upstream:
      repo: up
      package: up
      revision: up
    downstream:
      repo: dn-1
      package: dn-1
- apiVersion: config.porch.kpt.dev
  kind: PackageVariant
  metadata:
    name: my-pv-2
  spec:
    upstream:
      repo: up
      package: up
      revision: up
    downstream:
      repo: dn-2
      package: dn-2`

	switch v := obj.(type) {
	case *unstructured.UnstructuredList:
		return yaml.Unmarshal([]byte(podList), v)
	case *pkgvarapi.PackageVariantList:
		return yaml.Unmarshal([]byte(pvList), v)
	default:
		return nil
	}
}
