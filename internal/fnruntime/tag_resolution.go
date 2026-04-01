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
	regclientref "github.com/regclient/regclient/types/ref"
	"k8s.io/klog/v2"
)

// TagLister is an interface for listing tags for/from a function runtime/runner
type TagLister interface {
	Name() string
	List(ctx context.Context, image string) ([]string, error)
}

type TagResolver struct {
	// Listers is a slice of TagListers that are checked in order for a matching tag.
	Listers []TagLister
}

// ResolveFunctionImage substitutes the `function.image` with the latest tag matching the constraint in `function.tag`.
// No-op if `function.tag` is empty. If `function.tag` is non-empty the tag will *always* be overridden in `function.image`.
func (tr *TagResolver) ResolveFunctionImage(ctx context.Context, image, tag string) (string, error) {
	if tag == "" {
		return image, nil
	}

	ref, err := regclientref.New(image)
	if err != nil {
		return "", fmt.Errorf("failed to parse image %q as reference: %w", image, err)
	}
	ref.Tag = ""
	ref.Digest = ""
	image = ref.CommonName()

	if _, versionErr := semver.NewVersion(tag); versionErr == nil { //nolint:revive
		// A valid version is a valid constraint, but we don't want to waste time listing
		// when we are given an exact version. We just return from here.
	} else if constraint, constraintErr := semver.NewConstraint(tag); constraintErr == nil {
		for _, lister := range tr.Listers {
			possibleTags, err := lister.List(ctx, image)
			if err != nil {
				klog.Errorf("failed to list tags for image %q using lister %q: %v", image, lister.Name(), err)
				continue
			}

			if len(possibleTags) == 0 {
				klog.Infof("no tags found for image %q with lister %q", image, lister.Name())
				continue
			}

			filteredVersions := filterParseSortTags(possibleTags)
			for _, version := range filteredVersions {
				if constraint.Check(version) {
					ref.Tag = version.Original()
					return ref.CommonName(), nil
				}
			}

			klog.Infof("no tag matched the version constraint %q when using lister %q (from %s)", tag, lister.Name(), abbrevSlice(filteredVersions))
		}

		return "", fmt.Errorf("no tag could be found matching the version constraint %q", tag)
	} else {
		klog.Warningf("Tag %q could not be parsed as a semantic version (\"%s\") or constraint (\"%s\"), will use it literally",
			tag, versionErr, constraintErr)
	}

	ref.Tag = tag
	return ref.CommonName(), nil
}

// filterParseSortTags takes in a list of potential tags, and returns all the valid semvers in descending order
func filterParseSortTags(tags []string) []*semver.Version {
	var versions []*semver.Version
	for _, tag := range tags {
		if strings.HasPrefix(tag, "sha256-") {
			klog.V(3).Infof("Skipping tag %q because it looks like a hash", tag)
			continue
		}

		if strings.Contains(tag, "-git-") {
			klog.V(3).Infof("Skipping tag %q because it looks like a git-based build tag", tag)
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

func abbrevSlice(slice []*semver.Version) string {
	switch len(slice) {
	case 0:
		return "[]"
	case 1, 2, 3:
		out := make([]string, len(slice))
		for i, v := range slice {
			out[i] = v.Original()
		}
		return "[" + strings.Join(out, ", ") + "]"
	default:
		return fmt.Sprintf("[%s, %s, ..., %s]",
			slice[0].Original(), slice[1].Original(), slice[len(slice)-1].Original())
	}
}
