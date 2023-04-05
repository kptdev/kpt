// Copyright 2022 The kpt Authors
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

package rootsyncset

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

// checkSyncStatus fetches the RootSync using the provided client and computes the sync status. The rules
// for computing status here mirrors the one used in the status command in the nomos cli.
func checkSyncStatus(ctx context.Context, client dynamic.Interface, rssName string) (string, error) {
	// TODO: Change this to use the RootSync type instead of Unstructured.
	rs, err := client.Resource(rootSyncGVR).Namespace(rootSyncNamespace).Get(ctx, rssName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get RootSync: %w", err)
	}

	generation, _, err := unstructured.NestedInt64(rs.Object, "metadata", "generation")
	if err != nil {
		return "", fmt.Errorf("failed to read generation from RootSync: %w", err)
	}

	observedGeneration, _, err := unstructured.NestedInt64(rs.Object, "status", "observedGeneration")
	if err != nil {
		return "", fmt.Errorf("failed to read observedGeneration from RootSync: %w", err)
	}

	if generation != observedGeneration {
		return "Pending", nil
	}

	conditions, _, err := unstructured.NestedSlice(rs.Object, "status", "conditions")
	if err != nil {
		return "", fmt.Errorf("failed to extract conditions from RootSync: %w", err)
	}

	val, found, err := getConditionStatus(conditions, "Stalled")
	if err != nil {
		return "", fmt.Errorf("error fetching condition 'Stalled' from conditions slice: %w", err)
	}
	if found && val == "True" {
		return "Stalled", nil
	}

	val, found, err = getConditionStatus(conditions, "Reconciling")
	if err != nil {
		return "", fmt.Errorf("error fetching condition 'Reconciling' from conditions slice: %w", err)
	}
	if found && val == "True" {
		return "Reconciling", nil
	}

	cond, found, err := getCondition(conditions, "Syncing")
	if err != nil {
		return "", fmt.Errorf("error fetching condition 'Syncing' from conditions slice: %w", err)
	}
	if !found {
		return "Reconciling", nil
	}

	errCount, err := extractErrorCount(cond)
	if err != nil {
		return "", fmt.Errorf("error extracting error count from 'Syncing' condition: %w", err)
	}
	if errCount > 0 {
		return "Error", nil
	}

	val, err = extractStringField(cond, "status")
	if err != nil {
		return "", fmt.Errorf("error extracting status of 'Syncing' condition: %w", err)
	}
	if val == "True" {
		return "Pending", nil
	}

	return "Synced", nil
}

func getConditionStatus(conditions []interface{}, condType string) (string, bool, error) {
	cond, found, err := getCondition(conditions, condType)
	if err != nil {
		return "", false, err
	}
	if !found {
		return "", false, nil
	}
	s, err := extractStringField(cond, "status")
	if err != nil {
		return "", false, err
	}
	return s, true, nil
}

func getCondition(conditions []interface{}, condType string) (map[string]interface{}, bool, error) {
	for i := range conditions {
		cond, ok := conditions[i].(map[string]interface{})
		if !ok {
			return map[string]interface{}{}, false, fmt.Errorf("failed to extract condition %d from slice", i)
		}
		t, err := extractStringField(cond, "type")
		if err != nil {
			return map[string]interface{}{}, false, err
		}

		if t != condType {
			continue
		}
		return cond, true, nil
	}
	return map[string]interface{}{}, false, nil
}

func extractStringField(condition map[string]interface{}, field string) (string, error) {
	t, ok := condition[field]
	if !ok {
		return "", fmt.Errorf("condition does not have a type field")
	}
	condVal, ok := t.(string)
	if !ok {
		return "", fmt.Errorf("value of '%s' condition is not of type 'string'", field)
	}
	return condVal, nil
}

func extractErrorCount(cond map[string]interface{}) (int64, error) {
	count, found, err := unstructured.NestedInt64(cond, "errorSummary", "totalCount")
	if err != nil || !found {
		return 0, err
	}
	return count, nil
}
