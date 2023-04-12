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

package engine

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPkgUpdate(t *testing.T) {
	dfUpdater := &defaultPackageUpdater{}

	testdata, err := filepath.Abs(filepath.Join(".", "testdata", "update"))
	if err != nil {
		t.Fatalf("Failed to find testdata: %v", err)
	}

	localResources, err := loadResourcesFromDirectory(filepath.Join(testdata, "local"))
	if err != nil {
		t.Fatalf("failed to read local resources: %v", err)
	}

	originalResources, err := loadResourcesFromDirectory(filepath.Join(testdata, "original"))
	if err != nil {
		t.Fatalf("failed to read original resources: %v", err)
	}

	upstreamResources, err := loadResourcesFromDirectory(filepath.Join(testdata, "upstream"))
	if err != nil {
		t.Fatalf("failed to read upstream resources: %v", err)
	}

	expectedResources, err := loadResourcesFromDirectory(filepath.Join(testdata, "updated"))
	if err != nil {
		t.Fatalf("failed to read expected updated resources: %v", err)
	}

	updatedResources, err := dfUpdater.Update(context.Background(), localResources, originalResources, upstreamResources)
	if err != nil {
		t.Errorf("unexpected err: %v", err)
	}

	for k, v := range updatedResources.Contents {
		want := expectedResources.Contents[k]
		if diff := cmp.Diff(want, v); diff != "" && k != "Kptfile" {
			// TODO(droot): figure out correct expectation for Kptfile
			t.Errorf("file: %s unexpected result (-want, +got): %s", k, diff)
		}
	}
}
