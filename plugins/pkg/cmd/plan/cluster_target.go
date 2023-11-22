// Copyright 2023 The kpt Authors
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

package plan

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/kubebuilder-declarative-pattern/pkg/restmapper"
)

// ClusterTarget supports actions against a running kubernetes cluster.
type ClusterTarget struct {
	client     dynamic.Interface
	restMapper resourceFinder
}

func NewClusterTarget(restConfig *rest.Config) (*ClusterTarget, error) {
	client, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("creating kubernetes client: %w", err)
	}

	restMapper, err := restmapper.NewControllerRESTMapper(restConfig)
	if err != nil {
		return nil, fmt.Errorf("building REST mapper: %w", err)
	}

	return &ClusterTarget{
		client:     client,
		restMapper: restMapper,
	}, nil
}

type resourceFinder interface {
	RESTMapping(gk schema.GroupKind, versions ...string) (*meta.RESTMapping, error)
}

// ResourceForGVK gets the GVR / Scope for the specified object.
func (c *ClusterTarget) ResourceForGVK(ctx context.Context, gvk schema.GroupVersionKind) (*clusterResourceTarget, error) {
	mapping, err := c.restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, fmt.Errorf("cannot get RESTMapping for %v: %w", gvk, err)
	}
	return &clusterResourceTarget{info: mapping, client: c.client}, nil
}

// Apply is a wrapper around applying changes to a live cluster.
func (c *clusterResourceTarget) Apply(ctx context.Context, obj *unstructured.Unstructured, options metav1.PatchOptions) (*unstructured.Unstructured, error) {
	target, err := c.buildResource(ctx, obj)
	if err != nil {
		return nil, err
	}

	j, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("error marshalling object to JSON: %w", err)
	}

	// Apply with server-side apply (specified with ApplyPatchType)
	patched, err := target.Patch(ctx, obj.GetName(), types.ApplyPatchType, j, options)
	if err != nil {
		return nil, fmt.Errorf("server-side-apply failed: %w", err)
	}

	return patched, nil
}

// buildResource creates the dynamic ResourceInterface for the object
func (c *clusterResourceTarget) buildResource(ctx context.Context, obj *unstructured.Unstructured) (dynamic.ResourceInterface, error) {
	if c.info.Scope == meta.RESTScopeRoot {
		return c.client.Resource(c.info.Resource), nil
	} else {
		namespace := obj.GetNamespace()
		if namespace == "" {
			return nil, fmt.Errorf("namespace was not set, but is required for namespace-scoped objects")
		}
		return c.client.Resource(c.info.Resource).Namespace(namespace), nil
	}
}

// Get reads the current version of an object.
func (c *clusterResourceTarget) Get(ctx context.Context, obj *unstructured.Unstructured, options metav1.GetOptions) (*unstructured.Unstructured, error) {
	target, err := c.buildResource(ctx, obj)
	if err != nil {
		return nil, err
	}

	existing, err := target.Get(ctx, obj.GetName(), options)
	if err != nil {
		return nil, fmt.Errorf("get failed: %w", err)
	}

	return existing, nil
}

type clusterResourceTarget struct {
	info *meta.RESTMapping

	client dynamic.Interface
}
