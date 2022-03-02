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

package builtins

import (
	"io"
	"path"

	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
)

const pkgContextFile = "package-context.yaml"
const pkgContextName = "kptfile.kpt.dev"

// PackageContextGenerator is a built-in KRM function that generates
// a KRM object that contains package context information that can be
// used by functions such as `set-namespace` to customize package with
// minimal configuration.
type PackageContextGenerator struct{}

func (pc *PackageContextGenerator) Run(r io.Reader, w io.Writer) error {
	rw := &kio.ByteReadWriter{
		Reader:                r,
		Writer:                w,
		KeepReaderAnnotations: true,
	}
	return framework.Execute(pc, rw)
}

func (pc *PackageContextGenerator) Process(resourceList *framework.ResourceList) error {
	var contextResources, updatedResources []*yaml.RNode

	// This loop does the following:
	// - Filters out package context resources from the input resources
	// - Generates a package context resource for each kpt package (i.e Kptfile)
	for _, resource := range resourceList.Items {
		if isPkgContext(resource) {
			// drop existing package context resources
			continue
		}
		updatedResources = append(updatedResources, resource)
		if isKptfile(resource) {
			pkgContext, err := pkgContextResource(resource)
			if err != nil {
				resourceList.Results = framework.Results{
					&framework.Result{
						Message:  err.Error(),
						Severity: framework.Error,
					},
				}
				return resourceList.Results
			}
			contextResources = append(contextResources, pkgContext)
		}
	}

	for _, resource := range contextResources {
		updatedResources = append(updatedResources, resource)
		resourcePath, _, _ := kioutil.GetFileAnnotations(resource)
		resourceList.Results = append(resourceList.Results, &framework.Result{
			Message:  "generated package context",
			Severity: framework.Info,
			File:     &framework.File{Path: resourcePath, Index: 0},
		})
	}
	resourceList.Items = updatedResources
	return nil
}

// pkgContextResource generates package context resource from a given
// Kptfile. The resource is generated adjacent to the Kptfile of the package.
func pkgContextResource(kf *yaml.RNode) (*yaml.RNode, error) {
	cm := yaml.MustParse(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: cm
  annotations:
    config.kubernetes.io/local-config: "true"
    internal.config.kubernetes.io/path: 'package-context.yaml'
data: {}
`)
	if err := cm.SetName(pkgContextName); err != nil {
		return nil, err
	}
	kptfilePath, _, err := kioutil.GetFileAnnotations(kf)
	if err != nil {
		return nil, err
	}
	annotations := map[string]string{
		kioutil.PathAnnotation: path.Join(path.Dir(kptfilePath), pkgContextFile),
	}

	for k, v := range annotations {
		if _, err := cm.Pipe(yaml.SetAnnotation(k, v)); err != nil {
			return nil, err
		}
	}
	cm.SetDataMap(map[string]string{
		"name": kf.GetName(),
	})
	return cm, nil
}

func isKptfile(resource *yaml.RNode) bool {
	gvk := resid.GvkFromNode(resource)
	return gvk.Kind == kptfilev1.KptFileName &&
		gvk.ApiVersion() == kptfilev1.TypeMeta.APIVersion
}

func isPkgContext(resource *yaml.RNode) bool {
	gvk := resid.GvkFromNode(resource)
	return resource.GetName() == pkgContextName && gvk.Kind == "ConfigMap" && gvk.ApiVersion() == "v1"
}
