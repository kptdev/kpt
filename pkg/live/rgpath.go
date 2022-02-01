// Copyright 2020 Google LLC.
// SPDX-License-Identifier: Apache-2.0

package live

import (
	"encoding/json"

	"github.com/GoogleContainerTools/kpt/internal/pkg"
	rgfilev1alpha1 "github.com/GoogleContainerTools/kpt/pkg/api/resourcegroup/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/cli-utils/pkg/manifestreader"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
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
		PackagePath:     r.PkgPath,
		WrapBareSeqNode: true,
	}).Read()
	if err != nil {
		return objs, err
	}

	for _, n := range nodes {
		relPath, _, err := kioutil.GetFileAnnotations(n)
		if err != nil {
			return objs, err
		}
		if fcPaths.Has(relPath) && !isExplicitNotLocalConfig(n) {
			continue
		}

		if err := removeAnnotations(n, kioutil.IndexAnnotation, kioutil.LegacyIndexAnnotation); err != nil { // nolint:staticcheck
			return objs, err
		}
		u, err := kyamlNodeToUnstructured(n)
		if err != nil {
			return objs, err
		}

		// Skip if current file is a ResourceGroup resource. We do not want to apply/delete any ResourceGroup CRs when we
		// run any `kpt live` commands on a package. Instead, we have specific logic in place for handling ResourceGroups in
		// the live cluster.
		if u.GetKind() == rgfilev1alpha1.RGFileKind && u.GetAPIVersion() == rgfilev1alpha1.DefaultMeta.APIVersion {
			continue
		}
		objs = append(objs, u)
	}

	objs = filterLocalConfig(objs)
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

const NoLocalConfigAnnoVal = "false"

// isExplicitNotLocalConfig checks whether the resource has been explicitly
// label as NOT being local config. It checks for the config.kubernetes.io/local-config
// annotation with a value of "false".
func isExplicitNotLocalConfig(n *yaml.RNode) bool {
	if val, found := n.GetAnnotations()[filters.LocalConfigAnnotation]; found {
		if val == NoLocalConfigAnnoVal {
			return true
		}
	}
	return false
}

// filterLocalConfig returns a new slice of Unstructured where all resources
// that are designated as local config have been filtered out. It does this
// by looking at the config.kubernetes.io/local-config annotation. Any value
// except "false" is considered to mean the resource is local config.
func filterLocalConfig(objs []*unstructured.Unstructured) []*unstructured.Unstructured {
	var filteredObjs []*unstructured.Unstructured
	for _, obj := range objs {
		annoVal, found := obj.GetAnnotations()[filters.LocalConfigAnnotation]
		if found && annoVal != NoLocalConfigAnnoVal {
			continue
		}
		filteredObjs = append(filteredObjs, obj)
	}
	return filteredObjs
}
