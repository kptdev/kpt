// Copyright 2020 Google LLC.
// SPDX-License-Identifier: Apache-2.0

package live

import (
	"bytes"
	"io"

	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/cli-utils/pkg/manifestreader"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	excludedGKs = []schema.GroupKind{
		{
			Group: kptfilev1.KptFileGroup,
			Kind:  kptfilev1.KptFileKind,
		},
	}
)

// ResourceGroupStreamManifestReader encapsulates the default stream
// manifest reader.
type ResourceGroupStreamManifestReader struct {
	ReaderName string
	Reader     io.Reader

	manifestreader.ReaderOptions
}

// Read reads the manifests and returns them as Info objects.
// Transforms the Kptfile into the ResourceGroup inventory object,
// and appends it to the rest of the standard StreamManifestReader
// generated objects. Returns an error if one occurs. If the
// ResourceGroup inventory object does not exist, it is NOT an error.
func (p *ResourceGroupStreamManifestReader) Read() ([]*unstructured.Unstructured, error) {
	var objs []*unstructured.Unstructured
	nodes, err := (&kio.ByteReader{
		Reader: p.Reader,
	}).Read()
	if err != nil {
		return objs, err
	}

	for _, n := range nodes {
		if isExcluded(n) {
			continue
		}

		err = removeAnnotations(n, kioutil.IndexAnnotation)
		if err != nil {
			return objs, err
		}
		u, err := kyamlNodeToUnstructured(n)
		if err != nil {
			return objs, err
		}
		objs = append(objs, u)
	}

	err = manifestreader.SetNamespaces(p.Mapper, objs, p.Namespace, p.EnforceNamespace)
	return objs, err
}

func isExcluded(n *yaml.RNode) bool {
	kind := n.GetKind()
	group, _ := resid.ParseGroupVersion(n.GetApiVersion())
	for _, gk := range excludedGKs {
		if kind == gk.Kind && group == gk.Group {
			return true
		}
	}
	return false
}

var kptFileTemplate = kptfilev1.KptFile{ResourceMeta: kptfilev1.TypeMeta}

// isKptfile returns true if the passed resource config is a Kptfile; false otherwise
func isKptfile(resource []byte) bool {
	d := yaml.NewDecoder(bytes.NewReader(resource))
	d.KnownFields(true)
	if err := d.Decode(&kptFileTemplate); err == nil {
		return kptFileTemplate.ResourceMeta.TypeMeta == kptfilev1.TypeMeta.TypeMeta
	}
	return false
}
