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

package repository

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	kptfile "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	api "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"golang.org/x/mod/semver"
)

func ToApiReadinessGates(kf kptfile.KptFile) []api.ReadinessGate {
	var readinessGates []api.ReadinessGate
	if kf.Info != nil {
		for _, rg := range kf.Info.ReadinessGates {
			readinessGates = append(readinessGates, api.ReadinessGate{
				ConditionType: rg.ConditionType,
			})
		}
	}
	return readinessGates
}

func ToApiConditions(kf kptfile.KptFile) []api.Condition {
	var conditions []api.Condition
	if kf.Status != nil && kf.Status.Conditions != nil {
		for _, s := range kf.Status.Conditions {
			conditions = append(conditions, api.Condition{
				Type:    s.Type,
				Status:  toApiConditionStatus(s.Status),
				Reason:  s.Reason,
				Message: s.Message,
			})
		}
	}
	return conditions
}

func toApiConditionStatus(s kptfile.ConditionStatus) api.ConditionStatus {
	switch s {
	case kptfile.ConditionTrue:
		return api.ConditionTrue
	case kptfile.ConditionFalse:
		return api.ConditionFalse
	case kptfile.ConditionUnknown:
		return api.ConditionUnknown
	default:
		panic(fmt.Errorf("unknown condition status: %v", s))
	}
}

func NextRevisionNumber(revs []string) (string, error) {
	// Computes the next revision number as the latest revision number + 1.
	// This function only understands strict versioning format, e.g. v1, v2, etc. It will
	// ignore all revision numbers it finds that do not adhere to this format.
	// If there are no published revisions (in the recognized format), the revision
	// number returned here will be "v1".
	latestRev := "v0"
	for _, currentRev := range revs {
		if !semver.IsValid(currentRev) {
			// ignore this revision
			continue
		}
		// collect the major version. i.e. if we find that the latest published
		// version is v3.1.1, we will end up returning v4
		currentRev = semver.Major(currentRev)

		switch cmp := semver.Compare(currentRev, latestRev); {
		case cmp == 0:
			// Same revision.
		case cmp < 0:
			// current < latest; no change
		case cmp > 0:
			// current > latest; update latest
			latestRev = currentRev
		}

	}

	i, err := strconv.Atoi(latestRev[1:])
	if err != nil {
		return "", err
	}
	i++
	next := "v" + strconv.Itoa(i)
	return next, nil
}

// ValidateWorkspaceName validates WorkspaceName. It must:
//   - be at least 1 and at most 63 characters long
//   - contain only lowercase alphanumeric characters or '-'
//   - start and end with an alphanumeric character.
//
// '/ ' should never be allowed, because we use '/' to
// delimit branch names (e.g. the 'drafts/' prefix).
func ValidateWorkspaceName(workspace v1alpha1.WorkspaceName) error {
	wn := string(workspace)
	if len(wn) > 63 || len(wn) == 0 {
		return fmt.Errorf("workspaceName %q must be at least 1 and at most 63 characters long", wn)
	}
	if strings.HasPrefix(wn, "-") || strings.HasSuffix(wn, "-") {
		return fmt.Errorf("workspaceName %q must start and end with an alphanumeric character", wn)
	}

	match, err := regexp.MatchString(`^[a-z0-9-]+$`, wn)
	if err != nil {
		return err
	}
	if !match {
		return fmt.Errorf("workspaceName %q must be comprised only of lowercase alphanumeric characters and '-'", wn)
	}

	return nil
}
