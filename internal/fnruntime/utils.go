// Copyright 2021 The kpt Authors
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

package fnruntime

import (
	"bytes"
	"fmt"
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/internal/types"
	fnresult "github.com/GoogleContainerTools/kpt/pkg/api/fnresult/v1"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// ResourceIDAnnotation is used to uniquely identify the resource during round trip
// to and from a function execution. This annotation is meant to be consumed by
// kpt during round trip and should be deleted after that
const ResourceIDAnnotation = "internal.config.k8s.io/kpt-resource-id"

// SaveResults saves results gathered from running the pipeline at specified dir in the input FileSystem.
func SaveResults(fsys filesys.FileSystem, resultsDir string, fnResults *fnresult.ResultList) (string, error) {
	if resultsDir == "" {
		return "", nil
	}
	filePath := filepath.Join(resultsDir, "results.yaml")
	out := &bytes.Buffer{}

	// use kyaml encoder to ensure consistent indentation
	e := yaml.NewEncoderWithOptions(out, &yaml.EncoderOptions{SeqIndent: yaml.WideSequenceStyle})
	err := e.Encode(fnResults)
	if err != nil {
		return "", err
	}

	err = fsys.WriteFile(filePath, out.Bytes())
	if err != nil {
		return "", err
	}

	return filePath, nil
}

// MergeWithInput merges the transformed output with input resources
// input: all input resources, selectedInput: selected input resources
// output: output resources as the result of function on selectedInput resources
// for input: A,B,C,D; selectedInput: A,B; output: A,E(A transformed, B Deleted, E Added)
// the result should be A,C,D,E
// resources are identified by kpt-resource-id annotation
func MergeWithInput(output, selectedInput, input []*yaml.RNode) []*yaml.RNode {
	var result []*yaml.RNode
	for i := range input {
		presentInSelectedInput := presentIn(input[i], selectedInput)
		presentInOutput := presentIn(input[i], output)
		if !presentInSelectedInput {
			// this resource is untouched
			result = append(result, input[i])
			continue
		}

		if presentInOutput {
			// this resource modified by function, so replace it from the output
			result = append(result, nodeWithResourceID(input[i].GetAnnotations()[ResourceIDAnnotation], output))
		}
		// if presentInSelectedInput and !presentInOutput the resource is deleted, so ignore it
	}

	// add function generated resources to the result
	for i := range output {
		if output[i].GetAnnotations()[ResourceIDAnnotation] == "" {
			result = append(result, output[i])
		}
	}

	return result
}

// nodeWithResourceID returns the node with the input resourceId
func nodeWithResourceID(resourceID string, input []*yaml.RNode) *yaml.RNode {
	for _, node := range input {
		if node.GetAnnotations()[ResourceIDAnnotation] == resourceID {
			return node
		}
	}
	return nil
}

// presentIn returns true if the targetNode identified by kpt-resource-id annotation
// is present in the input list of resources
func presentIn(targetNode *yaml.RNode, input []*yaml.RNode) bool {
	id := targetNode.GetAnnotations()[ResourceIDAnnotation]
	for _, node := range input {
		if node.GetAnnotations()[ResourceIDAnnotation] == id {
			return true
		}
	}
	return false
}

// SetResourceIds adds kpt-resource-id annotation to each input resource
func SetResourceIds(input []*yaml.RNode) error {
	id := 0
	for i := range input {
		idStr := fmt.Sprintf("%v", id)
		err := input[i].PipeE(yaml.SetAnnotation(ResourceIDAnnotation, idStr))
		if err != nil {
			return err
		}
		id++
	}
	return nil
}

// DeleteResourceIds removes the kpt-resource-id annotation from all resources
func DeleteResourceIds(input []*yaml.RNode) error {
	for i := range input {
		err := input[i].PipeE(yaml.ClearAnnotation(ResourceIDAnnotation))
		if err != nil {
			return err
		}
	}
	return nil
}

type SelectionContext struct {
	RootPackagePath types.UniquePath
}

// SelectInput returns the selected resources based on criteria in selectors
func SelectInput(input []*yaml.RNode, selectors, exclusions []kptfilev1.Selector, _ *SelectionContext) ([]*yaml.RNode, error) {
	var selectedInput []*yaml.RNode
	if len(selectors) == 0 {
		selectedInput = input
	} else {
		for _, node := range input {
			for _, selector := range selectors {
				if IsMatch(node, selector) {
					selectedInput = append(selectedInput, node)
				}
			}
		}
	}
	if len(exclusions) == 0 {
		return selectedInput, nil
	}
	var filteredInput []*yaml.RNode
	for _, node := range selectedInput {
		matchesExclusion := false
		for _, exclusion := range exclusions {
			if !exclusion.IsEmpty() && IsMatch(node, exclusion) {
				matchesExclusion = true
				break
			}
		}
		if !matchesExclusion {
			filteredInput = append(filteredInput, node)
		}
	}
	return filteredInput, nil
}

// IsMatch returns true if the resource matches input selection criteria
func IsMatch(node *yaml.RNode, selector kptfilev1.Selector) bool {
	// keep expanding with new selectors
	return nameMatch(node, selector) && namespaceMatch(node, selector) &&
		kindMatch(node, selector) && apiVersionMatch(node, selector) &&
		labelMatch(node, selector) && annoMatch(node, selector)
}

// nameMatch returns true if the resource name matches input selection criteria
func nameMatch(node *yaml.RNode, selector kptfilev1.Selector) bool {
	return selector.Name == "" || selector.Name == node.GetName()
}

// namespaceMatch returns true if the resource namespace matches input selection criteria
func namespaceMatch(node *yaml.RNode, selector kptfilev1.Selector) bool {
	return selector.Namespace == "" || selector.Namespace == node.GetNamespace()
}

// kindMatch returns true if the resource kind matches input selection criteria
func kindMatch(node *yaml.RNode, selector kptfilev1.Selector) bool {
	return selector.Kind == "" || selector.Kind == node.GetKind()
}

// apiVersionMatch returns true if the resource apiVersion matches input selection criteria
func apiVersionMatch(node *yaml.RNode, selector kptfilev1.Selector) bool {
	return selector.APIVersion == "" || selector.APIVersion == node.GetApiVersion()
}

// labelMatch returns true if the resource labels match input selection criteria
func labelMatch(node *yaml.RNode, selector kptfilev1.Selector) bool {
	nodeLabels := node.GetLabels()
	for sk, sv := range selector.Labels {
		if nv, found := nodeLabels[sk]; !found || sv != nv {
			return false
		}
	}
	return true
}

// annoMatch returns true if the resource annotations match input selection criteria
func annoMatch(node *yaml.RNode, selector kptfilev1.Selector) bool {
	nodeAnnos := node.GetAnnotations()
	for sk, sv := range selector.Annotations {
		if nv, found := nodeAnnos[sk]; !found || sv != nv {
			return false
		}
	}
	return true
}

func NewConfigMap(data map[string]string) (*yaml.RNode, error) {
	node := yaml.NewMapRNode(&data)
	if node == nil {
		return nil, nil
	}
	// create a ConfigMap only for configMap config
	configMap := yaml.MustParse(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: function-input
data: {}
`)
	if err := node.VisitFields(func(node *yaml.MapNode) error {
		v := node.Value.YNode()
		v.Tag = yaml.NodeTagString
		node.Value.SetYNode(v)
		return nil
	}); err != nil {
		return nil, err
	}
	if err := configMap.PipeE(yaml.SetField("data", node)); err != nil {
		return nil, err
	}
	return configMap, nil
}
