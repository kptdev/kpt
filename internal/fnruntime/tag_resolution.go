// Copyright 2026 The kpt Authors
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
	"slices"
	"strings"

	"github.com/Masterminds/semver/v3"
	"k8s.io/klog/v2"
)

// TagLister is an interface for listing tags for/from a function runtime/runner
type TagLister interface {
	List(ctx context.Context, image string) ([]string, error)
}

type TagResolver struct {
	lister TagLister
}

// ResolveFunctionImage substitutes the `function.image` with the latest tag matching the constraint in `function.tag`.
// No-op if `function.tag` is empty. If `function.tag` is non-empty the tag will *always* be overridden in `function.image`.
func (tr *TagResolver) ResolveFunctionImage(ctx context.Context, image, tag string) (string, error) {
	if tag == "" {
		return image, nil
	}

	image = strings.Split(image, ":")[0]
	if _, versionErr := semver.NewVersion(tag); versionErr == nil {
		return fmt.Sprintf("%s:%s", image, tag), nil
	} else if constraint, constraintErr := semver.NewConstraint(tag); constraintErr == nil {
		possibleTags, err := tr.lister.List(ctx, image)
		if err != nil {
			return "", fmt.Errorf("failed to list tags for image %q: %w", image, err)
		}

		for _, possibleVersion := range filterParseSortTags(possibleTags) {
			if constraint.Check(possibleVersion) {
				return fmt.Sprintf("%s:%s", image, possibleVersion.Original()), nil
			}
		}

		return "", fmt.Errorf("no remote tag matched the version constraint %q from %v", tag, possibleTags)
	} else {
		klog.Warningf("Tag %q could not be parsed as a semantic version (\"%s\") or constraint (\"%s\"), will use it literally",
			tag, versionErr, constraintErr)
		return fmt.Sprintf("%s:%s", image, tag), nil
	}
}

// filterParseSortTags takes in a list of potential tags, and returns all the valid semvers in descending order
func filterParseSortTags(tags []string) []*semver.Version {
	var versions []*semver.Version
	for _, tag := range tags {
		if strings.HasPrefix(tag, "sha256-") {
			klog.V(3).Infof("Skipping tag %q because it looks like a hash", tag)
			continue
		}

		if strings.HasPrefix(tag, "master-git-") {
			klog.V(3).Infof("Skipping tag %q", tag)
			continue
		}

		version, err := semver.NewVersion(tag)
		if err != nil {
			klog.V(3).Infof("Failed to parse tag %q as semantic version, ignoring", tag)
			continue
		}

		versions = append(versions, version)
	}

	slices.SortFunc(versions, func(a, b *semver.Version) int {
		return b.Compare(a) // we want descending order
	})

	return versions
}
