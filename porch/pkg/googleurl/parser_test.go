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
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParse(t *testing.T) {
	grid := []struct {
		SelfLink string
		Want     Link
	}{
		{
			SelfLink: "https://container.googleapis.com/v1/projects/justinsb-acp-dev/locations/us-central1/clusters/auto-gke/nodePools/default-pool",
			Want: Link{
				Service:  "container.googleapis.com",
				Version:  "v1",
				Project:  "justinsb-acp-dev",
				Location: "us-central1",
				Extra: map[string]string{
					"clusters":  "auto-gke",
					"nodePools": "default-pool",
				},
			},
		},
		{
			SelfLink: "//gkehub.googleapis.com/v1/projects/justinsb-acp-dev/locations/global/memberships/example-cluster-1",
			Want: Link{
				Service:  "gkehub.googleapis.com",
				Version:  "v1",
				Project:  "justinsb-acp-dev",
				Location: "global",
				Extra: map[string]string{
					"memberships": "example-cluster-1",
				},
			},
		},
	}

	for _, g := range grid {
		t.Run(g.SelfLink, func(t *testing.T) {
			got, err := Parse(g.SelfLink)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(g.Want, *got); diff != "" {
				t.Errorf("unexpected diff (+got,-want): %s", diff)
			}
		})
	}
}

func TestParseUnversioned(t *testing.T) {
	grid := []struct {
		SelfLink string
		Want     Link
	}{
		{
			SelfLink: "//container.googleapis.com/projects/example-project/locations/us-central1/clusters/example-cluster",
			Want: Link{
				Service:  "container.googleapis.com",
				Project:  "example-project",
				Location: "us-central1",
				Extra: map[string]string{
					"clusters": "example-cluster",
				},
			},
		},
	}

	for _, g := range grid {
		t.Run(g.SelfLink, func(t *testing.T) {
			got, err := ParseUnversioned(g.SelfLink)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(g.Want, *got); diff != "" {
				t.Errorf("unexpected diff (+got,-want): %s", diff)
			}
		})
	}
}
