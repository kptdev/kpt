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
	"github.com/GoogleContainerTools/kpt/porch/api/porch"
	apiv1alpha1 "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/engine"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewRESTStorage(scheme *runtime.Scheme, codecs serializer.CodecFactory, cad engine.CaDEngine, coreClient client.WithWatch) (genericapiserver.APIGroupInfo, error) {
	packages := &packages{
		TableConvertor: packageTableConvertor,
		packageCommon: packageCommon{
			scheme:         scheme,
			cad:            cad,
			gr:             porch.Resource("packages"),
			coreClient:     coreClient,
			updateStrategy: packageStrategy{},
			createStrategy: packageStrategy{},
		},
	}

	packageRevisions := &packageRevisions{
		TableConvertor: packageRevisionTableConvertor,
		packageCommon: packageCommon{
			scheme:         scheme,
			cad:            cad,
			gr:             porch.Resource("packagerevisions"),
			coreClient:     coreClient,
			updateStrategy: packageRevisionStrategy{},
			createStrategy: packageRevisionStrategy{},
		},
	}

	packageRevisionsApproval := &packageRevisionsApproval{
		common: packageCommon{
			scheme:         scheme,
			cad:            cad,
			coreClient:     coreClient,
			gr:             porch.Resource("packagerevisions"),
			updateStrategy: packageRevisionApprovalStrategy{},
			createStrategy: packageRevisionApprovalStrategy{},
		},
	}

	packageRevisionResources := &packageRevisionResources{
		TableConvertor: packageRevisionResourcesTableConvertor,
		packageCommon: packageCommon{
			scheme:     scheme,
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
		apiv1alpha1.SchemeGroupVersion.Version: {
			"packages":                  packages,
			"packagerevisions":          packageRevisions,
			"packagerevisions/approval": packageRevisionsApproval,
			"packagerevisionresources":  packageRevisionResources,
			"functions":                 functions,
		},
	}

	{
		gvk := schema.GroupVersionKind{
			Group:   apiv1alpha1.GroupName,
			Version: apiv1alpha1.SchemeGroupVersion.Version,
			Kind:    "Package",
		}

		scheme.AddFieldLabelConversionFunc(gvk, convertPackageFieldSelector)
	}
	{
		gvk := schema.GroupVersionKind{
			Group:   apiv1alpha1.GroupName,
			Version: apiv1alpha1.SchemeGroupVersion.Version,
			Kind:    "PackageRevision",
		}

		scheme.AddFieldLabelConversionFunc(gvk, convertPackageRevisionFieldSelector)
	}
	{
		gvk := schema.GroupVersionKind{
			Group:   apiv1alpha1.GroupName,
			Version: apiv1alpha1.SchemeGroupVersion.Version,
			Kind:    "PackageRevisionResources",
		}

		scheme.AddFieldLabelConversionFunc(gvk, convertPackageRevisionResourcesFieldSelector)
	}

	return group, nil
}
