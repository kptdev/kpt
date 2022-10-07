// Copyright 2022 Google LLC
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

	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"golang.org/x/mod/semver"
)

const revisionRegex = "^v[0-9]+$"

func NextRevisionNumber(revs []PackageRevision) (string, error) {
	// Computes the next revision number as the latest revision number + 1.
	// This function only understands strict versioning format, e.g. v1, v2, etc. It will
	// ignore all revision numbers it finds that do not adhere to this format.
	// If there are no published revisions (in the recognized format), the revision
	// number returned here will be "v1".

	latestRev := "v0"
	for _, current := range revs {

		// Check if the current package revision is more recent than the one seen so far.
		// Only consider Published packages
		if current.Lifecycle() != v1alpha1.PackageRevisionLifecyclePublished {
			continue
		}

		currentRev := current.Key().Revision

		if err := ValidateRevisionNumber(currentRev); err != nil {
			// ignore this revision
			continue
		}

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

func ValidateDescription(descr string) error {
	if len(descr) > 63 || len(descr) == 0 {
		return fmt.Errorf("description %q must be at least 1 and at most 63 characters long", descr)
	}
	if strings.HasPrefix(descr, "-") || strings.HasSuffix(descr, "-") {
		return fmt.Errorf("description %q must start and end with an alphanumeric character", descr)
	}

	match, err := regexp.MatchString(`^[a-zA-Z0-9-]{0,63}$`, descr)
	if err != nil {
		return err
	}
	if !match {
		return fmt.Errorf("description %q must be comprised only of alphanumeric characters and '-'", descr)
	}

	return nil
}

func ValidateRevisionNumber(revision string) error {
	match, err := regexp.MatchString(revisionRegex, revision)
	if err != nil {
		return err
	}
	if !match {
		return fmt.Errorf("input string %s is not a valid revision number", revision)
	}
	return nil
}
