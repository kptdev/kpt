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

package remoterootsyncset

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"
)

func applyObjects(ctx context.Context, restMapper meta.RESTMapper, client dynamic.Interface, objects []*unstructured.Unstructured, patchOptions metav1.PatchOptions) error {
	for _, obj := range objects {
		name := obj.GetName()
		ns := obj.GetNamespace()
		gvk := obj.GetObjectKind().GroupVersionKind()

		restMapping, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return fmt.Errorf("error getting rest mapping for %v: %w", gvk, err)
		}
		gvr := restMapping.Resource

		var dynamicResource dynamic.ResourceInterface

		switch restMapping.Scope.Name() {
		case meta.RESTScopeNameNamespace:
			if ns == "" {
				return fmt.Errorf("namespace expected but not provided for object %v %s", gvk, obj.GetName())
			}
			dynamicResource = client.Resource(gvr).Namespace(ns)

		case meta.RESTScopeNameRoot:
			dynamicResource = client.Resource(gvr)

		default:
			return fmt.Errorf("unknown scope for gvk %s: %q", gvk, restMapping.Scope.Name())
		}

		j, err := obj.MarshalJSON()
		if err != nil {
			return fmt.Errorf("failed to marshal object %s %s/%s to JSON: %w", gvk, name, ns, err)
		}

		rs, err := dynamicResource.Patch(ctx, name, types.ApplyPatchType, j, patchOptions)
		if err != nil {
			return fmt.Errorf("failed to patch object %s %s/%s: %w", gvk, name, ns, err)
		} else {
			klog.Infof("Create/Update resource %s as %v", rootSyncName, rs)
		}
	}
	return nil

}
