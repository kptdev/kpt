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

package controllerrestmapper

import (
	"fmt"
	"strings"
	"sync"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// cache is our cache of schema information.
type cache struct {
	mutex         sync.Mutex
	groupVersions map[schema.GroupVersion]*cachedGroupVersion
}

// newCache is the constructor for a cache.
func newCache() *cache {
	return &cache{
		groupVersions: make(map[schema.GroupVersion]*cachedGroupVersion),
	}
}

// cachedGroupVersion caches (all) the resource information for a particular groupversion.
type cachedGroupVersion struct {
	gv    schema.GroupVersion
	mutex sync.Mutex
	kinds map[string]cachedGVR
}

// cachedGVR caches the information for a particular resource.
type cachedGVR struct {
	Resource string
	Scope    meta.RESTScope
}

// findRESTMapping returns the RESTMapping for the specified GVK, querying discovery if not cached.
func (c *cache) findRESTMapping(discovery discovery.DiscoveryInterface, gv schema.GroupVersion, kind string) (*meta.RESTMapping, error) {
	c.mutex.Lock()
	cached := c.groupVersions[gv]
	if cached == nil {
		cached = &cachedGroupVersion{gv: gv}
		c.groupVersions[gv] = cached
	}
	c.mutex.Unlock()
	return cached.findRESTMapping(discovery, kind)
}

// findRESTMapping returns the RESTMapping for the specified GVK, querying discovery if not cached.
func (c *cachedGroupVersion) findRESTMapping(discovery discovery.DiscoveryInterface, kind string) (*meta.RESTMapping, error) {
	kinds, err := c.fetch(discovery)
	if err != nil {
		return nil, err
	}

	cached, found := kinds[kind]
	if !found {
		return nil, nil
	}
	return &meta.RESTMapping{
		Resource:         c.gv.WithResource(cached.Resource),
		GroupVersionKind: c.gv.WithKind(kind),
		Scope:            cached.Scope,
	}, nil
}

// fetch returns the metadata, fetching it if not cached.
func (c *cachedGroupVersion) fetch(discovery discovery.DiscoveryInterface) (map[string]cachedGVR, error) {
	log := log.Log

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.kinds != nil {
		return c.kinds, nil
	}

	log.Info("discovering server resources for group/version", "gv", c.gv.String())
	resourceList, err := discovery.ServerResourcesForGroupVersion(c.gv.String())
	if err != nil {
		// We treat "no match" as an empty result, but any other error percolates back up
		if meta.IsNoMatchError(err) || apierrors.IsNotFound(err) {
			return nil, nil
		} else {
			klog.Infof("unexpected error from ServerResourcesForGroupVersion(%v): %v", c.gv, err)
			return nil, fmt.Errorf("error from ServerResourcesForGroupVersion(%v): %w", c.gv, err)
		}
	}

	kinds := make(map[string]cachedGVR)
	for i := range resourceList.APIResources {
		resource := resourceList.APIResources[i]

		// if we have a slash, then this is a subresource and we shouldn't create mappings for those.
		if strings.Contains(resource.Name, "/") {
			continue
		}

		scope := meta.RESTScopeRoot
		if resource.Namespaced {
			scope = meta.RESTScopeNamespace
		}
		kinds[resource.Kind] = cachedGVR{
			Resource: resource.Name,
			Scope:    scope,
		}
	}
	c.kinds = kinds
	return kinds, nil
}
