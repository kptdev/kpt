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

package image

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Masterminds/semver/v3"
)

func MatchesAnyConstraint(version string, constraints []string) bool {
	if len(constraints) == 0 {
		return false
	}

	if version == "" || slices.Contains(constraints, "*") {
		return true
	}

	v, err := semver.NewVersion(version)
	if err != nil {
		return slices.Contains(constraints, version)
	}
	for _, cs := range constraints {
		c, err := semver.NewConstraint(cs)
		if err != nil {
			continue
		}
		if c.Check(v) {
			return true
		}
	}
	return false
}

// FindBestSemverMatch selects the cache key whose semver tag best satisfies
// the constraint for the given imageName. It returns the full cache key
// (e.g. "ghcr.io/foo/bar:v1.2.3") of the highest matching version.
func FindBestSemverMatch(constraint string, cachedTags []string) (string, error) {
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		return "", fmt.Errorf("invalid semver constraint %q: %w", constraint, err)
	}

	type candidate struct {
		key     string
		version *semver.Version
	}

	var matches []candidate
	for _, tag := range cachedTags {
		v, err := semver.NewVersion(tag)
		if err != nil {
			// Failed to parse version
			continue
		}

		if c.Check(v) {
			matches = append(matches, candidate{key: tag, version: v})
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no tag matching constraint %q found among %+v", constraint, cachedTags)
	}

	slices.SortFunc(matches, func(a, b candidate) int {
		return a.version.Compare(b.version)
	})

	selected := matches[len(matches)-1]
	// selected tag

	return selected.key, nil
}

// WithPrefix prepends prefix to a name that contains no slash.
// An empty prefix, or a name that already contains a slash, is returned unchanged.
func WithPrefix(name, prefix string) string {
	if prefix == "" || strings.Contains(name, "/") {
		return name
	}
	return Join(prefix, name)
}

func Join(parts ...string) string {
	if len(parts) == 0 {
		return ""
	}

	sb := strings.Builder{}
	sb.WriteString(strings.Trim(parts[0], "/"))

	for _, part := range parts[1:] {
		part = strings.Trim(part, "/")
		sb.WriteString("/")
		sb.WriteString(part)
	}
	return sb.String()
}

// Parse creates a ParsedImage object from the full image name.
// Does not guarantee that the input string is a valid reference, unlike regclientref.New().
func Parse(fullImageName string) ParsedImage {
	output := ParsedImage{Original: fullImageName}

	firstSlash := strings.Index(fullImageName, "/")
	lastSlash := strings.LastIndex(fullImageName, "/")

	if firstSlash != -1 {
		str := fullImageName[:firstSlash]
		if registryRE.MatchString(str) {
			output.Registry = strings.TrimRight(str, "/")
			fullImageName = fullImageName[firstSlash+1:]

			lastSlash = strings.LastIndex(fullImageName, "/")
			if lastSlash != -1 {
				output.SubPath = strings.Trim(fullImageName[:lastSlash], "/")
				fullImageName = fullImageName[lastSlash+1:]
			}
		} else {
			output.SubPath = strings.Trim(fullImageName[:lastSlash], "/")
			fullImageName = fullImageName[lastSlash+1:]
		}
	}

	if lastAt := strings.LastIndex(fullImageName, "@"); lastAt != -1 {
		output.Digest = fullImageName[lastAt+1:]
		fullImageName = fullImageName[:lastAt]
	}

	if lastColon := strings.LastIndex(fullImageName, ":"); lastColon != -1 {
		output.Tag = fullImageName[lastColon+1:]
		fullImageName = fullImageName[:lastColon]
	}

	output.BaseName = strings.TrimLeft(fullImageName, "/")
	return output
}
