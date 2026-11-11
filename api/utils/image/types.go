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

import "strings"

// ParsedImage stores an OCI image name split into all of its components.
type ParsedImage struct {
	// The registry part of the image name without trailing slash.
	// Example: ghcr.io
	Registry string
	// The part of the image name between Registry and BaseName without leading or trailing slashes.
	// Example: kptdev/krm-functions-catalog
	SubPath string
	// The last part of the image name, after all slashes, without the leading slash.
	// Example: apply-setters
	BaseName string
	// The tag of the image without the leading colon.
	// Example: v0.2.3
	Tag string
	// The sha256 digest of the image, without the leading @, but containing the `sha256` prefix.
	// Example: sha256:7d89a74f106241391f687fc2985c8e6de597bb21f0d0014def5edc730618d9cc
	Digest string
	// Original contains the unparsed image name. Intended for testing.
	// Should be the same as the output of Full().
	Original string
}

// Full reconstructs the full image name from the parsed parts.
func (p *ParsedImage) Full() string {
	sb := strings.Builder{}

	if p.Registry != "" {
		sb.WriteString(p.Registry)
		sb.WriteString("/")
	}

	if p.SubPath != "" {
		sb.WriteString(p.SubPath)
		sb.WriteString("/")
	}

	sb.WriteString(p.BaseName)
	if p.Tag != "" {
		sb.WriteString(":")
		sb.WriteString(p.Tag)
	}

	if p.Digest != "" {
		sb.WriteString("@")
		sb.WriteString(p.Digest)
	}

	return sb.String()
}

// Repository is the image name without a tag or digest.
func (p ParsedImage) Repository() string {
	if prefix := p.Prefix(); prefix != "" {
		return Join(prefix, p.BaseName)
	}
	return p.BaseName
}

// Prefix returns the part before BaseName with *no* trailing slash.
func (p *ParsedImage) Prefix() string {
	sb := strings.Builder{}

	if p.Registry != "" {
		sb.WriteString(p.Registry)
		if p.SubPath != "" {
			sb.WriteString("/")
		}
	}
	if p.SubPath != "" {
		sb.WriteString(p.SubPath)
	}

	return sb.String()
}
