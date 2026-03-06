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
	"encoding/json"
	"fmt"

	"github.com/kptdev/kpt/internal/util/pathutil"
	rgfilev1alpha1 "github.com/kptdev/kpt/pkg/api/resourcegroup/v1alpha1"
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
	absPkgPath, _, err := pathutil.ResolveAbsAndRelPaths(r.PkgPath)
	if err != nil {
		return nil, err
	}

	var objs []*unstructured.Unstructured
	nodes, err := (&kio.LocalPackageReader{
		PackagePath:     absPkgPath,
		WrapBareSeqNode: true,
	}).Read()
	if err != nil {
		return objs, err
	}

	for _, n := range nodes {
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
		if u.GroupVersionKind() == rgfilev1alpha1.ResourceGroupGVK() {
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
//
//nolint:interfacer
func kyamlNodeToUnstructured(n *yaml.RNode) (*unstructured.Unstructured, error) {
	if err := validateMetadataStringMaps(n); err != nil {
		return nil, err
	}
	b, err := n.MarshalJSON()
	if err != nil {
		return nil, err
	}

	var m map[string]any
	err = json.Unmarshal(b, &m)
	if err != nil {
		return nil, err
	}

	return &unstructured.Unstructured{
		Object: m,
	}, nil
}

// validateMetadataStringMaps inspects the raw YAML nodes for metadata.annotations
// and metadata.labels to ensure all values are strings. This prevents silent data
// loss that occurs when non-string values (booleans, integers, floats) are
// unmarshaled into map[string]string fields.
//
// Parameters:
//   - n: The RNode representing a Kubernetes resource
//
// Returns:
//   - error: A descriptive error if any annotation or label value is not a string,
//     nil otherwise
func validateMetadataStringMaps(n *yaml.RNode) error {
	metadata := n.Field("metadata")
	if metadata == nil || metadata.Value == nil {
		return nil
	}

	// Validate annotations
	if err := validateStringMap(metadata.Value, "annotations", "annotation"); err != nil {
		return err
	}

	// Validate labels
	if err := validateStringMap(metadata.Value, "labels", "label"); err != nil {
		return err
	}

	return nil
}

// validateStringMap checks that all values in a metadata field (annotations or labels)
// are strings. It inspects the YAML tag of each value to detect non-string types
// that would be silently dropped during unmarshaling.
//
// Parameters:
//   - metadata: The RNode for the metadata field
//   - fieldName: The name of the field to validate ("annotations" or "labels")
//   - fieldType: Human-readable name for error messages ("annotation" or "label")
//
// Returns:
//   - error: A descriptive error if any value is not a string, nil otherwise
func validateStringMap(metadata *yaml.RNode, fieldName, fieldType string) error {
	field := metadata.Field(fieldName)
	if field == nil || field.Value == nil {
		return nil
	}

	mapNode := field.Value.YNode()
	if mapNode == nil {
		return nil
	}

	// Handle explicit null (e.g., annotations: null)
	if mapNode.Tag == yaml.NodeTagNull {
		return nil
	}

	if mapNode.Kind != yaml.MappingNode {
		return fmt.Errorf("metadata.%s must be a string map", fieldName)
	}

	for i := 0; i < len(mapNode.Content); i += 2 {
		keyNode := mapNode.Content[i]
		valueNode := mapNode.Content[i+1]
		if valueNode.Kind != yaml.ScalarNode || valueNode.Tag != yaml.NodeTagString {
			return fmt.Errorf("%s %q must be a string, got %s", fieldType, keyNode.Value, yamlTagToType(valueNode))
		}
	}

	return nil
}

// yamlTagToType converts a YAML node's tag to a human-readable type name
// for use in error messages.
//
// Parameters:
//   - node: The YAML node to inspect
//
// Returns:
//   - string: A human-readable type name (e.g., "boolean", "integer", "number")
func yamlTagToType(node *yaml.Node) string {
	if node.Kind != yaml.ScalarNode {
		return "non-scalar"
	}
	switch node.Tag {
	case yaml.NodeTagBool:
		return "boolean"
	case yaml.NodeTagInt:
		return "integer"
	case yaml.NodeTagFloat:
		return "number"
	case yaml.NodeTagNull:
		return "null"
	case yaml.NodeTagString:
		return "string"
	default:
		if node.Tag == yaml.NodeTagEmpty {
			return "unknown"
		}
		return node.Tag
	}
}

const NoLocalConfigAnnoVal = "false"

// filterLocalConfig returns a new slice of Unstructured where all resources
// that are designated as local config have been filtered out. It does this
// by looking at the config.kubernetes.io/local-config annotation. Any value
// except "false" is considered to mean the resource is local config.
// Note(droot): Since we stopped giving special treatment to functionConfigs
// "false" value for the local-config annotation doesn't make much sense.
// With that we can probably enable just presence of local-config annotation
// as a way to mark that config is local. Can get rid of confusion as pointed out in the
// issue --> https://github.com/kptdev/kpt/issues/2767
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
