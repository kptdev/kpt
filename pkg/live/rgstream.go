// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package live

import (
	"bytes"
	"fmt"
	"io"

	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/klog"
	"sigs.k8s.io/cli-utils/pkg/manifestreader"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// ResourceGroupStreamManifestReader encapsulates the default stream
// manifest reader.
type ResourceGroupStreamManifestReader struct {
	streamReader *manifestreader.StreamManifestReader
}

var ResourceSeparator = []byte("\n---\n")

// Read reads the manifests and returns them as Info objects.
// Transforms the Kptfile into the ResourceGroup inventory object,
// and appends it to the rest of the standard StreamManifestReader
// generated objects. Returns an error if one occurs. If the
// ResourceGroup inventory object does not exist, it is NOT an error.
func (p *ResourceGroupStreamManifestReader) Read() ([]*resource.Info, error) {
	var resourceBytes bytes.Buffer
	_, err := io.Copy(&resourceBytes, p.streamReader.Reader)
	if err != nil {
		return []*resource.Info{}, err
	}
	// Split the bytes into resource configs, and if the resource
	// config is a Kptfile, transform it into a ResourceGroup object.
	var rgInfo *resource.Info
	var filteredBytes bytes.Buffer
	resources := bytes.Split(resourceBytes.Bytes(), ResourceSeparator)
	for _, r := range resources {
		if !isKptfile(r) {
			r = append(r, ResourceSeparator...)
			_, err := filteredBytes.Write(r)
			if err != nil {
				return []*resource.Info{}, err
			}
		} else {
			rgInfo, err = transformKptfile(r)
			if err != nil {
				return []*resource.Info{}, err
			}
		}
	}
	// Reset the stream reader, and generate the infos. Append the
	// ResourceGroup inventory info if it exists.
	p.streamReader.Reader = bytes.NewReader(filteredBytes.Bytes())
	infos, err := p.streamReader.Read()
	if rgInfo != nil {
		infos = append(infos, rgInfo)
	}
	return infos, err
}

var kptFileTemplate = kptfile.KptFile{ResourceMeta: kptfile.TypeMeta}

// isKptfile returns true if the passed resource config is a Kptfile; false otherwise
func isKptfile(resource []byte) bool {
	d := yaml.NewDecoder(bytes.NewReader(resource))
	d.KnownFields(true)
	if err := d.Decode(&kptFileTemplate); err == nil {
		return kptFileTemplate.ResourceMeta.TypeMeta == kptfile.TypeMeta.TypeMeta
	}
	return false
}

// transformKptfile transforms the passed kptfile resource config
// into the ResourceGroup inventory object, or an error.
func transformKptfile(resource []byte) (*resource.Info, error) {
	d := yaml.NewDecoder(bytes.NewReader(resource))
	d.KnownFields(true)
	if err := d.Decode(&kptFileTemplate); err != nil {
		return nil, err
	}
	if kptFileTemplate.ResourceMeta.TypeMeta != kptfile.TypeMeta.TypeMeta {
		return nil, fmt.Errorf("invalid kptfile type: %s", kptFileTemplate.ResourceMeta.TypeMeta)
	}
	inv := kptFileTemplate.Inventory
	klog.V(4).Infof("generating ResourceGroup inventory object %s/%s/%s", inv.Namespace, inv.Name, inv.InventoryID)
	return generateInventoryObj(inv.Name, inv.Namespace, inv.InventoryID)
}
