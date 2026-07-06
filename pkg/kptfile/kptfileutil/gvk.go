// Copyright 2026 The kpt Authors
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

package kptfileutil

import (
	kptapischema "github.com/kptdev/kpt/api/schema/v1"
	machineryschema "k8s.io/apimachinery/pkg/runtime/schema"
)

// ToKptGVK converts an apimachinery GroupVersionKind to the kpt API schema GroupVersionKind.
func ToKptGVK(gvk machineryschema.GroupVersionKind) kptapischema.GroupVersionKind {
	return kptapischema.GroupVersionKind{
		Group:   gvk.Group,
		Version: gvk.Version,
		Kind:    gvk.Kind,
	}
}

// FromKptGVK converts a kpt API schema GroupVersionKind to an apimachinery GroupVersionKind.
func FromKptGVK(gvk kptapischema.GroupVersionKind) machineryschema.GroupVersionKind {
	return machineryschema.GroupVersionKind{
		Group:   gvk.Group,
		Version: gvk.Version,
		Kind:    gvk.Kind,
	}
}
