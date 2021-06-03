// Copyright 2020 Google LLC.
// SPDX-License-Identifier: Apache-2.0

package live

import (
	"encoding/json"

	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/cli-utils/pkg/manifestreader"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// ResourceGroupPathManifestReader encapsulates the default path
// manifest reader.
type ResourceGroupPathManifestReader struct {
	PkgPath string

	manifestreader.ReaderOptions
}

// Read reads the manifests and returns them as Info objects.
// Generates and adds a ResourceGroup inventory object from
// Kptfile data. If unable to generate the ResourceGroup inventory
// object from the Kptfile, it is NOT an error.
func (r *ResourceGroupPathManifestReader) Read() ([]*unstructured.Unstructured, error) {
	p, err := pkg.New(r.PkgPath)
	if err != nil {
		return nil, err
	}

	// Lookup all files referenced by all subpackages.
	fcPaths, err := pkg.FunctionConfigFilePaths(p.UniquePath, true)
	if err != nil {
		return nil, err
	}

	var objs []*unstructured.Unstructured
	nodes, err := (&kio.LocalPackageReader{
		PackagePath: r.PkgPath,
	}).Read()
	if err != nil {
		return objs, err
	}

	for _, n := range nodes {
		relPath, _, err := kioutil.GetFileAnnotations(n)
		if err != nil {
			return objs, err
		}
		if fcPaths.Has(relPath) {
			continue
		}

		if err := removeAnnotations(n, kioutil.IndexAnnotation); err != nil {
			return objs, err
		}

		u, err := kyamlNodeToUnstructured(n)
		if err != nil {
			return objs, err
		}
		objs = append(objs, u)
	}

	err = manifestreader.SetNamespaces(r.Mapper, objs, r.Namespace, r.EnforceNamespace)
	return objs, err
}

// removeAnnotations removes the specified kioutil annotations from the resource.
func removeAnnotations(n *yaml.RNode, annotations ...kioutil.AnnotationKey) error {
	for _, a := range annotations {
		err := n.PipeE(yaml.ClearAnnotation(a))
		if err != nil {
			return err
		}
	}
	return nil
}

// kyamlNodeToUnstructured take a resource represented as a kyaml RNode and
// turns it into an Unstructured object.
//nolint:interfacer
func kyamlNodeToUnstructured(n *yaml.RNode) (*unstructured.Unstructured, error) {
	b, err := n.MarshalJSON()
	if err != nil {
		return nil, err
	}

	var m map[string]interface{}
	err = json.Unmarshal(b, &m)
	if err != nil {
		return nil, err
	}

	return &unstructured.Unstructured{
		Object: m,
	}, nil
}
