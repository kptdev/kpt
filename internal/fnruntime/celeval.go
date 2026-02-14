// Copyright 2026 The kpt and Nephio Authors
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
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// CELEvaluator evaluates CEL expressions against KRM resources
type CELEvaluator struct {
	env *cel.Env
}

// NewCELEvaluator creates a new CEL evaluator with the standard environment
func NewCELEvaluator() (*CELEvaluator, error) {
	env, err := cel.NewEnv(
		cel.Variable("resources", cel.ListType(cel.DynType)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	return &CELEvaluator{
		env: env,
	}, nil
}

// EvaluateCondition evaluates a CEL condition expression against a list of resources
// Returns true if the condition is met, false otherwise
func (e *CELEvaluator) EvaluateCondition(condition string, resources []*yaml.RNode) (bool, error) {
	if condition == "" {
		return true, nil
	}

	// Compile the expression
	ast, issues := e.env.Compile(condition)
	if issues != nil && issues.Err() != nil {
		return false, fmt.Errorf("failed to compile CEL expression: %w", issues.Err())
	}

	// Create the program
	prg, err := e.env.Program(ast)
	if err != nil {
		return false, fmt.Errorf("failed to create CEL program: %w", err)
	}

	// Convert resources to a format suitable for CEL
	resourceList, err := e.resourcesToList(resources)
	if err != nil {
		return false, fmt.Errorf("failed to convert resources: %w", err)
	}

	// Evaluate the expression
	out, _, err := prg.Eval(map[string]interface{}{
		"resources": resourceList,
	})
	if err != nil {
		return false, fmt.Errorf("failed to evaluate CEL expression: %w", err)
	}

	// Extract the boolean result
	result, ok := out.(types.Bool)
	if !ok {
		return false, fmt.Errorf("CEL expression must return a boolean, got %T", out)
	}

	return bool(result), nil
}

// resourcesToList converts RNodes to a list of maps for CEL evaluation
func (e *CELEvaluator) resourcesToList(resources []*yaml.RNode) ([]interface{}, error) {
	result := make([]interface{}, 0, len(resources))

	for _, resource := range resources {
		// Convert each resource to a map
		resourceMap, err := e.resourceToMap(resource)
		if err != nil {
			return nil, err
		}
		result = append(result, resourceMap)
	}

	return result, nil
}

// resourceToMap converts a single RNode to a map for CEL evaluation
func (e *CELEvaluator) resourceToMap(resource *yaml.RNode) (map[string]interface{}, error) {
	// Get the YAML string representation
	yamlStr, err := resource.String()
	if err != nil {
		return nil, fmt.Errorf("failed to convert resource to string: %w", err)
	}

	// Parse into a generic map
	var result map[string]interface{}
	err = yaml.Unmarshal([]byte(yamlStr), &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal resource: %w", err)
	}

	return result, nil
}

// Helper functions for common CEL operations

// ResourceExists checks if any resource matches the given predicate
// This is exposed as a CEL macro/function
func ResourceExists(resources []interface{}, predicate func(interface{}) bool) bool {
	for _, r := range resources {
		if predicate(r) {
			return true
		}
	}
	return false
}

// FilterResources filters resources based on a predicate
func FilterResources(resources []interface{}, predicate func(interface{}) bool) []interface{} {
	result := make([]interface{}, 0)
	for _, r := range resources {
		if predicate(r) {
			result = append(result, r)
		}
	}
	return result
}
