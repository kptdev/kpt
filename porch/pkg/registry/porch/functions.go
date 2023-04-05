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

package porch

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/engine"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type functions struct {
	rest.TableConvertor

	cad        engine.CaDEngine
	coreClient client.Client
}

var _ rest.Storage = &functions{}
var _ rest.Scoper = &functions{}
var _ rest.Lister = &functions{}
var _ rest.Getter = &functions{}

func (f *functions) New() runtime.Object {
	return &v1alpha1.Function{}
}

func (f *functions) Destroy() {}

func (f *functions) NamespaceScoped() bool {
	return true
}

func (f *functions) NewList() runtime.Object {
	return &v1alpha1.FunctionList{}
}

// List selects resources in the storage which match to the selector. 'options' can be nil.
func (f *functions) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	var opts []client.ListOption
	if ns, ok := request.NamespaceFrom(ctx); ok {
		opts = append(opts, client.InNamespace(ns))
	}

	var repositories configapi.RepositoryList
	if err := f.coreClient.List(ctx, &repositories, opts...); err != nil {
		return nil, fmt.Errorf("failed to list registered repositories: %w", err)
	}

	result := &v1alpha1.FunctionList{}

	for i := range repositories.Items {
		repo := &repositories.Items[i]
		fns, err := f.cad.ListFunctions(ctx, repo)
		if err != nil {
			return nil, fmt.Errorf("failed to list repository %s functions: %w", repositories.Items[i].Name, err)
		}
		for _, f := range fns {
			api, err := f.GetFunction()
			if err != nil {
				return nil, fmt.Errorf("failed to get function details %s: %w", f.Name(), err)
			}
			result.Items = append(result.Items, *api)
		}
	}

	return result, nil
}

func (f *functions) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	var repositoryKey client.ObjectKey
	if ns, ok := request.NamespaceFrom(ctx); !ok {
		return nil, apierrors.NewBadRequest("namespace must be specified")
	} else {
		repositoryKey.Namespace = ns
	}

	if fn, err := parseFunctionName(name); err != nil {
		return nil, apierrors.NewNotFound(schema.GroupResource(v1alpha1.FunctionGVR.GroupResource()), name)
	} else {
		repositoryKey.Name = fn.repository
	}

	var repository configapi.Repository
	if err := f.coreClient.Get(ctx, repositoryKey, &repository); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, apierrors.NewNotFound(schema.GroupResource(v1alpha1.FunctionGVR.GroupResource()), name)
		}
		return nil, fmt.Errorf("error getting repository %s: %w", repositoryKey, err)
	}

	// TODO: implement get to avoid listing
	fns, err := f.cad.ListFunctions(ctx, &repository)
	if err != nil {
		return nil, fmt.Errorf("failed to list repository %s functions: %w", repository.Name, err)
	}

	for _, f := range fns {
		if f.Name() == name {
			return f.GetFunction()
		}
	}

	return nil, apierrors.NewNotFound(schema.GroupResource(v1alpha1.FunctionGVR.GroupResource()), name)
}

type functionName struct {
	repository, name, version string
}

func parseFunctionName(name string) (functionName, error) {
	var result functionName

	parts := strings.SplitN(name, ":", 3)
	if len(parts) != 3 {
		return result, fmt.Errorf("invalid name %q; expect name in the format <repository>:<function name>:<version>", name)
	}

	result.repository = parts[0]
	result.name = parts[1]
	result.version = parts[2]

	return result, nil
}
