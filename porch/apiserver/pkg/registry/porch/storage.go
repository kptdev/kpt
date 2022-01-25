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

	"github.com/GoogleContainerTools/kpt/porch/api/porch"
	"github.com/GoogleContainerTools/kpt/porch/engine/pkg/engine"
	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/cache"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	genericregistry "k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewRESTStorage(scheme *runtime.Scheme, codecs serializer.CodecFactory, restOptionsGetter genericregistry.RESTOptionsGetter,
	coreClient client.WithWatch, cacheDirectory string) (genericapiserver.APIGroupInfo, error) {

	c := cache.NewCache(cacheDirectory)
	cad, err := engine.NewCaDEngine(c)

	if err != nil {
		return genericapiserver.APIGroupInfo{}, err
	}

	packageRevisions := &packageRevisions{
		TableConvertor: rest.NewDefaultTableConvertor(porch.Resource("packagerevisions")),
		packageCommon: packageCommon{
			cad:        cad,
			gr:         porch.Resource("packagerevisions"),
			coreClient: coreClient,
		},
	}

	packageRevisionResources := &packageRevisionResources{
		TableConvertor: rest.NewDefaultTableConvertor(porch.Resource("packagerevisionresources")),
		packageCommon: packageCommon{
			cad:        cad,
			gr:         porch.Resource("packagerevisionresources"),
			coreClient: coreClient,
		},
	}

	functions := &functions{
		TableConvertor: rest.NewDefaultTableConvertor(porch.Resource("functions")),
		cad:            cad,
		coreClient:     coreClient,
	}

	group := genericapiserver.NewDefaultAPIGroupInfo(porch.GroupName, scheme, metav1.ParameterCodec, codecs)

	group.VersionedResourcesStorageMap = map[string]map[string]rest.Storage{
		"v1alpha1": {
			"packagerevisions":         packageRevisions,
			"packagerevisionresources": packageRevisionResources,
			"functions":                functions,
		},
	}

	b := background{
		coreClient: coreClient,
		cache:      c,
	}
	ctx := context.Background() // TODO: support cancellation
	go b.RunForever(ctx)

	return group, nil
}
