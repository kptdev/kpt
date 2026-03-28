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

package runneroptions

import (
	"context"
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/ext"
	k8scellib "k8s.io/apiserver/pkg/cel/library"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const celCheckFrequency = 100

// celCostLimit gives about .1 seconds of CPU time for the evaluation to run
const celCostLimit = 1000000

// CELEnvironment holds a shared CEL environment for evaluating conditions.
// The environment is created once and reused; programs are compiled per condition call.
type CELEnvironment struct {
	env *cel.Env
}

// NewCELEnvironment creates a new CELEnvironment with the standard KRM variable bindings.
// Includes cel-go built-in extensions and k8s-specific validators (IP, CIDR, Quantity, SemVer)
// from k8s.io/apiserver/pkg/cel/library for full Kubernetes CEL compatibility.
func NewCELEnvironment() (*CELEnvironment, error) {
	env, err := cel.NewEnv(
		cel.Variable("resources", cel.ListType(cel.DynType)),
		cel.HomogeneousAggregateLiterals(),
		cel.DefaultUTCTimeZone(true),
		cel.CrossTypeNumericComparisons(true),
		cel.OptionalTypes(),
		ext.Strings(ext.StringsVersion(2)),
		ext.Sets(),
		ext.TwoVarComprehensions(),
		ext.Lists(ext.ListsVersion(3)),
		k8scellib.IP(),
		k8scellib.CIDR(),
		k8scellib.Quantity(),
		k8scellib.SemverLib(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}
	return &CELEnvironment{env: env}, nil
}

// EvaluateCondition compiles and evaluates a CEL condition against a list of KRM resources.
// Returns true if the condition is met, false otherwise.
// An empty condition always returns true (function executes unconditionally).
func (e *CELEnvironment) EvaluateCondition(ctx context.Context, condition string, resources []*yaml.RNode) (bool, error) {
	if condition == "" {
		return true, nil
	}

	ast, issues := e.env.Compile(condition)
	if issues != nil && issues.Err() != nil {
		return false, fmt.Errorf("failed to compile CEL expression: %w", issues.Err())
	}

	if ast.OutputType() != cel.BoolType {
		return false, fmt.Errorf("CEL expression must return a boolean, got %v", ast.OutputType())
	}

	prg, err := e.env.Program(ast,
		cel.CostLimit(celCostLimit),
		cel.InterruptCheckFrequency(celCheckFrequency),
	)
	if err != nil {
		return false, fmt.Errorf("failed to create CEL program: %w", err)
	}

	resourceList, err := resourcesToList(resources)
	if err != nil {
		return false, fmt.Errorf("failed to convert resources: %w", err)
	}

	out, _, err := prg.ContextEval(ctx, map[string]interface{}{
		"resources": resourceList,
	})
	if err != nil {
		return false, fmt.Errorf("failed to evaluate CEL expression: %w", err)
	}

	result, ok := out.(types.Bool)
	if !ok {
		return false, fmt.Errorf("CEL expression must return a boolean, got %T", out)
	}

	return bool(result), nil
}

func resourcesToList(resources []*yaml.RNode) ([]interface{}, error) {
	result := make([]interface{}, 0, len(resources))
	for _, resource := range resources {
		m, err := resourceToMap(resource)
		if err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, nil
}

func resourceToMap(resource *yaml.RNode) (map[string]interface{}, error) {
	node := resource.YNode()
	if node == nil {
		return nil, fmt.Errorf("resource has nil yaml.Node")
	}
	var result map[string]interface{}
	if err := node.Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode resource: %w", err)
	}
	// Ensure standard KRM fields are always present so CEL expressions like
	// r.kind == "Deployment" never error with "no such key".
	if _, ok := result["apiVersion"]; !ok {
		result["apiVersion"] = ""
	}
	if _, ok := result["kind"]; !ok {
		result["kind"] = ""
	}
	if _, ok := result["metadata"]; !ok {
		result["metadata"] = map[string]interface{}{}
	}
	return result, nil
}
