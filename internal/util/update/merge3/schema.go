// Copyright 2025 The kpt Authors
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

package merge3

import (
	"maps"
	"strings"

	"github.com/kptdev/krm-functions-sdk/go/fn"
	pkgerrors "github.com/pkg/errors"
	"k8s.io/klog/v2"
	"k8s.io/kube-openapi/pkg/validation/spec"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/yaml"
)

const (
	crdKind = "CustomResourceDefinition"

	// copied from kyaml openapi/openapi.go
	kubernetesGVKExtensionKey = "x-kubernetes-group-version-kind"
)

// FilterCrds splits the input between non-CRD and CRD KubeObjects.
func FilterCrds(kos fn.KubeObjects) (filtered fn.KubeObjects, crds fn.KubeObjects) {
	crds, filtered = kos.Split(IsCrd)
	return
}

func IsCrd(ko *fn.KubeObject) bool {
	return ko.GetKind() == crdKind
}

// SchemasFromCrdKubeObjects extracts the Kustomize OpenApi definitions keyed with their GVK from multiple CRD KubeObjects
func SchemasFromCrdKubeObjects(kos fn.KubeObjects) (map[string]spec.Schema, error) {
	definitions := map[string]spec.Schema{}
	for _, ko := range kos {
		defs, err := SchemasFromCrdKubeObject(ko)
		if err != nil {
			return nil, err
		}

		// duplicate check, just in case
		// TODO: should this be an error?
		for k := range maps.Keys(defs) {
			if _, ok := definitions[k]; ok {
				klog.Warningf("duplicate schema definition %q, later occurrence will be used", k)
			}
		}

		maps.Copy(definitions, defs)
	}
	return definitions, nil
}

// SchemasFromCrdKubeObject extracts the Kustomize OpenApi definitions keyed with their GVK from a single CRD KubeObject
func SchemasFromCrdKubeObject(ko *fn.KubeObject) (map[string]spec.Schema, error) {
	if !IsCrd(ko) {
		return nil, pkgerrors.Errorf("expected kind to be %q, but is %q", crdKind, ko.GetKind())
	}

	group, found, err := ko.NestedString("spec", "group")
	if err != nil {
		return nil, pkgerrors.Wrap(err, "error getting spec.group")
	}
	if !found {
		return nil, pkgerrors.New("could not find spec.group")
	}

	kind, found, err := ko.NestedString("spec", "names", "kind")
	if err != nil {
		return nil, pkgerrors.Wrap(err, "error getting spec.names.kind")
	}
	if !found {
		return nil, pkgerrors.New("could not find spec.names.kind")
	}

	versions, found, err := ko.NestedSlice("spec", "versions")
	if err != nil {
		return nil, pkgerrors.Wrap(err, "error getting spec.versions")
	}
	if !found {
		return nil, pkgerrors.New("could not find spec.versions")
	}

	definitions := map[string]spec.Schema{}

	for i, version := range versions {
		vname, found, err := version.NestedString("name")
		if err != nil {
			return nil, pkgerrors.Wrapf(err, "error getting spec.versions[%d].name", i)
		}
		if !found {
			return nil, pkgerrors.Errorf("could not find spec.versions[%d].name", i)
		}

		schema, err := extractSchema(version)
		if err != nil {
			return nil, err
		}

		addGVKExtension(schema, group, vname, kind)

		key := strings.Join([]string{group, vname, kind}, ".")
		definitions[key] = *schema
	}

	return definitions, nil
}

// SchemasFromCrdRNode extracts all the Kustomize OpenApi definitions keyed with their GVK from a single CRD RNode
func SchemasFromCrdRNode(node *kyaml.RNode) (map[string]spec.Schema, error) {
	return SchemasFromCrdKubeObject(fn.CopyToKubeObject(node))
}

// SchemasFromCrdRNodes extracts all the Kustomize OpenApi definitions keyed with their GVK from multiple CRD RNodes
func SchemasFromCrdRNodes(nodes []*kyaml.RNode) (map[string]spec.Schema, error) {
	return SchemasFromCrdKubeObjects(fn.MoveToKubeObjects(nodes))
}

func extractSchema(version *fn.SubObject) (*spec.Schema, error) {
	schemaSO, found, err := version.NestedSubObject("schema", "openAPIV3Schema")
	if err != nil {
		return nil, pkgerrors.Wrap(err, "error getting spec.versions[#].schema.openAPIV3Schema")
	}
	if !found {
		return nil, pkgerrors.New("could not find spec.versions[#].schema.openAPIV3Schema")
	}

	schema := &spec.Schema{}
	if err := yaml.Unmarshal(schemaSO.Bytes(), schema); err != nil {
		return nil, pkgerrors.Wrap(err, "error unmarshaling spec.versions[#].schema.openAPIV3Schema into schema")
	}

	return schema, nil
}

func addGVKExtension(schema *spec.Schema, group, version, kind string) {
	schema.Extensions = map[string]any{
		kubernetesGVKExtensionKey: []map[string]string{
			{
				"group":   group,
				"version": version,
				"kind":    kind,
			},
		},
	}
}
