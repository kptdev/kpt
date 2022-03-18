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
	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/controllers/pkg/apis/porch/v1alpha1"
	coreapi "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateClient(flags *genericclioptions.ConfigFlags) (client.Client, error) {
	config, err := flags.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	scheme, err := createScheme()
	if err != nil {
		return nil, err
	}

	c, err := client.New(config, client.Options{
		Scheme: scheme,
		Mapper: createRESTMapper(),
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}

func createScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()

	for _, api := range (runtime.SchemeBuilder{
		configapi.AddToScheme,
		porchapi.AddToScheme,
		coreapi.AddToScheme,
	}) {
		if err := api(scheme); err != nil {
			return nil, err
		}
	}
	return scheme, nil
}

func createRESTMapper() meta.RESTMapper {
	rm := meta.NewDefaultRESTMapper([]schema.GroupVersion{
		configapi.GroupVersion,
		porchapi.SchemeGroupVersion,
		coreapi.SchemeGroupVersion,
	})

	for _, r := range []struct {
		kind             schema.GroupVersionKind
		plural, singular string
	}{
		{kind: configapi.GroupVersion.WithKind("Repository"), plural: "repositories", singular: "repository"},
		{kind: porchapi.SchemeGroupVersion.WithKind("PackageRevision"), plural: "packagerevisions", singular: "packagerevision"},
		{kind: porchapi.SchemeGroupVersion.WithKind("PackageRevisionResources"), plural: "packagerevisionresources", singular: "packagerevisionresources"},
		{kind: porchapi.SchemeGroupVersion.WithKind("Function"), plural: "functions", singular: "function"},
		{kind: coreapi.SchemeGroupVersion.WithKind("Secret"), plural: "secrets", singular: "secret"},
	} {
		rm.AddSpecific(
			r.kind,
			r.kind.GroupVersion().WithResource(r.plural),
			r.kind.GroupVersion().WithResource(r.singular),
			meta.RESTScopeNamespace,
		)
	}

	return rm
}
