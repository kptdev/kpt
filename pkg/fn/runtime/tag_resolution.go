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

package runtime

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/Masterminds/semver/v3"
	regclientref "github.com/regclient/regclient/types/ref"
	"k8s.io/klog/v2"
)

const imageTagError = "start with an alphanumeric character or underscore, followed by at most 127 alphanumeric characters, underscores, periods, or dashes"

var imageTagRegex = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9._-]{0,127}$`)

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
		var allFilteredVersions []*semver.Version
		for _, lister := range tr.Listers {
			possibleTags, err := lister.List(ctx, image)
			if err != nil {
				klog.V(2).Infof("failed to list tags for image %q using lister %q: %v", image, lister.Name(), err)
				continue
			}

			if len(possibleTags) == 0 {
				klog.V(2).Infof("no tags found for image %q with lister %q", image, lister.Name())
				continue
			}

			filteredVersions := filterParseTags(possibleTags, constraint)
			if len(filteredVersions) > 0 {
				allFilteredVersions = append(allFilteredVersions, filteredVersions...)
			} else {
				klog.V(2).Infof("no tag matched the version constraint %q when using lister %q (from %s)", tag, lister.Name(), abbrevSlice(filteredVersions))
			}
		}

		if len(allFilteredVersions) == 0 {
			return "", fmt.Errorf("no tag could be found matching the version constraint %q", tag)
		}

		sortVersions(allFilteredVersions)

		ref.Tag = allFilteredVersions[0].Original()
		return ref.CommonName(), nil
	} else {
		resolvedImage, tagErr := imageWithLiteralTag(ref, tag)
		if tagErr != nil {
			return "", tagErr
		}
		klog.Warningf("Tag %q could not be parsed as a semantic version (\"%s\") or constraint (\"%s\"), will use it literally",
			tag, versionErr, constraintErr)
		return resolvedImage, nil
	}

	return imageWithLiteralTag(ref, tag)
}

func imageWithLiteralTag(ref regclientref.Ref, tag string) (string, error) {
	if !imageTagRegex.MatchString(tag) {
		return "", fmt.Errorf("`function.tag` %q must be a valid image tag: %s", tag, imageTagError)
	}
	ref.Tag = tag
	return ref.CommonName(), nil
}

// filterParseTags takes in a list of potential tags, and returns all the valid semvers matching the constraint
func filterParseTags(tags []string, constraint *semver.Constraints) []*semver.Version {
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

		if constraint.Check(version) {
			versions = append(versions, version)
		} else {
			klog.V(3).Infof("Tag %q did not match constraint %q", tag, constraint.String())
		}
	}

	return versions
}

func sortVersions(versions []*semver.Version) {
	slices.SortFunc(versions, func(a, b *semver.Version) int {
		return b.Compare(a) // we want descending order
	})
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
