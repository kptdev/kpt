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

package status

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/testutil"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type fakeClusterReader struct {
	testutil.NoopClusterReader

	getResource *unstructured.Unstructured
	getErr      error

	listResources *unstructured.UnstructuredList
	listErr       error
}

func (f *fakeClusterReader) Get(_ context.Context, _ client.ObjectKey, u *unstructured.Unstructured) error {
	if f.getResource != nil {
		u.Object = f.getResource.Object
	}
	return f.getErr
}

func (f *fakeClusterReader) ListNamespaceScoped(_ context.Context, list *unstructured.UnstructuredList, _ string, _ labels.Selector) error {
	if f.listResources != nil {
		list.Items = f.listResources.Items
	}
	return f.listErr
}
