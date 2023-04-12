// Copyright 2020 The kpt Authors
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

package live

import (
	"bytes"
	"io"

	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	rgfilev1alpha1 "github.com/GoogleContainerTools/kpt/pkg/api/resourcegroup/v1alpha1"
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
		kptfilev1.KptFileGVK().GroupKind(),
		rgfilev1alpha1.ResourceGroupGVK().GroupKind(),
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
		Reader:          p.Reader,
		WrapBareSeqNode: true,
	}).Read()
	if err != nil {
		return objs, err
	}

	for _, n := range nodes {
		if isExcluded(n) {
			continue
		}

		err = removeAnnotations(n, kioutil.IndexAnnotation, kioutil.LegacyIndexAnnotation) // nolint:staticcheck
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
