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
	"context"
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/ext"
	k8scellib "k8s.io/apiserver/pkg/cel/library"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const checkFrequency = 100

// This gives about .1 seconds of CPU time for the evaluation to run
const costLimit = 1000000

// CELEvaluator evaluates CEL expressions against KRM resources
type CELEvaluator struct {
	env *cel.Env
	prg cel.Program // Pre-compiled program for the condition
}

// NewCELEvaluator creates a new CEL evaluator with the standard environment
// for the given condition string.
func NewCELEvaluator(condition string) (*CELEvaluator, error) {
	env, err := cel.NewEnv(
		cel.Variable("resources", cel.ListType(cel.DynType)),
		// Below is a list of Env settings that is a selection of https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/cel/environment/base.go
		// General rules are for maintaining this list.
		// 1. utility functions should be available. This allows for more compatibility with k8s's own CEL conditions
		// 2. AST validation is not needed as kpt will recompile CEL expressions every time, there is no cost-saving in exiting early
		// 3. Compile time optimisations do not make sense, as each CEL expression will be evaluated once before being discarded.
		// 3. Things that are helping with authorization in k8s are not needed, as they're returning either ResourceCheck or Decision types, which are not needed for kpt
		cel.HomogeneousAggregateLiterals(),
		cel.DefaultUTCTimeZone(true),
		k8scellib.URLs(),
		k8scellib.Regex(),
		k8scellib.Lists(),
		cel.CrossTypeNumericComparisons(true),
		cel.OptionalTypes(),
		k8scellib.Quantity(),
		ext.Strings(ext.StringsVersion(2)),
		ext.Sets(),
		k8scellib.IP(),
		k8scellib.CIDR(),
		k8scellib.Format(),
		ext.TwoVarComprehensions(),
		k8scellib.SemverLib(k8scellib.SemverVersion(1)),
		ext.Lists(ext.ListsVersion(3)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	evaluator := &CELEvaluator{
		env: env,
	}

	// Pre-compile the condition if provided
	if condition != "" {
		ast, issues := env.Compile(condition)
		if issues != nil && issues.Err() != nil {
			return nil, fmt.Errorf("failed to compile CEL expression: %w", issues.Err())
		}

		// Validate that the expression returns a boolean
		if ast.OutputType() != cel.BoolType {
			return nil, fmt.Errorf("CEL expression must return a boolean, got %v", ast.OutputType())
		}

		// Create the program with a hard cost limit and cost tracking enabled
		prg, err := env.Program(ast,
			cel.CostLimit(costLimit),
			cel.InterruptCheckFrequency(checkFrequency),
			cel.CostTracking(&k8scellib.CostEstimator{}),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create CEL program: %w", err)
		}

		evaluator.prg = prg
	}

	return evaluator, nil
}

// EvaluateCondition evaluates a CEL condition expression against a list of resources
// Returns true if the condition is met, false otherwise
// The program is pre-compiled, so this just evaluates it with the given resources
func (e *CELEvaluator) EvaluateCondition(ctx context.Context, resources []*yaml.RNode) (bool, error) {
	if e.prg == nil {
		return true, nil
	}

	// Convert resources to a format suitable for CEL
	resourceList, err := e.resourcesToList(resources)
	if err != nil {
		return false, fmt.Errorf("failed to convert resources: %w", err)
	}

	// Evaluate the expression
	out, _, err := e.prg.ContextEval(ctx, map[string]interface{}{
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
// Converts yaml.Node directly to avoid serialization overhead
func (e *CELEvaluator) resourceToMap(resource *yaml.RNode) (map[string]interface{}, error) {
	// Get the underlying yaml.Node
	node := resource.YNode()
	if node == nil {
		return nil, fmt.Errorf("resource has nil yaml.Node")
	}

	// Convert yaml.Node to map[string]interface{} directly
	var result map[string]interface{}
	if err := node.Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode resource: %w", err)
	}

	return result, nil
}
