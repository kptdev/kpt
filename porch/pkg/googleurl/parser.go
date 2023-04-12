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

package googleurl

import (
	"fmt"
	"net/url"
	"strings"
)

// Link holds the results of parsing a google selfLink URL.
type Link struct {
	// Service is the service name e.g. container.googleapis.com
	Service string
	// Version is the version in the link
	Version string

	// Project is the GCP project ID in the link
	Project string

	// Zone is the GCP zone found in the link
	Zone string

	// Location is the GCP location (region or zone) found in the link
	Location string

	// Extra holds any additional named fields found in the link
	Extra map[string]string
}

// Parse parses the selfLink into fields in a Link object.
// A selfLink is expected to have a version as the first path token, otherwise use ParseUnversioned.
func Parse(selfLink string) (*Link, error) {
	u, err := url.Parse(selfLink)
	if err != nil {
		return nil, fmt.Errorf("selfLink %q was not a valid URL", selfLink)
	}

	link := &Link{
		Service: u.Host,
	}

	tokens := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")

	link.Version = tokens[0]
	tokens = tokens[1:]

	if len(tokens)%2 != 0 {
		return nil, fmt.Errorf("unexpected url %q (unexpected tokens)", selfLink)
	}

	return parseTokens(link, tokens)
}

func parseTokens(link *Link, tokens []string) (*Link, error) {
	for i := 0; i < len(tokens); i += 2 {
		switch tokens[i] {
		case "projects":
			link.Project = tokens[i+1]
		case "zones":
			link.Zone = tokens[i+1]
		case "locations":
			link.Location = tokens[i+1]
		default:
			if link.Extra == nil {
				link.Extra = make(map[string]string)
			}
			link.Extra[tokens[i]] = tokens[i+1]
		}
	}
	return link, nil
}

// ParseUnversioned parses the unversioned selfLink into fields in a Link object.
// An unversioned reference differs from a normal selfLink in that it doesn't have a version,
// for example "//container.googleapis.com/projects/example-project/locations/us-central1/clusters/example-cluster"
func ParseUnversioned(selfLink string) (*Link, error) {
	u, err := url.Parse(selfLink)
	if err != nil {
		return nil, fmt.Errorf("selfLink %q was not a valid URL", selfLink)
	}

	link := &Link{
		Service: u.Host,
	}

	tokens := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")

	if len(tokens)%2 != 0 {
		return nil, fmt.Errorf("unexpected url %q (unexpected tokens)", selfLink)
	}

	return parseTokens(link, tokens)
}
