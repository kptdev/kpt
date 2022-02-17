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

package porch

import (
	"context"

	api "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/rest"
)

type packageRevisionsApproval struct {
	revisions *packageRevisions
}

var _ rest.Storage = &packageRevisionsApproval{}
var _ rest.Scoper = &packageRevisionsApproval{}
var _ rest.Getter = &packageRevisionsApproval{}
var _ rest.Updater = &packageRevisionsApproval{}

// New returns an empty object that can be used with Create and Update after request data has been put into it.
// This object must be a pointer type for use with Codec.DecodeInto([]byte, runtime.Object)
func (a *packageRevisionsApproval) New() runtime.Object {
	return &api.PackageRevision{}
}

// NamespaceScoped returns true if the storage is namespaced
func (a *packageRevisionsApproval) NamespaceScoped() bool {
	return true
}

func (a *packageRevisionsApproval) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return a.revisions.Get(ctx, name, options)
}

// Update finds a resource in the storage and updates it. Some implementations
// may allow updates creates the object - they should set the created boolean
// to true.
func (a *packageRevisionsApproval) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	forceAllowCreate = false // do not allow create on update
	return a.revisions.Update(ctx, name, objInfo, createValidation, updateValidation, forceAllowCreate, options)
}
